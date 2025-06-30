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
	"google.golang.org/api/iterator"
)

// ConfigParams holds the configuration parameters from the user input.
// These parameters are used to define a load generation test.
type ConfigParams struct {
	ID          string `firestore:"-" json:"id"`
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
	http.HandleFunc("/api/configs", handleGetConfigs)
	http.HandleFunc("/api/delete/", handleDeleteConfig)
	http.HandleFunc("/api/update/", handleUpdateConfig)

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

// handleGetConfigs handles the GET request to the "/api/configs" URL. It fetches
// all the configurations from Firestore and returns them as a JSON array.
func handleGetConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	var configs []ConfigParams
	iter := firestoreClient.Collection(collectionName).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating documents: %v", err)
			http.Error(w, "Failed to retrieve configurations", http.StatusInternalServerError)
			return
		}
		var config ConfigParams
		if err := doc.DataTo(&config); err != nil {
			log.Printf("Error converting document data: %v", err)
			continue
		}
		config.ID = doc.Ref.ID
		configs = append(configs, config)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(configs); err != nil {
		log.Printf("Error encoding configs to JSON: %v", err)
		http.Error(w, "Failed to encode configurations", http.StatusInternalServerError)
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

// handleDeleteConfig handles the DELETE request to the "/api/delete/{id}" URL.
// It deletes the specified configuration from Firestore.
func handleDeleteConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Only DELETE method is allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/api/delete/"):]
	ctx := r.Context()
	_, err := firestoreClient.Collection(collectionName).Doc(id).Delete(ctx)
	if err != nil {
		log.Printf("Error deleting document: %v", err)
		http.Error(w, "Failed to delete configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Configuration deleted successfully"})
}

// handleUpdateConfig handles the PUT request to the "/api/update/{id}" URL.
// It updates the specified configuration in Firestore.
func handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Only PUT method is allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/api/update/"):]
	var config ConfigParams
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	_, err := firestoreClient.Collection(collectionName).Doc(id).Set(ctx, config)
	if err != nil {
		log.Printf("Error updating document: %v", err)
		http.Error(w, "Failed to update configuration", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Configuration updated successfully"})
}
