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
type ConfigParams struct {
	TargetURL     string `firestore:"targetUrl"`
	TargetCPU     int    `firestore:"targetCpu,omitempty"`
	QPS           int    `firestore:"qps,omitempty"`
	Duration      int    `firestore:"duration,omitempty"`
	FirestoreID   string `firestore:"-"` // Used to store document ID, not stored in Firestore fields
}

const projectIDEnv = "GOOGLE_CLOUD_PROJECT"
const collectionName = "loadgen-configs"

var (
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
	firestoreClient *firestore.Client
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
