package agent

import (
	"context"

	"github.com/mslarkin/models/adk-agent/internal/model"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
)

type ModelConfig struct {
	ID          string
	DisplayName string
	IsThinking  bool
	Region      string
}

var SupportedModels = []ModelConfig{
	{ID: "qwen/qwen3-coder-480b-a35b-instruct-maas", DisplayName: "Qwen 3 Coder 480B (Instruct)", IsThinking: false, Region: "us-south1"},
	{ID: "qwen/qwen3-next-80b-a3b-thinking-maas", DisplayName: "Qwen 3 Next 80B (Thinking)", IsThinking: true, Region: "global"},
	{ID: "google/gemini-3.0-flash", DisplayName: "Gemini 3.0 Flash", IsThinking: true, Region: "global"},
	{ID: "google/gemini-3.0-pro", DisplayName: "Gemini 3.0 Pro", IsThinking: true, Region: "global"},
	{ID: "deepseek-ai/deepseek-r1", DisplayName: "DeepSeek R1", IsThinking: true, Region: "global"},
	{ID: "moonshot-ai/kimi-k2-thinking-maas", DisplayName: "Kimi K2 Thinking", IsThinking: true, Region: "global"},
}

func NewAgents(ctx context.Context, projectID string) ([]agent.Agent, error) {
	var agents []agent.Agent

	for _, cfg := range SupportedModels {
		// Initialize the model wrapper
		m, err := model.NewVertexLLM(ctx, projectID, cfg.Region, cfg.ID, cfg.DisplayName)
		if err != nil {
			return nil, err
		}

		instruction := "You are a helpful AI assistant."
		if cfg.IsThinking {
			instruction += " You are a thinking model that reasons before answering."
		}

		// Create the LLM agent
		ag, err := llmagent.New(llmagent.Config{
			Name:        cfg.DisplayName, // Using DisplayName as the agent name
			Model:       m,
			Instruction: instruction,
		})
		if err != nil {
			return nil, err
		}
		agents = append(agents, ag)
	}

	return agents, nil
}
