package code

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mslarkin/coding-multi-agent/internal/worker"
	"github.com/mslarkin/coding-multi-agent/pkg/agent"
	"github.com/mslarkin/coding-multi-agent/pkg/config"
	"github.com/mslarkin/coding-multi-agent/pkg/llm"
	"github.com/mslarkin/coding-multi-agent/pkg/mcp"
	"github.com/mslarkin/coding-multi-agent/pkg/state"
)

type PrimaryAgent struct {
	*worker.WorkerAgent
	SecondaryAgentURL string
}

func NewPrimaryAgent(cfg config.ModelConfig, llmClient llm.Client, mcpClient mcp.Client, state *state.Service) *PrimaryAgent {
	base := worker.NewWorkerAgent("code_primary", cfg, llmClient, mcpClient, state,
		"You are the Primary Code Agent. Generate 3 distinct options for the code requested.")
	return &PrimaryAgent{
		WorkerAgent:       base,
		SecondaryAgentURL: "http://agent-code-secondary:8080",
	}
}

func (a *PrimaryAgent) Process(ctx context.Context, input agent.AgentInput) (*agent.AgentResponse, error) {
	// 1. Generate Options
	resp, err := a.WorkerAgent.Process(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. Call Secondary Agent for Review
	reviewInput := agent.AgentInput{
		Task:    "Review these options and select the best one.",
		Context: append(input.Context, agent.Message{Role: "assistant", Content: resp.Content}),
	}

	reviewResp, err := a.callRemoteAgent(ctx, a.SecondaryAgentURL, reviewInput)
	if err != nil {
		// Fallback: return own options if review fails? Or error out?
		// For now, return error to indicate failure in pipeline
		return nil, fmt.Errorf("secondary review failed: %w", err)
	}

	// 3. Return Final Result
	return reviewResp, nil
}

func (a *PrimaryAgent) callRemoteAgent(ctx context.Context, url string, input agent.AgentInput) (*agent.AgentResponse, error) {
	jsonBody, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url+"/process", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote agent returned status %d", resp.StatusCode)
	}

	var agentResp agent.AgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		return nil, err
	}

	return &agentResp, nil
}
