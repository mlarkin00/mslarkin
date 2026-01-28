// Package api provides the HTTP handlers for the REST API.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"k8s-status-backend/chat"
	"k8s-status-backend/mcpclient"
)

// Server holds dependencies for the API handlers.
type Server struct {
	MCPClient *mcpclient.MCPClient
	Chat      *chat.ChatService
}

// ListProjects returns a hardcoded list of projects for the demo.
func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request) {
	// Hardcoded for now as per demo
	projects := []string{"mslarkin-ext", "mslarkin-demo"}
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
		http.Error(w, fmt.Sprintf("failed to list clusters: %v", err), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("failed to list workloads: %v", err), http.StatusInternalServerError)
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
		http.Error(w, fmt.Sprintf("failed to start chat: %v", err), http.StatusInternalServerError)
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
