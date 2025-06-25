// Package main implements a simple web server that provides a form to configure
// load generation parameters. The configuration is then saved to a Firestore
// database.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"cloud.google.com/go/firestore"
	"github.com/donseba/go-htmx"
	"google.golang.org/api/iterator"
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

// PageData holds the data to be passed to the HTML template.
type PageData struct {
	Configs []ConfigParams
	Message string
}

// projectIDEnv is the environment variable that contains the Google Cloud project ID.
const projectIDEnv = "GOOGLE_CLOUD_PROJECT"

// collectionName is the name of the Firestore collection where the load generation
// configurations are stored.
const collectionName = "loadgen-configs"

var (
	// firestoreClient is the client used to interact with Firestore.
	firestoreClient *firestore.Client
	htmx_app        *htmx.HTMX
	htmx_handler    *htmx.Handler
	html_template   *template.Template
)

func getPageData(ctx context.Context) (*PageData, error) {
	iter := firestoreClient.Collection(collectionName).Documents(ctx)
	var configs []ConfigParams
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Failed to iterate configs: %v", err)
			return nil, err
		}
		var config ConfigParams
		if err := doc.DataTo(&config); err != nil {
			log.Printf("Failed to decode config: %v", err)
			continue
		}
		config.FirestoreID = doc.Ref.ID
		configs = append(configs, config)
	}

	return &PageData{Configs: configs}, nil
}

// main is the entry point of the application. It initializes the Firestore client,
// sets up the HTTP server and handlers, and starts listening for requests.
func main() {
	var err error
	ctx := context.Background()
	projectID := os.Getenv(projectIDEnv)
	if projectID == "" {
		projectID = "mslarkin-ext" // Default project ID
	}

	htmx_app = htmx.New()
	html_template, err = template.ParseGlob("templates/*")
	if err != nil {
		panic(err)
	}
	log.Printf("Templates: %v", html_template.DefinedTemplates())

	firestoreClient, err = firestore.NewClientWithDatabase(ctx, projectID, "loadgen-target-config")
	if err != nil {
		log.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer firestoreClient.Close()

	http.HandleFunc("/", handleForm)
	http.HandleFunc("/submit", handleSubmit)
	http.HandleFunc("/delete", handleDelete)
	http.HandleFunc("/update", handleUpdate)
	http.HandleFunc("/get_config", handleGetConfig)

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

	htmx_handler = htmx_app.NewHandler(w, r)

	pageData, err := getPageData(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve configs", http.StatusInternalServerError)
		return
	}

	html_template.ExecuteTemplate(w, "form.html", pageData)
	// htmx_handler.Render(r.Context(), htmx_form)
	// htmx_handler.Render(w, r, "form.html", pageData)
}

// handleDelete handles the POST request to the "/delete" URL. It deletes a
// configuration from Firestore.
func handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	htmx_handler = htmx_app.NewHandler(w, r)

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "Missing config ID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	_, err := firestoreClient.Collection(collectionName).Doc(id).Delete(ctx)
	if err != nil {
		log.Printf("Failed to delete config %s: %v", id, err)
		pageData, err := getPageData(r.Context())
		if err != nil {
			http.Error(w, "Failed to retrieve configs", http.StatusInternalServerError)
			return
		}
		pageData.Message = fmt.Sprintf("Failed to delete config for %s: %v", r.FormValue("targetURL"), err)
		// htmx_handler.Render(w, r, "form.html", pageData)
		// htmx_handler.Render(r.Context(), htmx_form)
		html_template.ExecuteTemplate(w, "form.html", pageData)
		return
	}

	log.Printf("Deleted config %s", id)
	pageData, err := getPageData(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve configs", http.StatusInternalServerError)
		return
	}
	pageData.Message = fmt.Sprintf("Successfully deleted config for %s", r.FormValue("targetURL"))
	// htmx_handler.Render(w, r, "form.html", pageData)
	// htmx_handler.Render(r.Context(), htmx_form)
	html_template.ExecuteTemplate(w, "form.html", pageData)
}

// handleGetConfig handles the GET request to the "/get_config" URL. It returns
// a configuration as JSON.
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// htmx_handler = htmx_app.NewHandler(w, r)

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing config ID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	doc, err := firestoreClient.Collection(collectionName).Doc(id).Get(ctx)
	if err != nil {
		log.Printf("Failed to get config %s: %v", id, err)
		http.Error(w, "Failed to get config", http.StatusInternalServerError)
		return
	}

	var config ConfigParams
	if err := doc.DataTo(&config); err != nil {
		log.Printf("Failed to decode config: %v", err)
		http.Error(w, "Failed to decode config", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// handleSubmit handles the POST request to the "/submit" URL. It parses the
// form data, validates it, and saves it to Firestore.
func handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	htmx_handler = htmx_app.NewHandler(w, r)

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
		pageData, err := getPageData(r.Context())
		if err != nil {
			http.Error(w, "Failed to retrieve configs", http.StatusInternalServerError)
			return
		}
		pageData.Message = fmt.Sprintf("Error saving configuration for %s: %v", config.TargetURL, err)
		// htmx_handler.Render(w, r, "form.html", pageData)
		// htmx_handler.Render(r.Context(), htmx_form)
		html_template.ExecuteTemplate(w, "form.html", pageData)
		return
	}

	log.Printf("Configuration saved with ID: %s. TargetURL: %s, QPS: %d, Duration: %d, TargetCPU: %d",
		docRef.ID, config.TargetURL, config.QPS, config.Duration, config.TargetCPU)

	pageData, err := getPageData(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve configs", http.StatusInternalServerError)
		return
	}
	pageData.Message = fmt.Sprintf("Successfully added config for %s", config.TargetURL)
	// htmx_handler.Render(w, r, "form.html", pageData)
	// htmx_handler.Render(r.Context(), htmx_form)
	html_template.ExecuteTemplate(w, "form.html", pageData)
}

// // handleUpdate handles the POST request to the "/update" URL. It parses the
// // form data, validates it, and updates it in Firestore.
func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	htmx_handler = htmx_app.NewHandler(w, r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "Missing config ID", http.StatusBadRequest)
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
	if config.QPS < 1 {
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
	if config.Duration < 1 {
		config.Duration = 1
	}

	ctx := r.Context()
	_, err = firestoreClient.Collection(collectionName).Doc(id).Set(ctx, config)
	if err != nil {
		log.Printf("Error updating document in Firestore: %v", err)
		pageData, err := getPageData(r.Context())
		if err != nil {
			http.Error(w, "Failed to retrieve configs", http.StatusInternalServerError)
			return
		}
		pageData.Message = fmt.Sprintf("Error updating configuration for %s: %v", config.TargetURL, err)
		// htmx_handler.Render(w, r, "form.html", pageData)
		// htmx_handler.Render(r.Context(), htmx_form)
		html_template.ExecuteTemplate(w, "form.html", pageData)
		return
	}

	log.Printf("Configuration updated with ID: %s. TargetURL: %s, QPS: %d, Duration: %d, TargetCPU: %d",
		id, config.TargetURL, config.QPS, config.Duration, config.TargetCPU)

	pageData, err := getPageData(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve configs", http.StatusInternalServerError)
		return
	}
	pageData.Message = fmt.Sprintf("Successfully updated config for %s", config.TargetURL)
	// htmx_handler.Render(w, r, "form.html", pageData)
	// htmx_handler.Render(r.Context(), htmx_form)
	html_template.ExecuteTemplate(w, "form.html", pageData)
}
