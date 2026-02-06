package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	internalAgent "github.com/mslarkin/models/adk-agent/internal/agent"
	"google.golang.org/adk/agent" // Import base agent package
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/server/adkrest"
	"google.golang.org/adk/session/database"
)

func main() {
	ctx := context.Background()
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		projectID = "mslarkin-ext" // Default project
	}

	// 1. Initialize Agents
	agents, err := internalAgent.NewAgents(ctx, projectID)

	if err != nil {
		log.Fatalf("Failed to initialize agents: %v", err)
	}

	if len(agents) == 0 {
		log.Fatal("No agents initialized")
	}

	// 2. Initialize MultiLoader
	var loader agent.Loader
	if len(agents) > 1 {
		// Provide root agent (agents[0]) and additional agents (agents[1:])
		loader, err = agent.NewMultiLoader(agents[0], agents[1:]...)
	} else {
		loader, err = agent.NewMultiLoader(agents[0])
	}
	if err != nil {
		log.Fatalf("Failed to create multi-loader: %v", err)
	}

	// 3. Initialize Services (Session & Memory)
	// Using in-memory SQLite for simplicity and speed.
	db, err := database.NewSessionService(sqlite.Open(":memory:"))
	if err != nil {
		log.Fatalf("Failed to create session service: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	mem := memory.InMemoryService()

	// 4. Create Launcher Config
	// Use InMemoryArtifactService for now
	artifactService := artifact.InMemoryService()

	config := &launcher.Config{
		AgentLoader:     loader,
		SessionService:  db,
		MemoryService:   mem,
		ArtifactService: artifactService,
	}

	// 5. Create ADK Handler
	adkHandler := adkrest.NewHandler(config, 30*time.Second)

	mux := http.NewServeMux()

	// Explicit health check for GKE Probes
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Mount ADK handler under /api/ with StripPrefix for Ingress compatibility
	// This matches the pattern in GKE AI Agent Patterns
	mux.Handle("/api/", http.StripPrefix("/api", adkHandler))
	// Also mount at root for direct access if needed
	mux.Handle("/", adkHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting ADK Agent Server on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
