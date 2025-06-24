package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	// monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	// monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

// ConfigParams holds the configuration parameters read from Firestore.
type ConfigParams struct {
	TargetURL     string `firestore:"targetUrl"`
	TargetCPU     int    `firestore:"targetCpu,omitempty"`
	QPS           int    `firestore:"qps,omitempty"`
	Duration      int    `firestore:"duration,omitempty"`
	FirestoreID   string `firestore:"-"` // Document ID
}

const projectIDEnv = "GOOGLE_CLOUD_PROJECT"
const collectionName = "loadgen-configs" // Same collection as used by loadgenConfig

var (
	firestoreClient *firestore.Client
	// monitoringClient *monitoring.MetricClient // To be used later
)

func main() {
	ctx := context.Background()
	projectID := os.Getenv(projectIDEnv)
	if projectID == "" {
		log.Fatalf("Environment variable %s must be set.", projectIDEnv)
	}

	var err error
	firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	// Initialize Monitoring client (placeholder for now)
	// monitoringClient, err = monitoring.NewMetricClient(ctx)
	// if err != nil {
	// 	log.Fatalf("Failed to create monitoring client: %v", err)
	// }
	// defer monitoringClient.Close()

	log.Println("RequestLoadgen service started. Reading configurations from Firestore...")

	configs, err := readConfigs(ctx)
	if err != nil {
		log.Fatalf("Failed to read configs from Firestore: %v", err)
	}

	if len(configs) == 0 {
		log.Println("No configurations found in Firestore. Exiting.")
		return
	}

	var wg sync.WaitGroup
	for _, config := range configs {
		wg.Add(1)
		go func(cfg ConfigParams) {
			defer wg.Done()
			log.Printf("Starting load generation for ID %s: TargetURL: %s, QPS: %d, Duration: %ds, CPU: %d%%",
				cfg.FirestoreID, cfg.TargetURL, cfg.QPS, cfg.Duration, cfg.TargetCPU)
			generateLoad(cfg)
		}(config)
	}

	wg.Wait()
	log.Println("All load generation tasks completed.")
}

func readConfigs(ctx context.Context) ([]ConfigParams, error) {
	var configs []ConfigParams
	iter := firestoreClient.Collection(collectionName).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating documents: %w", err)
		}
		var config ConfigParams
		if err := doc.DataTo(&config); err != nil {
			log.Printf("Warning: Failed to parse document %s: %v. Skipping.", doc.Ref.ID, err)
			continue
		}
		config.FirestoreID = doc.Ref.ID

		// Apply defaults if not set (matching loadgenConfig service)
		if config.QPS == 0 {
			config.QPS = 1
		}
		if config.Duration == 0 {
			config.Duration = 1
		}

		configs = append(configs, config)
	}
	return configs, nil
}

func generateLoad(config ConfigParams) {
	target, err := url.Parse(config.TargetURL)
	if err != nil {
		log.Printf("[%s] Invalid Target URL %s: %v", config.FirestoreID, config.TargetURL, err)
		return
	}

	query := target.Query()
	if config.TargetCPU > 0 {
		query.Set("targetCpuPct", strconv.Itoa(config.TargetCPU))
	}
	// Duration is used for how long this function runs, but also appended as a query param
	// as per the original request.
	if config.Duration > 0 {
		query.Set("durationS", strconv.Itoa(config.Duration))
	}
	target.RawQuery = query.Encode()
	finalURL := target.String()

	log.Printf("[%s] Starting requests to: %s (QPS: %d, Duration: %ds)",
		config.FirestoreID, finalURL, config.QPS, config.Duration)

	if config.QPS <= 0 {
		log.Printf("[%s] QPS is %d, must be positive. Skipping.", config.FirestoreID, config.QPS)
		return
	}
	if config.Duration <= 0 {
		log.Printf("[%s] Duration is %d, must be positive. Skipping.", config.FirestoreID, config.Duration)
		return
	}

	interval := time.Second / time.Duration(config.QPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	durationTimer := time.NewTimer(time.Duration(config.Duration) * time.Second)
	defer durationTimer.Stop()

	client := &http.Client{Timeout: 10 * time.Second}
	requestCount := 0

	for {
		select {
		case <-ticker.C:
			go func() {
				resp, err := client.Get(finalURL)
				if err != nil {
					log.Printf("[%s] Error sending GET request to %s: %v", config.FirestoreID, finalURL, err)
					return
				}
				defer resp.Body.Close()
				// Log basic response info, can be expanded
				// log.Printf("[%s] Request to %s - Status: %s", config.FirestoreID, finalURL, resp.Status)
			}()
			requestCount++
		case <-durationTimer.C:
			log.Printf("[%s] Duration of %d seconds reached for %s. Sent %d requests.",
				config.FirestoreID, config.Duration, finalURL, requestCount)
			// This is where we would query Cloud Monitoring
			queryAndLogMetrics(config, finalURL, projectIDEnv, requestCount)
			return
		}
	}
}

// queryAndLogMetrics is a placeholder for Cloud Monitoring integration.
// For now, it will log the configured QPS and a calculated actual QPS based on requests sent by this instance.
// The `run.googleapis.com/request_count` metric would give a more authoritative value if the target is a Cloud Run service.
func queryAndLogMetrics(config ConfigParams, actualURL string, projectID string, requestsSentByThisInstance int) {
	// Placeholder: In a real scenario, you'd query Cloud Monitoring API.
	// This requires:
	// 1. Identifying the Cloud Run service (service_id, region) from the targetURL or additional config.
	// 2. Using the monitoring client to query "run.googleapis.com/request_count".
	// 3. Filtering by revision_name, service_name, etc.
	// 4. Calculating QPS based on the change in request_count over a time window.

	calculatedQPS := float64(requestsSentByThisInstance) / float64(config.Duration)

	log.Printf("METRICS [%s]: Target URL: %s, Configured QPS: %d, Actual QPS (calculated by this instance): %.2f",
		config.FirestoreID, actualURL, config.QPS, calculatedQPS)

	// Example of what would be logged if we had actual monitoring data:
	// log.Printf("METRICS [%s]: Target URL: %s, Configured QPS: %d, Actual QPS (from Cloud Monitoring): %.2f",
	//    config.FirestoreID, actualURL, config.QPS, actualMonitoringQPS)
}
