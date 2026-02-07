package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/mslarkin/coding-multi-agent/internal/code"
	"github.com/mslarkin/coding-multi-agent/internal/planning"
	"github.com/mslarkin/coding-multi-agent/internal/worker"
	"github.com/mslarkin/coding-multi-agent/pkg/agent"
	"github.com/mslarkin/coding-multi-agent/pkg/config"
	"github.com/mslarkin/coding-multi-agent/pkg/llm"
	"github.com/mslarkin/coding-multi-agent/pkg/mcp"
	"github.com/mslarkin/coding-multi-agent/pkg/state"
	"github.com/mslarkin/coding-multi-agent/pkg/telemetry"
)

func main() {
	// 1. Load Config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Printf("Warning: Failed to load config file: %v. Using defaults/env vars.", err)
		cfg, _ = config.Load("") // Load defaults
	}

	// 2. Initialize Telemetry
	shutdownTracer := telemetry.InitTracer("adk-agent")
	defer shutdownTracer()
	telemetry.InitMetrics()

	// 3. Initialize Shared Services
	ctx := context.Background()
	stateService := state.NewService(cfg.Storage.Redis)

	// Initialize MCP Client (generic one, typically configured per agent)
	// For simplicity, we use the "docs_onemcp" config as default for workers if not specified
	mcpCfg := cfg.MCPClients["docs_onemcp"]
	mcpClient, err := mcp.NewClient(ctx, mcpCfg)
	if err != nil {
		log.Printf("Warning: Failed to init MCP client: %v", err)
	}

	// 4. Initialize Agent based on Role
	role := os.Getenv("AGENT_ROLE")
	if role == "" {
		role = "planning" // Default
	}
	log.Printf("Starting agent with role: %s", role)

	var ag agent.Agent
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "mslarkin-ext"
	}

	useMocks := os.Getenv("USE_MOCKS") == "true"

	getLLMClient := func(ctx context.Context, modelCfg config.ModelConfig) llm.Client {
		if useMocks {
			log.Println("Using Mock LLM Client")
			return llm.NewMockClient()
		}
		client, err := llm.NewVertexClient(ctx, projectID, modelCfg.Location)
		if err != nil {
			log.Fatalf("Failed to create Vertex AI client: %v", err)
		}
		return client
	}

	switch role {
	case "planning":
		modelCfg := cfg.Models["planning"]
		llmClient := getLLMClient(ctx, modelCfg)
		ag = planning.NewPlanningAgent(modelCfg, llmClient)

	case "code_primary":
		modelCfg := cfg.Models["code_primary"]
		llmClient := getLLMClient(ctx, modelCfg)
		ag = code.NewPrimaryAgent(modelCfg, llmClient, mcpClient, stateService)

	case "code_secondary":
		modelCfg := cfg.Models["code_secondary"]
		llmClient := getLLMClient(ctx, modelCfg)
		ag = worker.NewWorkerAgent("code_secondary", modelCfg, llmClient, mcpClient, stateService,
			"You are the Secondary Code Agent. Review the code options.")

	case "design":
		modelCfg := cfg.Models["design"]
		llmClient := getLLMClient(ctx, modelCfg)
		ag = worker.NewWorkerAgent("design", modelCfg, llmClient, mcpClient, stateService,
			"You are the Design Agent. Generate kubernetes manifests.")

	case "ops":
		modelCfg := cfg.Models["ops"]
		llmClient := getLLMClient(ctx, modelCfg)
		ag = worker.NewWorkerAgent("ops", modelCfg, llmClient, mcpClient, stateService,
			"You are the Ops Agent. execute operations.")

	case "validation":
		modelCfg := cfg.Models["validation"]
		llmClient := getLLMClient(ctx, modelCfg)
		ag = worker.NewWorkerAgent("validation", modelCfg, llmClient, mcpClient, stateService,
			"You are the Validation Agent. Validate the code.")

	case "review":
		modelCfg := cfg.Models["review"]
		llmClient := getLLMClient(ctx, modelCfg)
		ag = worker.NewWorkerAgent("review", modelCfg, llmClient, mcpClient, stateService,
			"You are the Review Agent. Provide final approval.")

	default:
		log.Fatalf("Unknown role: %s", role)
	}

	// 5. Start HTTP Server
	mux := http.NewServeMux()

	// Health Checks
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Main Endpoint
	// /v1/chat/completions (OpenAI style) - for Planning Agent primarily
	// /process (Internal A2A) - for all agents

	mux.HandleFunc("POST /v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		// Adapt OpenAI request to AgentInput
		// This is a simplification. Real implementation needs full adapter.
		var req struct {
			Messages []agent.Message `json:"messages"`
			Model    string          `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		input := agent.AgentInput{
			Task:    req.Messages[len(req.Messages)-1].Content,
			Context: req.Messages,
		}

		// Support Streaming
		// if stream=true ...

		resp, err := ag.Process(r.Context(), input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return OpenAI style response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": resp.Content,
					},
				},
			},
		})
	})

	mux.HandleFunc("POST /process", func(w http.ResponseWriter, r *http.Request) {
		var input agent.AgentInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := ag.Process(r.Context(), input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(resp)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Starting server on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
