// Package api provides the HTTP handlers for the REST API.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"k8s-status-backend/chat"
	"k8s-status-backend/mcpclient"
)

// Server holds dependencies for the API handlers.
type Server struct {
	MCPClient *mcpclient.MCPClient
	Chat      *chat.ChatService
}

// ListProjects returns a list of projects.
func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request) {
	// Dynamically fetch projects from env or default
	raw := os.Getenv("PROJECT_IDS")
	var projects []string
	if raw != "" {
		projects = strings.Split(raw, ",")
	} else {
		projects = []string{"mslarkin-ext", "mslarkin-demo"}
	}
	json.NewEncoder(w).Encode(projects)
}

// ListClusters returns a list of clusters for a given project.
// Query param: project
func (s *Server) ListClusters(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		http.Error(w, "project is required", http.StatusBadRequest)
		return
	}

	clusters, err := s.MCPClient.ListClusters(r.Context(), projectID)
	if err != nil {
		log.Printf("Error listing clusters: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(clusters)
}

// ListWorkloads returns a list of workloads for a given cluster and namespace.
// Query params: cluster, namespace
func (s *Server) ListWorkloads(w http.ResponseWriter, r *http.Request) {
	cluster := r.URL.Query().Get("cluster")
	namespace := r.URL.Query().Get("namespace")
	if cluster == "" || namespace == "" {
		http.Error(w, "cluster and namespace are required", http.StatusBadRequest)
		return
	}

	workloads, err := s.MCPClient.ListWorkloads(r.Context(), cluster, namespace)
	if err != nil {
		log.Printf("Error listing workloads: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(workloads)
}

// ChatHandler handles the streaming chat interface using SSE.
// It proxies the user message to the ADK agent and streams the response back.
func (s *Server) ChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	type ChatRequest struct {
		Message string `json:"message"`
		Session string `json:"session"`
	}
	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		req.Message = string(body)
		req.Session = "default"
	}
	if req.Session == "" {
		req.Session = "default"
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	events, err := s.Chat.Chat(r.Context(), req.Session, req.Message)
	if err != nil {
		log.Printf("Error starting chat: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	for event, err := range events {
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %v\n\n", err)
			flusher.Flush()
			return
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// GetWorkload returns details for a specific workload.
func (s *Server) GetWorkload(w http.ResponseWriter, r *http.Request) {
	cluster := r.URL.Query().Get("cluster")
	namespace := r.URL.Query().Get("namespace")
	name := r.PathValue("name") // Go 1.22+ regex path match or just extracting from path manually if older.

	// Check if r.PathValue is available (Go 1.22+). If not, we might need manual parsing or mux vars.
	// Assuming Go 1.22+ since we used "GET /api/..." in mux matching.

	if cluster == "" || namespace == "" || name == "" {
		http.Error(w, "cluster, namespace, and name are required", http.StatusBadRequest)
		return
	}

	workload, err := s.MCPClient.GetWorkload(r.Context(), cluster, namespace, name)
	if err != nil {
		log.Printf("Error getting workload: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(workload)
}

// ListPods returns pods for a specific workload.
func (s *Server) ListPods(w http.ResponseWriter, r *http.Request) {
	cluster := r.URL.Query().Get("cluster")
	namespace := r.URL.Query().Get("namespace")
	name := r.PathValue("name")

	if cluster == "" || namespace == "" || name == "" {
		http.Error(w, "cluster, namespace, and name are required", http.StatusBadRequest)
		return
	}

	pods, err := s.MCPClient.ListPods(r.Context(), cluster, namespace, name)
	if err != nil {
		log.Printf("Error listing pods: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(pods)
}
