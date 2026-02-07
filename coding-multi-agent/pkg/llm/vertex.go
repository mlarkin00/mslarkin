package llm

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type VertexClient struct {
	client *genai.Client
	projectID string
	location string
}

func NewVertexClient(ctx context.Context, projectID, location string) (*VertexClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, err
	}
	return &VertexClient{
		client:    client,
		projectID: projectID,
		location:  location,
	}, nil
}

func (v *VertexClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	// This is a simplified implementation.
	// Real implementation would handle parameters, history, etc.

	resp, err := v.client.Models.GenerateContent(ctx, req.ModelID, genai.Text(req.Prompt), nil)
	if err != nil {
		return nil, err
	}

	// Extract content
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}

	// Assuming first part is text for now
	content := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Text != "" {
			content += part.Text
		}
	}

	usage := TokenUsage{}
	if resp.UsageMetadata != nil {
		usage.PromptTokens = int(resp.UsageMetadata.PromptTokenCount)
		usage.CompletionTokens = int(resp.UsageMetadata.CandidatesTokenCount)
		usage.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
	}

	return &GenerateResponse{
		Content: content,
		Usage:   usage,
	}, nil
}
