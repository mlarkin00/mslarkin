package agent

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/vertexai/genai"
)

type Agent struct {
	client *genai.Client
	model  *genai.GenerativeModel
	rootDir string
}

func NewAgent(ctx context.Context, projectID, location, modelName, rootDir string) (*Agent, error) {
	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel(modelName)
	model.Tools = []*genai.Tool{clusterTool}
	model.SetTemperature(0.0) // Deterministic for tools

	return &Agent{
		client:  client,
		model:   model,
		rootDir: rootDir,
	}, nil
}

// Run executes a prompt and handles tool calls in a loop.
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
	chat := a.model.StartChat()

	send := func(msg string) (*genai.GenerateContentResponse, error) {
		return chat.SendMessage(ctx, genai.Text(msg))
	}

	res, err := send(prompt)
	if err != nil {
		return "", err
	}

	// Tool use loop
	for {
		if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
			return "No response from model", nil
		}

		part := res.Candidates[0].Content.Parts[0]

		// Check for Function Call
		if funcCall, ok := part.(genai.FunctionCall); ok {
			log.Printf("Agent calling tool: %s", funcCall.Name)

			result, err := executeTool(ctx, funcCall.Name, funcCall.Args, a.rootDir)
			if err != nil {
				// Feed error back to model?
				// For now just error out or send back error string
				// Let's send back the error as tool response
				log.Printf("Tool error: %v", err)
				res, err = chat.SendMessage(ctx, genai.FunctionResponse{
					Name: funcCall.Name,
					Response: map[string]interface{}{"error": err.Error()},
				})
			} else {
				res, err = chat.SendMessage(ctx, genai.FunctionResponse{
					Name: funcCall.Name,
					Response: result,
				})
			}

			if err != nil {
				return "", err
			}
			continue // Loop to see if model has more to say or more tools calls
		}

		// Check for text response (final answer)
		if text, ok := part.(genai.Text); ok {
			return string(text), nil
		}

		return "", fmt.Errorf("unexpected response type: %T", part)
	}
}

func (a *Agent) Close() {
	a.client.Close()
}
