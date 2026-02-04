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

// ChatService handles the interaction with the Gemini agent.
type ChatService struct {
	Runner *runner.Runner
}

// NewChatService creates a new ChatService.
// It initializes the Gemini model, tool definitions, and the agent runner.
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
		Instruction: buildInstruction(ctx, mcpClient),
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

// Chat sends a message to the agent and returns a stream of events.
// It maintains session context using the sessionID.
func (s *ChatService) Chat(ctx context.Context, sessionID string, message string) (iter.Seq2[*session.Event, error], error) {
	content := &genai.Content{
		Parts: []*genai.Part{
			{Text: message},
		},
	}
	// userID can be "user" or similar.
	return s.Runner.Run(ctx, "user", sessionID, content, agent.RunConfig{}), nil
}

// buildInstruction constructs the system instruction for the agent.
// It incorporates available MCP tools, prompts, and resources into the prompt context.
func buildInstruction(ctx context.Context, client *mcpclient.MCPClient) string {
	baseInstruction := `You are a helpful AI assistant for the GKE Demo.
You MUST output your response as a JSON object strictly following the A2UI format.
Do NOT wrap the JSON in markdown code blocks.
When asked to list clusters or workloads, use the provided tools.
The tools return A2UI components; incorporate them into your response structure.
If the user asks a general question, wrap your text answer in an A2UI 'text' component.
Structure your response as a root 'container' component.`

	// Fetch MCP Context
	var sb string
	sb += baseInstruction + "\n\n"
	sb += "SYSTEM CONTEXT FROM MCP:\n"

	// Tools
	tools, err := client.ListTools(ctx)
	if err == nil && len(tools) > 0 {
		sb += "Available MCP Tools (use 'call_mcp_tool' to execute):\n"
		for _, t := range tools {
			sb += fmt.Sprintf("- %s: %s\n", t.Name, t.Description)
            // We could add schema here if needed, but keeping it brief for now.
            // Actually, for generic call, schema is useful.
            // Only adding truncated schema or just name/desc to save tokens?
            // "use 'call_mcp_tool' with 'name' and 'arguments' matching the tool."
		}
		sb += "\n"
	}

	// Prompts
	prompts, err := client.ListPrompts(ctx)
	if err == nil && len(prompts) > 0 {
		sb += "Available Prompts (use these for context or starting points):\n"
		for _, p := range prompts {
			sb += fmt.Sprintf("- %s: %s\n", p.Name, p.Description)
		}
		sb += "\n"
	}

	// Resources
	resources, err := client.ListResources(ctx)
	if err == nil && len(resources) > 0 {
		sb += "Available Resources (read these if relevant):\n"
		for _, r := range resources {
			sb += fmt.Sprintf("- %s (MIME: %s): %s\n", r.URI, r.MIMEType, r.Name)
		}
		sb += "\n"
	}

	// Server Instructions
	// Collect instructions from all servers
	var instructions []string
	if client.OneMCPSession != nil {
		initRes := client.OneMCPSession.InitializeResult()
		if initRes != nil && initRes.Instructions != "" {
			instructions = append(instructions, fmt.Sprintf("OneMCP Instructions: %s", initRes.Instructions))
		}
	}
	if client.OSSMCPSession != nil {
		initRes := client.OSSMCPSession.InitializeResult()
		if initRes != nil && initRes.Instructions != "" {
			instructions = append(instructions, fmt.Sprintf("OSSMCP Instructions: %s", initRes.Instructions))
		}
	}

	if len(instructions) > 0 {
		sb += "Server Instructions:\n"
		for _, instr := range instructions {
			sb += instr + "\n"
		}
		sb += "\n"
	}

	return sb
}
