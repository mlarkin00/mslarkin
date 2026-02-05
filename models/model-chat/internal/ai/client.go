package ai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/oauth2/google"
)

const (
	ProjectID      = "mslarkin-ext"
	DefaultRegion  = "us-west1"
	FallbackRegion = "us-central1"
)

// Model defines a supported model
type Model struct {
	ID          string
	DisplayName string
	IsThinking  bool // Some models might support thinking process
}

var SupportedModels = []Model{
	{ID: "qwen/qwen3-next-80b-a3b-thinking-maas", DisplayName: "Qwen 3 Next 80B (Thinking)", IsThinking: true},
	{ID: "qwen/qwen3-coder-480b-a35b-instruct-maas", DisplayName: "Qwen 3 Coder 480B (Instruct)", IsThinking: false},
	{ID: "publishers/zai-org/models/glm-4.7:GLM-4.7-FP8", DisplayName: "GLM 4.7", IsThinking: false},
	{ID: "publishers/minimaxai/models/minimax-m2-maas", DisplayName: "Minimax M2", IsThinking: false},
}

type Client struct {
	region    string
	projectID string
}

func NewClient() *Client {
	return &Client{
		region:    DefaultRegion,
		projectID: ProjectID,
	}
}

func (c *Client) Chat(ctx context.Context, modelID string, messages []openai.ChatCompletionMessage) (string, error) {
	// Get ADC token
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to get application default credentials: %w", err)
	}
	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	// Determine endpoint based on model (some might be regional restricted, we assume strictly us-west1 or fallback for now)
	// For simplicity, we use the client's configured region.
	// Vertex AI OpenAI-compatible endpoint:
	// https://{REGION}-aiplatform.googleapis.com/v1beta1/projects/{PROJECT}/locations/{REGION}/endpoints/openapi
	baseURL := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1beta1/projects/%s/locations/%s/endpoints/openapi", c.region, c.projectID, c.region)

	config := openai.DefaultConfig(token.AccessToken)
	config.BaseURL = baseURL

	// Vertex AI doesn't always need the "model" field in the request body if the endpoint is model-specific,
	// BUT for the "openapi" endpoint which aggregates, we DO need the model ID.
	// The model ID usually needs to be formatted correctly if it's a publisher model.
	// However, for MaaS, the ID passed in the API is typically the Model Garden ID.

	client := openai.NewClientWithConfig(config)

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    modelID,
			Messages: messages,
			Stream:   false, // Start with non-streaming for simplicity
		},
	)
	if err != nil {
		// Fallback logic could go here (e.g., try us-central1)
		return "", fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices")
	}

	return resp.Choices[0].Message.Content, nil
}
