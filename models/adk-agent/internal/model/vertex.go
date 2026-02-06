package model

import (
	"context"
	"fmt"
	"io"
	"iter"
	"net/http"

	"github.com/sashabaranov/go-openai"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

type VertexLLM struct {
	client  *openai.Client
	modelID string
	name    string
}

func NewVertexLLM(ctx context.Context, projectID, region, modelID, name string) (*VertexLLM, error) {
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %w", err)
	}

	host := region
	if region == "global" {
		host = "aiplatform.googleapis.com"
	} else {
		host = fmt.Sprintf("%s-aiplatform.googleapis.com", region)
	}

	baseURL := fmt.Sprintf("https://%s/v1beta1/projects/%s/locations/%s/endpoints/openapi",
		host, projectID, region)

	config := openai.DefaultConfig("") // API Key is empty as we use OAuth2
	config.BaseURL = baseURL
	config.HTTPClient = &http.Client{
		Transport: &oauthTransport{
			base:   http.DefaultTransport,
			source: creds.TokenSource,
		},
	}

	client := openai.NewClientWithConfig(config)
	return &VertexLLM{
		client:  client,
		modelID: modelID,
		name:    name,
	}, nil
}

func (m *VertexLLM) Name() string { return m.name }

func (m *VertexLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		messages := convertToOpenAIMessages(req.Contents)

		// Basic stream support
		if stream {
			streamReq := openai.ChatCompletionRequest{
				Model:    m.modelID,
				Messages: messages,
				Stream:   true,
			}
			s, err := m.client.CreateChatCompletionStream(ctx, streamReq)
			if err != nil {
				yield(nil, err)
				return
			}
			defer s.Close()

			for {
				resp, err := s.Recv()
				if err != nil {
					if err != io.EOF {
						yield(nil, err)
					}
					return
				}

				content := resp.Choices[0].Delta.Content
				// Map to ADK response
				yield(&model.LLMResponse{
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: content}},
						Role:  "model",
					},
					TurnComplete: false,
				}, nil)
			}
		} else {
			resp, err := m.client.CreateChatCompletion(
				ctx,
				openai.ChatCompletionRequest{
					Model:    m.modelID,
					Messages: messages,
					Stream:   false,
				},
			)
			if err != nil {
				yield(nil, err)
				return
			}

			yield(&model.LLMResponse{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: resp.Choices[0].Message.Content}},
					Role:  "model",
				},
				TurnComplete: true,
			}, nil)
		}
	}
}

// Convert ADK GenAI content to OpenAI messages
func convertToOpenAIMessages(contents []*genai.Content) []openai.ChatCompletionMessage {
	var messages []openai.ChatCompletionMessage
	for _, c := range contents {
		role := openai.ChatMessageRoleUser
		if c.Role == "model" {
			role = openai.ChatMessageRoleAssistant
		} else if c.Role == "system" {
			role = openai.ChatMessageRoleSystem
		}

		var content string
		for _, p := range c.Parts {
			if p.Text != "" {
				content += p.Text
			}
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: content,
		})
	}
	return messages
}

type oauthTransport struct {
	base   http.RoundTripper
	source oauth2.TokenSource
}

func (t *oauthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.source.Token()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return t.base.RoundTrip(req)
}
