package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

    gke "k8s-status-backend/pkg/gke"
    k8s "k8s-status-backend/pkg/k8s"
    status "k8s-status-backend/pkg/status"
)

func main() {
    ctx := context.Background()
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    // Initialize Services
    discovery, err := gke.NewDiscoveryClient(ctx)
    if err != nil {
        log.Fatalf("Failed to init GKE discovery: %v", err)
    }
    defer discovery.Close()

    clientManager := k8s.NewClientManager()
    aggregator := status.NewAggregator(clientManager)

    // Handlers
    mux := http.NewServeMux()

    // CORS middleware
    enableCORS := func(h http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all for demo
            w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
            if r.Method == "OPTIONS" {
                return
            }
            h(w, r)
        }
    }

    mux.HandleFunc("GET /api/status", enableCORS(func(w http.ResponseWriter, r *http.Request) {
        // Projects to monitor
        projects := []string{"mslarkin-ext", "mslarkin-demo"}

        clusters, err := discovery.ListClusters(r.Context(), projects)
        if err != nil {
            http.Error(w, "Failed to list clusters: "+err.Error(), http.StatusInternalServerError)
            return
        }

        data := aggregator.FetchAll(r.Context(), clusters)

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(data)
    }))

    mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("ok"))
    })

    log.Printf("Starting backend on :%s", port)
    if err := http.ListenAndServe(":"+port, mux); err != nil {
        log.Fatal(err)
    }
}
