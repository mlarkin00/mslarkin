package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"k8s-status-frontend/models"
	"k8s-status-frontend/views"
)

var backendURL string

func main() {
	backendURL = os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080" // Default for local
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleLanding)
	mux.HandleFunc("GET /dashboard", handleDashboard)
	mux.HandleFunc("GET /partials/workloads", handlePartialsWorkloads)
	mux.HandleFunc("POST /chat/proxy", handleChatProxy)

	log.Printf("Starting frontend on :%s (Backend: %s)", port, backendURL)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func handleLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	views.Landing().Render(w)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Fetch clusters
	resp, err := http.Get(fmt.Sprintf("%s/api/clusters?project=%s", backendURL, url.QueryEscape(project)))
	if err != nil {
		http.Error(w, "Failed to connect to backend", http.StatusBadGateway)
		log.Printf("Error fetching clusters: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch clusters", http.StatusInternalServerError)
		log.Printf("Backend returned status: %d", resp.StatusCode)
		return
	}

	var clusters []models.Cluster
	if err := json.NewDecoder(resp.Body).Decode(&clusters); err != nil {
		http.Error(w, "Failed to decode clusters", http.StatusInternalServerError)
		return
	}

	views.Dashboard(views.DashboardData{
		Project:  project,
		Clusters: clusters,
	}).Render(w)
}

func handlePartialsWorkloads(w http.ResponseWriter, r *http.Request) {
	cluster := r.URL.Query().Get("cluster")
	namespace := r.URL.Query().Get("namespace")

	resp, err := http.Get(fmt.Sprintf("%s/api/workloads?cluster=%s&namespace=%s", backendURL, url.QueryEscape(cluster), url.QueryEscape(namespace)))
	if err != nil {
		http.Error(w, "Failed to connect to backend", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var workloads []models.Workload
	if err := json.NewDecoder(resp.Body).Decode(&workloads); err != nil {
		http.Error(w, "Failed to decode workloads", http.StatusInternalServerError)
		return
	}

	views.WorkloadsList(workloads).Render(w)
}

func handleChatProxy(w http.ResponseWriter, r *http.Request) {
	targetURL := fmt.Sprintf("%s/api/chat", backendURL)

	proxyReq, err := http.NewRequest("POST", targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Failed to proxy chat", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

    // Manual copy with flush support could be better, but io.Copy is usually fine for simple proxying.
    // If strict streaming is needed, we loop and flush.
    // But since backend returns "Transfer-Encoding: chunked" or SSE content type,
    // client.Do might buffer? No, Body is a stream.
	io.Copy(w, resp.Body)
}
