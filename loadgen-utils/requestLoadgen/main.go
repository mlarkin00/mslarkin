package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// ConfigParams defines the structure for load generation configuration.
// These parameters are read from a Firestore collection.
type ConfigParams struct {
	// TargetURL is the URL to which the load generation will send requests.
	TargetURL string `firestore:"targetUrl"`
	// TargetCPU is the target CPU utilization percentage for the target service.
	// This is an optional parameter that can be passed as a query parameter to the target URL.
	TargetCPU int `firestore:"targetCpu,omitempty"`
	// QPS is the number of queries per second to generate.
	QPS int `firestore:"qps,omitempty"`
	// Duration is the length of time in seconds for which to generate load.
	Duration int `firestore:"duration,omitempty"`
	// Active indicates whether this configuration is currently active and should be used for load generation.
	Active bool `firestore:"active"`
	// FirestoreID is the unique identifier of the document in Firestore.
	// It is not stored in Firestore but is populated when the config is read.
	FirestoreID string `firestore:"-"`
}

// collectionName is the name of the Firestore collection where load generation configurations are stored.
const collectionName = "loadgen-configs"

var projectID string

var (
	// firestoreClient is the global client for interacting with Firestore.
	firestoreClient *firestore.Client
)

// Create channel to listen for signals.
var signalChan chan (os.Signal) = make(chan os.Signal, 1)

// main is the entry point of the application.
// It initializes the Firestore client, and then enters a loop to periodically
// read load generation configurations and manage the load generation goroutines.
func main() {
	// Create a background context.
	ctx := context.Background()
	// Get the project ID from the environment variable, with a default value.
	projectID = os.Getenv("PROJECT_ID")
	if projectID == "" {
		projectID = "mslarkin-ext"
	}
	pollRateS := os.Getenv("POLL_RATE_S")
	pollRate := 30 // Default poll rate is 30 seconds
	if pollRateS != "" {
		pollRate, _ = strconv.Atoi(os.Getenv("POLL_RATE_S"))
	}

	// SIGINT handles Ctrl+C locally.
	// SIGTERM handles Cloud Run termination signal.
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize the Firestore client.
	var err error
	firestoreClient, err = firestore.NewClientWithDatabase(ctx, projectID, "loadgen-target-config")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	// Defer closing the client until the function returns.
	defer firestoreClient.Close()

	log.Println("RequestLoadgen service started. Reading configurations from Firestore...")

	// configs stores the current set of active load generation configurations.
	configs := make(map[string]ConfigParams)
	// stopChans holds channels that can be used to stop the corresponding goroutines.
	stopChans := make(map[string]chan struct{})
	// wg is a WaitGroup to wait for all goroutines to finish before exiting.
	var wg sync.WaitGroup

	loadCtx, loadCtxCancel := context.WithCancel(context.Background())
	defer loadCtx.Done()
	// Start load generation in a goroutine to allow for signal handling and graceful shutdown.
	go func(ctx context.Context) {
		// Create a ticker to periodically re-read the configurations from Firestore.
		ticker := time.NewTicker(time.Duration(pollRate) * time.Second)
		defer ticker.Stop()

		// Main loop to manage load generation based on Firestore configurations.
		for ; true; <-ticker.C {
			// log.Println("Reading configurations from Firestore...")
			// Read the latest configurations from Firestore.
			newConfigs, err := readConfigs(ctx)
			if err != nil {
				log.Printf("Error reading configs from Firestore: %v", err)
				continue
			}

			// Create a map of the new configurations for easy lookup.
			newConfigMap := make(map[string]ConfigParams)
			for _, config := range newConfigs {
				newConfigMap[config.FirestoreID] = config
			}

			// Stop goroutines for configurations that have been removed or deactivated.
			for id := range configs {
				//Check if the config was in the old set but not the new set
				if newConfig, ok := newConfigMap[id]; !ok || !newConfig.Active {
					//If the config was removed (in old, not new) and is running, stop it.
					if _, exists := stopChans[id]; exists {
						log.Printf("[%s] Stopping load generation: %s", id, newConfig.TargetURL)
						close(stopChans[id])
						delete(stopChans, id)
					}
				}
			}

			// Start or update goroutines for new or existing active configs.
			for id, config := range newConfigMap {
				// Skip inactive configurations.
				if !config.Active {
					continue
				}
				// If the configuration isn't running, start a new goroutine for it.
				if _, exists := stopChans[id]; !exists {
					// log.Printf("Starting load generation for TargetURL: %s", config.TargetURL)
					stopChan := make(chan struct{})
					stopChans[id] = stopChan
					wg.Add(1)
					go func(cfg ConfigParams, stop <-chan struct{}) {
						defer wg.Done()
						generateLoad(cfg, stop)
					}(config, stopChan)
				}
			}

			// Update the current set of configurations.
			configs = newConfigMap
		}

		// Wait for all load generation goroutines to complete.
		wg.Wait()
		log.Println("All load generation tasks completed.")
	}(loadCtx)
	// Wait for a termination signal.
	sig := <-signalChan
	log.Printf("%s signal caught", sig)
	// Close all stop channels to signal goroutines to stop.
	for id, stopChan := range stopChans {
		log.Printf("Stopping load generation for config %s", id)
		close(stopChan)
	}
	// Wait for all goroutines to finish.
	wg.Wait()
	// Cancel the load generation context to stop all goroutines.
	loadCtxCancel()
	log.Println("RequestLoadgen service stopped gracefully.")
}

