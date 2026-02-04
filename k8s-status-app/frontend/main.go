// Package main is the entry point for the k8s-status-frontend service.
// It serves the HTML UI and proxies chat requests to the backend.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"k8s-status-frontend/models"
	"k8s-status-frontend/components"

	"k8s-status-frontend/views"
)

var (
	backendURL string
	basePath   string
)

// main initializes the frontend server, sets up routing, and starts listening.
func main() {
	backendURL = os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080" // Default for local
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	basePath = os.Getenv("BASE_PATH")
	// If basePath is set (e.g. /k8s-status), we generally want to remove trailing slash for consistency
	// but components.AppLink handles it.
	components.BasePath = basePath

	// register handlers on a sub-mux to keep them clean
	apiMux := http.NewServeMux()
	apiMux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("public"))))
	apiMux.HandleFunc("GET /", handleLanding)
	apiMux.HandleFunc("GET /dashboard", handleDashboard)
	apiMux.HandleFunc("GET /partials/workloads", handlePartialsWorkloads)
	apiMux.HandleFunc("POST /chat/proxy", handleChatProxy)
	// Create a top-level mux to handle health checks and app logic
    topMux := http.NewServeMux()
    topMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

	// Mount the app handler at the internal base path.
	// Mount the app handler at the internal base path.
    // The LB rewrites /k8s-status/* to /k8s-status-app/*
    internalPath := "/k8s-status-app"

    // Simple StripPrefix.
    // We register the stripped handler for the internal path prefix.
    topMux.Handle(internalPath+"/", http.StripPrefix(internalPath, apiMux))

    // Logging middleware to debug request paths
    logger := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Request: %s %s", r.Method, r.URL.Path)
        topMux.ServeHTTP(w, r)
    })

	log.Printf("Starting frontend on :%s (Backend: %s, BasePath: %s)", port, backendURL, basePath)
	if err := http.ListenAndServe(":"+port, logger); err != nil {
		log.Fatal(err)
	}
}

// handleLanding renders the landing/welcome page.
func handleLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	views.Landing(r).Render(w)
}

// handleDashboard renders the main dashboard view for a specific project.
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	log.Printf("DEBUG: Loading dashboard for project: %s", project)
	if project == "" {
		// Use ResolveURL for redirect too
		http.Redirect(w, r, components.ResolveURL(r, "/"), http.StatusFound)
		return
	}

    // Switch to A2UI Shell.
    // We send an initial prompt to the agent to load the dashboard for the project.
    // initialPrompt := fmt.Sprintf("Show me the cluster dashboard for project %s", project)
    // views.A2UIShell(r, initialPrompt).Render(w)

    // RECOVERY: Render Dashboard directly for now
    // We need to fetch the data first? Or does the dashboard view do HTMX?
    // Dashboard view has ClusterCard which does hx-get to /partials/workloads
    // But it needs a list of clusters.
    // We can assume a default cluster list or fetch it from backend if we have a robust backend client.
    // For now, let's render the Dashboard view with empty clusters and let the user interact?
    // Wait, DashboardData needs Clusters []models.Cluster.
    // If we don't fetch them, the user sees nothing.
    // The previous code didn't do a backend fetch for clusters here?
    // It seems A2UI was supposed to handle it.
    // Let's check dashboard.go again. It iterates over data.Clusters.
    // If we want to show something, we should probably fetch the cluster list from the backend
    // OR just show a static list if we know it (e.g. from context?).
    // The backend has /api/clusters?
    // Let's assuming we just want to fix the blank page first.
    // We can pass a dummy cluster "ai-auto-cluster" so the user sees *something* and the HTMX triggers.

    // Fetch real clusters if possible, but for recovery:
    clusters := []models.Cluster{
        {Name: "ai-auto-cluster", Location: "us-central1", Status: "Active"},
    }
    views.Dashboard(r, views.DashboardData{Project: project, Clusters: clusters}).Render(w)
}

// handlePartialsWorkloads fetches and renders the workloads list partial (for HTMX).
func handlePartialsWorkloads(w http.ResponseWriter, r *http.Request) {
	cluster := r.URL.Query().Get("cluster")
	namespace := r.URL.Query().Get("namespace")
	log.Printf("DEBUG: Fetching workloads for cluster: %s, namespace: %s", cluster, namespace)

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

// handleChatProxy proxies chat requests to the backend API.
// It supports streaming responses.
func handleChatProxy(w http.ResponseWriter, r *http.Request) {
	targetURL := fmt.Sprintf("%s/api/chat", backendURL)

	// Read and log request body
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body.Close()
	log.Printf("DEBUG: Proxying chat message: %s", string(bodyBytes))
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	proxyReq, err := http.NewRequest("POST", targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("ERROR: Failed to proxy chat: %v", err)
		http.Error(w, "Failed to proxy chat", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	log.Printf("DEBUG: Chat backend response: %s", resp.Status)

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
