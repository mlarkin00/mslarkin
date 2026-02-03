package chat

import (
	"context"
	"fmt"
	"iter"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
	"k8s-status-backend/mcpclient"
	"k8s-status-backend/tools"
)

type ChatService struct {
	Runner *runner.Runner
}

func NewChatService(ctx context.Context, projectID, location, modelName string, mcpClient *mcpclient.MCPClient) (*ChatService, error) {
	if modelName == "" {
		modelName = "gemini-1.5-pro-002"
	}

	model, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini model: %w", err)
	}

	k8sTools := tools.NewK8sTools(mcpClient)

	myAgent, err := llmagent.New(llmagent.Config{
		Name:        "GKEAssistant",
		Description: "A helpful AI assistant for the GKE Demo.",
		Model:       model,
		Tools:       k8sTools.GetTools(),
		Instruction: `You are a helpful AI assistant for the GKE Demo.
You MUST output your response as a JSON object strictly following the A2UI format.
Do NOT wrap the JSON in markdown code blocks.
When asked to list clusters or workloads, use the provided tools.
The tools return A2UI components; incorporate them into your response structure.
If the user asks a general question, wrap your text answer in an A2UI 'text' component.
Structure your response as a root 'container' component.`,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	r, err := runner.New(runner.Config{
		Agent:          myAgent,
		SessionService: session.InMemoryService(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}

	return &ChatService{Runner: r}, nil
}

func (s *ChatService) Chat(ctx context.Context, sessionID string, message string) (iter.Seq2[*session.Event, error], error) {
	content := &genai.Content{
		Parts: []*genai.Part{
			{Text: message},
		},
	}
	// userID can be "user" or similar.
	return s.Runner.Run(ctx, "user", sessionID, content, agent.RunConfig{}), nil
}
