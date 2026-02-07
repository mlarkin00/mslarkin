package llm

import (
	"context"
	"fmt"
)

type MockClient struct {
	Responses map[string]string // Simple map of prompt suffix -> response
}

func NewMockClient() *MockClient {
	return &MockClient{
		Responses: make(map[string]string),
	}
}

func (m *MockClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	// Simple mock logic
	return &GenerateResponse{
		Content: fmt.Sprintf("Mock response for: %s", req.Prompt),
		Usage: TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}
