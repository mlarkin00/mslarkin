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
	"k8s-status-frontend/components"

	"k8s-status-frontend/views"
)

var (
	backendURL string
	basePath   string
)

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

func handleLanding(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	views.Landing(r).Render(w)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		// Use ResolveURL for redirect too
		http.Redirect(w, r, components.ResolveURL(r, "/"), http.StatusFound)
		return
	}

    // Switch to A2UI Shell.
    // We send an initial prompt to the agent to load the dashboard for the project.
    initialPrompt := fmt.Sprintf("Show me the cluster dashboard for project %s", project)
    views.A2UIShell(r, initialPrompt).Render(w)
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
