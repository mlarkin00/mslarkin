// Package main implements a simple web server that provides a form to configure
// load generation parameters. The configuration is then saved to a Firestore
// database.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
)

// ConfigParams holds the configuration parameters from the user input.
// These parameters are used to define a load generation test.
type ConfigParams struct {
	// TargetURL is the URL of the service to be tested.
	TargetURL string `firestore:"targetUrl" json:"targetUrl"`
	// TargetCPU is the target CPU utilization percentage for the load test.
	TargetCPU int `firestore:"targetCpu,omitempty" json:"targetCpu,omitempty"`
	// QPS is the number of queries per second to be sent to the target URL.
	QPS int `firestore:"qps,omitempty" json:"qps,omitempty"`
	// Duration is the duration of the load test in seconds.
	Duration int `firestore:"duration,omitempty" json:"duration,omitempty"`
}

// projectIDEnv is the environment variable that contains the Google Cloud project ID.
const projectIDEnv = "GOOGLE_CLOUD_PROJECT"

// collectionName is the name of the Firestore collection where the load generation
// configurations are stored.
const collectionName = "loadgen-configs"

var (
	// firestoreClient is the client used to interact with Firestore.
	firestoreClient *firestore.Client
)

//go:embed all:public
var publicFS embed.FS

// main is the entry point of the application. It initializes the Firestore client,
// sets up the HTTP server and handlers, and starts listening for requests.
func main() {
	var err error
	ctx := context.Background()
	projectID := os.Getenv(projectIDEnv)
	if projectID == "" {
		projectID = "mslarkin-ext" // Default project ID
	}

	firestoreClient, err = firestore.NewClientWithDatabase(ctx, projectID, "loadgen-target-config")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	fs, err := fs.Sub(publicFS, "public")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.FS(fs)))
	http.HandleFunc("/api/submit", handleSubmit)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// handleSubmit handles the POST request to the "/api/submit" URL. It parses the
// form data, validates it, and saves it to Firestore.
func handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var config ConfigParams
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	if config.TargetURL == "" {
		http.Error(w, "Target URL is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	docRef, _, err := firestoreClient.Collection(collectionName).Add(ctx, config)
	if err != nil {
		log.Printf("Error adding document to Firestore: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save configuration"})
		return
	}

	log.Printf("Configuration saved with ID: %s. TargetURL: %s, QPS: %d, Duration: %d, TargetCPU: %d",
		docRef.ID, config.TargetURL, config.QPS, config.Duration, config.TargetCPU)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Configuration saved successfully"})
}
