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
)

type ChatService struct {
	Runner *runner.Runner
}

func NewChatService(ctx context.Context, projectID, location, modelName string) (*ChatService, error) {
	if modelName == "" {
		modelName = "gemini-1.5-pro-002"
	}

	model, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini model: %w", err)
	}

	myAgent, err := llmagent.New(llmagent.Config{
		Name:        "GKEAssistant",
		Description: "A helpful AI assistant for the GKE Demo.",
		Model:       model,
		Instruction: "You are a helpful AI assistant for the GKE Demo. You can answer questions about GKE and the demo environment.",
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
