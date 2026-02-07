package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mslarkin/coding-multi-agent/internal/code"
	"github.com/mslarkin/coding-multi-agent/internal/planning"
	"github.com/mslarkin/coding-multi-agent/internal/worker"
	"github.com/mslarkin/coding-multi-agent/pkg/agent"
	"github.com/mslarkin/coding-multi-agent/pkg/config"
	"github.com/mslarkin/coding-multi-agent/pkg/llm"
)

func TestPlanningToCodeFlow(t *testing.T) {
	ctx := context.Background()

	// 1. Setup Mock Services
	mockLLM := llm.NewMockClient()
	// Populate mock responses
	mockLLM.Responses["Review the code options."] = "Option A is the best."

	// 2. Setup Secondary Agent Server
	secondaryAgent := worker.NewWorkerAgent("code_secondary", config.ModelConfig{ModelID: "test"}, mockLLM, nil, nil, "System Prompt")
	secondaryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input agent.AgentInput
		json.NewDecoder(r.Body).Decode(&input)
		resp, _ := secondaryAgent.Process(r.Context(), input)
		json.NewEncoder(w).Encode(resp)
	})
	secondaryServer := httptest.NewServer(secondaryHandler)
	defer secondaryServer.Close()

	// 3. Setup Primary Agent Server
	primaryAgent := code.NewPrimaryAgent(config.ModelConfig{ModelID: "test"}, mockLLM, nil, nil)
	primaryAgent.SecondaryAgentURL = secondaryServer.URL

	primaryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input agent.AgentInput
		json.NewDecoder(r.Body).Decode(&input)
		resp, _ := primaryAgent.Process(r.Context(), input)
		json.NewEncoder(w).Encode(resp)
	})
	primaryServer := httptest.NewServer(primaryHandler)
	defer primaryServer.Close()

	// 4. Setup Planning Agent
	planningAgent := planning.NewPlanningAgent(config.ModelConfig{ModelID: "test"}, mockLLM)
	planningAgent.CodeAgentURL = primaryServer.URL

	// 5. Run Test
	input := agent.AgentInput{
		Task: "Please implement a fibonacci function in Go",
	}

	resp, err := planningAgent.Process(ctx, input)
	if err != nil {
		t.Fatalf("Planning agent failed: %v", err)
	}

	// 6. Verify
	if resp == nil {
		t.Fatal("Response is nil")
	}
	// The mock LLM returns "Mock response for: ..."
	// And Primary Agent calls Secondary.
	// Verify that we got a response that bubbled up.
	if !strings.Contains(resp.Content, "Mock response") {
		t.Errorf("Expected mock response content, got: %s", resp.Content)
	}
	t.Logf("Final Response: %s", resp.Content)
}
