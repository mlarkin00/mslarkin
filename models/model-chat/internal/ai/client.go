package ai

import (
	"context"
	"fmt"

	"encoding/json"
	"log"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
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

// ListClusters lists GKE clusters in the given project/location.
func (c *Client) ListClusters(ctx context.Context, projectID, location string) (string, error) {
	if projectID == "" {
		projectID = c.projectID
	}
	if location == "" {
		location = "-"
	}

	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to get ADC: %w", err)
	}

	svc, err := container.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return "", fmt.Errorf("failed to create container service: %w", err)
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	resp, err := svc.Projects.Locations.Clusters.List(parent).Do()
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}

	// Simplify response
	type clusterInfo struct {
		Name     string `json:"name"`
		Location string `json:"location"`
		Status   string `json:"status"`
	}
	var clusters []clusterInfo
	for _, c := range resp.Clusters {
		clusters = append(clusters, clusterInfo{
			Name:     c.Name,
			Location: c.Location,
			Status:   c.Status,
		})
	}

	data, err := json.Marshal(clusters)
	if err != nil {
		return "", err
	}
	return string(data), nil
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

	host := ""
	if region == "global" {
		host = "aiplatform.googleapis.com"
	} else {
		host = fmt.Sprintf("%s-aiplatform.googleapis.com", region)
	}

	baseURL := fmt.Sprintf("https://%s/v1beta1/projects/%s/locations/%s/endpoints/openapi", host, c.projectID, region)

	config := openai.DefaultConfig(token.AccessToken)
	config.BaseURL = baseURL

	client := openai.NewClientWithConfig(config)

	// Define tools
	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "list_clusters",
				Description: "List GKE clusters in the specified project.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"project_id": {
							Type:        jsonschema.String,
							Description: "The Google Cloud Project ID to list clusters from. Defaults to current project if not specified.",
						},
						"location": {
							Type:        jsonschema.String,
							Description: "The location (region or zone) to list clusters from. Use '-' for all locations.",
						},
					},
				},
			},
		},
	}

	// Tool loop
	maxTurns := 5
	for i := 0; i < maxTurns; i++ {
		resp, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model:    modelID,
				Messages: messages,
				Tools:    tools,
			},
		)
		if err != nil {
			fmt.Printf("Chat completion error for model %s in region %s: %v\n", modelID, region, err)
			return nil, fmt.Errorf("chat completion failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("no response choices")
		}

		msg := resp.Choices[0].Message
		messages = append(messages, msg)

		if len(msg.ToolCalls) > 0 {
			log.Printf("Tool calls detected: %d", len(msg.ToolCalls))
			for _, toolCall := range msg.ToolCalls {
				if toolCall.Function.Name == "list_clusters" {
					log.Printf("Executing tool: %s", toolCall.Function.Name)
					var args struct {
						ProjectID string `json:"project_id"`
						Location  string `json:"location"`
					}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						log.Printf("Failed to unmarshal args: %v", err)
						continue
					}

					result, err := c.ListClusters(ctx, args.ProjectID, args.Location)
					content := result
					if err != nil {
						content = fmt.Sprintf("Error: %v", err)
					}

					messages = append(messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    content,
						ToolCallID: toolCall.ID,
					})
				}
			}
			continue // Next turn
		}

		return &ChatResponse{
			Content: msg.Content,
			Usage:   resp.Usage,
		}, nil
	}

	return nil, fmt.Errorf("max turns exceeded")
}
