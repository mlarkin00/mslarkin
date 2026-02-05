package ai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/oauth2/google"
)

const (
	ProjectID      = "mslarkin-ext"
	DefaultRegion  = "us-central1"
	FallbackRegion = "us-central1"
)

// Model defines a supported model
type Model struct {
	ID          string
	DisplayName string
	IsThinking  bool   // Some models might support thinking process
	Region      string // Optional: specific region for this model
}

var SupportedModels = []Model{
	{ID: "qwen/qwen3-next-80b-a3b-thinking-maas", DisplayName: "Qwen 3 Next 80B (Thinking)", IsThinking: true, Region: "global"},
	{ID: "qwen/qwen3-coder-480b-a35b-instruct-maas", DisplayName: "Qwen 3 Coder 480B (Instruct)", IsThinking: false, Region: "us-south1"},
	{ID: "zai-org/glm-4.7-maas", DisplayName: "GLM 4.7", IsThinking: false, Region: "global"},
	{ID: "minimaxai/minimax-m2-maas", DisplayName: "Minimax M2", IsThinking: false, Region: "global"},
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

// ChatResponse encapsulates the model response and metadata
type ChatResponse struct {
	Content        string
	Usage          openai.Usage
	Thinking       string // If available
	CachedTokens   int    // If available separately
}

func (c *Client) Chat(ctx context.Context, modelID string, messages []openai.ChatCompletionMessage) (*ChatResponse, error) {
	// Determine region
	region := c.region
	for _, m := range SupportedModels {
		if m.ID == modelID && m.Region != "" {
			region = m.Region
			break
		}
	}

	// Get ADC token
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to get application default credentials: %w", err)
	}
	token, err := creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	// Determine endpoint based on model (some might be regional restricted, we assume strictly us-west1 or fallback for now)
	// For simplicity, we use the client's configured region.
	// Vertex AI OpenAI-compatible endpoint:
	// https://{REGION}-aiplatform.googleapis.com/v1beta1/projects/{PROJECT}/locations/{REGION}/endpoints/openapi

	host := ""
	if region == "global" {
		host = "aiplatform.googleapis.com" // Use generic host for global location
	} else {
		host = fmt.Sprintf("%s-aiplatform.googleapis.com", region)
	}

	baseURL := fmt.Sprintf("https://%s/v1beta1/projects/%s/locations/%s/endpoints/openapi", host, c.projectID, region)

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
		fmt.Printf("Chat completion error for model %s in region %s: %v\n", modelID, region, err)
		// Fallback logic could go here (e.g., try us-central1)
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices")
	}

	return &ChatResponse{
		Content: resp.Choices[0].Message.Content,
		Usage:   resp.Usage,
	}, nil
}
