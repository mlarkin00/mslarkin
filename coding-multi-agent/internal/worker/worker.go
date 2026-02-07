package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/mslarkin/coding-multi-agent/pkg/agent"
	"github.com/mslarkin/coding-multi-agent/pkg/config"
	"github.com/mslarkin/coding-multi-agent/pkg/llm"
	"github.com/mslarkin/coding-multi-agent/pkg/mcp"
	"github.com/mslarkin/coding-multi-agent/pkg/state"
)

type WorkerAgent struct {
	Role         string
	ModelConfig  config.ModelConfig
	LLMClient    llm.Client
	MCPClient    mcp.Client
	StateService *state.Service
	SystemPrompt string
}

func NewWorkerAgent(role string, cfg config.ModelConfig, llmClient llm.Client, mcpClient mcp.Client, state *state.Service, sysPrompt string) *WorkerAgent {
	return &WorkerAgent{
		Role:         role,
		ModelConfig:  cfg,
		LLMClient:    llmClient,
		MCPClient:    mcpClient,
		StateService: state,
		SystemPrompt: sysPrompt,
	}
}

func (a *WorkerAgent) Process(ctx context.Context, input agent.AgentInput) (*agent.AgentResponse, error) {
	startTime := time.Now()

	// 1. Construct Prompt
	// Simple concatenation for now. In reality, we'd format chat history nicely.
	prompt := a.SystemPrompt + "\n\nTask: " + input.Task + "\n"
	for _, msg := range input.Context {
		prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	// Add Constraints
	if len(input.Constraints) > 0 {
		prompt += "\nConstraints:\n"
		for _, c := range input.Constraints {
			prompt += "- " + c + "\n"
		}
	}

	// 2. Call LLM
	req := llm.GenerateRequest{
		ModelID:     a.ModelConfig.ModelID,
		Prompt:      prompt,
		IsThinking:  a.ModelConfig.IsThinking,
		Temperature: 0.7,
	}

	resp, err := a.LLMClient.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm generation failed: %w", err)
	}

	// 3. Parse Response (if needed) and format AgentResponse
	// For simplicity, we assume the model returns the content directly.
	// Status determination logic would go here (e.g., looking for specific tags).

	status := agent.StatusCompleted
	if a.Role == "code_primary" {
		status = agent.StatusPendingReview
	}

	return &agent.AgentResponse{
		Content:   resp.Content,
		Status:    status,
		Latency:   time.Since(startTime),
		TokenUsage: agent.TokenMetrics{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (a *WorkerAgent) StreamProcess(ctx context.Context, input agent.AgentInput, outputChan chan<- agent.AgentStreamUpdate) error {
	// Not implemented for basic worker yet.
	return nil
}