// readConfigs reads all documents from the configured Firestore collection
// and returns them as a slice of ConfigParams.
func readConfigs(ctx context.Context) ([]ConfigParams, error) {
	var configs []ConfigParams
	// Get an iterator for all documents in the collection.
	iter := firestoreClient.Collection(collectionName).Documents(ctx)
	defer iter.Stop()
	// Iterate over the documents.
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating documents: %w", err)
		}
		// Decode the document into a ConfigParams struct.
		var config ConfigParams
		if err := doc.DataTo(&config); err != nil {
			log.Printf("Warning: Failed to parse document %s: %v. Skipping.", doc.Ref.ID, err)
			continue
		}
		// Set the FirestoreID from the document reference.
		config.FirestoreID = doc.Ref.ID

		// Set default values for QPS and Duration if they are not provided.
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

// generateLoad generates HTTP requests to a target URL based on the provided configuration.
// It runs until the duration is reached or a stop signal is received.
func generateLoad(config ConfigParams, stop <-chan struct{}) {
	// Parse the target URL.
	target, err := url.Parse(config.TargetURL)
	if err != nil {
		log.Printf("[%s] Invalid Target URL %s: %v", config.FirestoreID, config.TargetURL, err)
		return
	}

	// Add the target CPU as a query parameter if it is specified.
	query := target.Query()
	if config.TargetCPU > 0 {
		query.Set("targetCpuPct", strconv.Itoa(config.TargetCPU))
	}
	target.RawQuery = query.Encode()
	finalURL := target.String()

	log.Printf("[%s] Starting requests to: %s (QPS: %d, Duration: %ds)",
		config.FirestoreID, finalURL, config.QPS, config.Duration)

	// Ensure QPS is a positive number.
	if config.QPS <= 0 {
		log.Printf("[%s] QPS is %d, must be positive. Skipping.", config.FirestoreID, config.QPS)
		return
	}

	// Create a ticker to control the request rate (QPS).
	interval := time.Second / time.Duration(config.QPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Create a timer to stop the load generation after the specified duration.
	var durationTimer <-chan time.Time
	if config.Duration != -1 {
		durationTimer = time.NewTimer(time.Duration(config.Duration) * time.Second).C
	}

	// Create an HTTP client with a timeout.
	client := &http.Client{Timeout: 10 * time.Second}
	requestCount := 0

	// Main loop for sending requests.
	for {
		select {
		// When the ticker fires, send a request.
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
		// When the duration timer fires, stop the load generation.
		case <-durationTimer:
			log.Printf("[%s] Duration of %d seconds reached for %s. Sent %d requests.",
				config.FirestoreID, config.Duration, finalURL, requestCount)
			queryAndLogMetrics(config, finalURL, projectID, requestCount)
			return
		// When a stop signal is received, stop the load generation.
		case <-stop:
			log.Printf("[%s] Stopping load generation for %s.", config.FirestoreID, finalURL)
			return
		}
	}
}

// queryAndLogMetrics calculates and logs metrics about the load generation.
func queryAndLogMetrics(config ConfigParams, actualURL string, projectID string, requestsSentByThisInstance int) {
	// Calculate the actual QPS based on the number of requests sent and the duration.
	calculatedQPS := float64(requestsSentByThisInstance) / float64(config.Duration)

	// Log the metrics.
	log.Printf("METRICS [%s]: Target URL: %s, Configured QPS: %d, Actual QPS (calculated by this instance): %.2f",
		config.FirestoreID, actualURL, config.QPS, calculatedQPS)
}
