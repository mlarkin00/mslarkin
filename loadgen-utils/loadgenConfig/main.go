// Package main implements a simple web server that provides a form to configure
// load generation parameters. The configuration is then saved to a Firestore
// database.
package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"cloud.google.com/go/firestore"
)

// ConfigParams holds the configuration parameters from the user input.
// These parameters are used to define a load generation test.
type ConfigParams struct {
	// TargetURL is the URL of the service to be tested.
	TargetURL string `firestore:"targetUrl"`
	// TargetCPU is the target CPU utilization percentage for the load test.
	TargetCPU int `firestore:"targetCpu,omitempty"`
	// QPS is the number of queries per second to be sent to the target URL.
	QPS int `firestore:"qps,omitempty"`
	// Duration is the duration of the load test in seconds.
	Duration int `firestore:"duration,omitempty"`
	// FirestoreID is the document ID of the configuration in Firestore.
	// It is not stored in Firestore itself, but used to reference the document.
	FirestoreID string `firestore:"-"` // Used to store document ID, not stored in Firestore fields
}

// projectIDEnv is the environment variable that contains the Google Cloud project ID.
const projectIDEnv = "GOOGLE_CLOUD_PROJECT"

// collectionName is the name of the Firestore collection where the load generation
// configurations are stored.
const collectionName = "loadgen-configs"

var (
	// tmpl is the HTML template for the configuration form.
	tmpl = template.Must(template.New("form").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>Loadgen Config</title>
</head>
<body>
    <h2>Configure Load Generation</h2>
    <form method="POST" action="/submit">
        <label for="targetURL">Target URL (required):</label><br>
        <input type="text" id="targetURL" name="targetURL" required><br><br>

        <label for="targetCPU">Target CPU Utilization % (optional, default 0):</label><br>
        <input type="number" id="targetCPU" name="targetCPU" min="0" max="100" value="0"><br><br>

        <label for="qps">QPS (Queries Per Second, optional, default 1):</label><br>
        <input type="number" id="qps" name="qps" min="1" value="1"><br><br>

        <label for="duration">Duration in seconds (optional, default 1):</label><br>
        <input type="number" id="duration" name="duration" min="1" value="1"><br><br>

        <input type="submit" value="Submit">
    </form>
</body>
</html>
`))
	// firestoreClient is the client used to interact with Firestore.
	firestoreClient *firestore.Client
)

// main is the entry point of the application. It initializes the Firestore client,
// sets up the HTTP server and handlers, and starts listening for requests.
func main() {
	ctx := context.Background()
	projectID := os.Getenv(projectIDEnv)
	if projectID == "" {
		log.Fatalf("Environment variable %s must be set.", projectIDEnv)
	}

	var err error
	firestoreClient, err = firestore.NewClientWithDatabase(ctx, projectID, "loadgen-target-config")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	http.HandleFunc("/", handleForm)
	http.HandleFunc("/submit", handleSubmit)

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

// handleForm handles the GET request to the root URL ("/"). It displays the
// configuration form to the user.
func handleForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error executing template: %v", err), http.StatusInternalServerError)
	}
}

// handleSubmit handles the POST request to the "/submit" URL. It parses the
// form data, validates it, and saves it to Firestore.
func handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
		return
	}

	config := ConfigParams{
		TargetURL: r.FormValue("targetURL"),
	}

	if config.TargetURL == "" {
		http.Error(w, "Target URL is required", http.StatusBadRequest)
		return
	}

	var err error
	if cpuStr := r.FormValue("targetCPU"); cpuStr != "" {
		config.TargetCPU, err = strconv.Atoi(cpuStr)
		if err != nil {
			http.Error(w, "Invalid Target CPU value", http.StatusBadRequest)
			return
		}
	} else {
		config.TargetCPU = 0 // Default
	}

	if qpsStr := r.FormValue("qps"); qpsStr != "" {
		config.QPS, err = strconv.Atoi(qpsStr)
		if err != nil {
			http.Error(w, "Invalid QPS value", http.StatusBadRequest)
			return
		}
	} else {
		config.QPS = 1 // Default
	}
	if config.QPS < 1 { // Ensure QPS is at least 1 even if a non-default empty value was somehow submitted
		config.QPS = 1
	}

	if durStr := r.FormValue("duration"); durStr != "" {
		config.Duration, err = strconv.Atoi(durStr)
		if err != nil {
			http.Error(w, "Invalid Duration value", http.StatusBadRequest)
			return
		}
	} else {
		config.Duration = 1 // Default
	}
	if config.Duration < 1 { // Ensure duration is at least 1
		config.Duration = 1
	}

	ctx := r.Context()
	docRef, _, err := firestoreClient.Collection(collectionName).Add(ctx, config)
	if err != nil {
		log.Printf("Error adding document to Firestore: %v", err)
		http.Error(w, "Error saving configuration to database", http.StatusInternalServerError)
		return
	}

	log.Printf("Configuration saved with ID: %s. TargetURL: %s, QPS: %d, Duration: %d, TargetCPU: %d",
		docRef.ID, config.TargetURL, config.QPS, config.Duration, config.TargetCPU)

	fmt.Fprintf(w, "Configuration saved successfully! Document ID: %s", docRef.ID)
}
