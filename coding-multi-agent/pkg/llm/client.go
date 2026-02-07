package llm

import (
	"context"
)

type GenerateRequest struct {
	ModelID     string
	Prompt      string
	IsThinking  bool
	Temperature float32
}

type GenerateResponse struct {
	Content string
	Usage   TokenUsage
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Client interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}
