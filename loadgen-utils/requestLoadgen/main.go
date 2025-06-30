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
)

type ConfigParams struct {
	TargetURL   string `firestore:"targetUrl"`
	TargetCPU   int    `firestore:"targetCpu,omitempty"`
	QPS         int    `firestore:"qps,omitempty"`
	Duration    int    `firestore:"duration,omitempty"`
	Active      bool   `firestore:"active"`
	FirestoreID string `firestore:"-"`
}

const projectIDEnv = "GOOGLE_CLOUD_PROJECT"
const collectionName = "loadgen-configs"

var (
	firestoreClient *firestore.Client
)

func main() {
	ctx := context.Background()
	projectID := os.Getenv(projectIDEnv)
	if projectID == "" {
		projectID = "mslarkin-ext"
	}

	var err error
	firestoreClient, err = firestore.NewClientWithDatabase(ctx, projectID, "loadgen-target-config")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	log.Println("RequestLoadgen service started. Reading configurations from Firestore...")

	configs := make(map[string]ConfigParams)
	stopChans := make(map[string]chan struct{})
	var wg sync.WaitGroup

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		log.Println("Reading configurations from Firestore...")
		newConfigs, err := readConfigs(ctx)
		if err != nil {
			log.Printf("Error reading configs from Firestore: %v", err)
			continue
		}

		newConfigMap := make(map[string]ConfigParams)
		for _, config := range newConfigs {
			newConfigMap[config.FirestoreID] = config
		}

		// Stop goroutines for configs that have been removed or deactivated
		for id := range configs {
			if newConfig, ok := newConfigMap[id]; !ok || !newConfig.Active {
				log.Printf("Stopping load generation for config %s", id)
				close(stopChans[id])
				delete(stopChans, id)
			}
		}

		// Start or update goroutines for new or existing active configs
		for id, config := range newConfigMap {
			if !config.Active {
				continue
			}
			if _, ok := configs[id]; !ok {
				log.Printf("Starting load generation for TargetURL: %s", config.TargetURL)
				stopChan := make(chan struct{})
				stopChans[id] = stopChan
				wg.Add(1)
				go func(cfg ConfigParams, stop <-chan struct{}) {
					defer wg.Done()
					generateLoad(cfg, stop)
				}(config, stopChan)
			}
		}

		configs = newConfigMap
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

func generateLoad(config ConfigParams, stop <-chan struct{}) {
	target, err := url.Parse(config.TargetURL)
	if err != nil {
		log.Printf("[%s] Invalid Target URL %s: %v", config.FirestoreID, config.TargetURL, err)
		return
	}

	query := target.Query()
	if config.TargetCPU > 0 {
		query.Set("targetCpuPct", strconv.Itoa(config.TargetCPU))
	}
	target.RawQuery = query.Encode()
	finalURL := target.String()

	log.Printf("[%s] Starting requests to: %s (QPS: %d, Duration: %ds)",
		config.FirestoreID, finalURL, config.QPS, config.Duration)

	if config.QPS <= 0 {
		log.Printf("[%s] QPS is %d, must be positive. Skipping.", config.FirestoreID, config.QPS)
		return
	}

	interval := time.Second / time.Duration(config.QPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var durationTimer <-chan time.Time
	if config.Duration != -1 {
		durationTimer = time.NewTimer(time.Duration(config.Duration) * time.Second).C
	}

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
			}()
			requestCount++
		case <-durationTimer:
			log.Printf("[%s] Duration of %d seconds reached for %s. Sent %d requests.",
				config.FirestoreID, config.Duration, finalURL, requestCount)
			queryAndLogMetrics(config, finalURL, projectIDEnv, requestCount)
			return
		case <-stop:
			log.Printf("[%s] Stopping load generation for %s.", config.FirestoreID, finalURL)
			return
		}
	}
}

func queryAndLogMetrics(config ConfigParams, actualURL string, projectID string, requestsSentByThisInstance int) {
	calculatedQPS := float64(requestsSentByThisInstance) / float64(config.Duration)

	log.Printf("METRICS [%s]: Target URL: %s, Configured QPS: %d, Actual QPS (calculated by this instance): %.2f",
		config.FirestoreID, actualURL, config.QPS, calculatedQPS)
}
