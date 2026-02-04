package mcpclient

import (
	"context"
	"encoding/json"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// loggingSession wraps an MCPSession and logs all calls and results.
type loggingSession struct {
	inner MCPSession
	name  string
}

// NewLoggingSession creates a new logging wrapper around an MCPSession.
func NewLoggingSession(inner MCPSession, name string) MCPSession {
	return &loggingSession{
		inner: inner,
		name:  name,
	}
}

func (s *loggingSession) ListResources(ctx context.Context, params *mcp.ListResourcesParams) (*mcp.ListResourcesResult, error) {
	s.logRequest("ListResources", params)
	res, err := s.inner.ListResources(ctx, params)
	s.logResponse("ListResources", res, err)
	return res, err
}

func (s *loggingSession) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	s.logRequest("CallTool", params)
	res, err := s.inner.CallTool(ctx, params)
	s.logResponse("CallTool", res, err)
	return res, err
}

func (s *loggingSession) ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	s.logRequest("ListTools", params)
	res, err := s.inner.ListTools(ctx, params)
	s.logResponse("ListTools", res, err)
	return res, err
}

func (s *loggingSession) ListPrompts(ctx context.Context, params *mcp.ListPromptsParams) (*mcp.ListPromptsResult, error) {
	s.logRequest("ListPrompts", params)
	res, err := s.inner.ListPrompts(ctx, params)
	s.logResponse("ListPrompts", res, err)
	return res, err
}

func (s *loggingSession) InitializeResult() *mcp.InitializeResult {
	return s.inner.InitializeResult()
}

func (s *loggingSession) Close() error {
	log.Printf("[DEBUG] %s: Closing session", s.name)
	return s.inner.Close()
}

func (s *loggingSession) logRequest(method string, params interface{}) {
	jsonParams, _ := json.Marshal(params)
	log.Printf("[DEBUG] %s: %s Request: %s", s.name, method, string(jsonParams))
}

func (s *loggingSession) logResponse(method string, result interface{}, err error) {
	if err != nil {
		log.Printf("[DEBUG] %s: %s Error: %v", s.name, method, err)
		return
	}
	jsonResult, _ := json.Marshal(result)
	log.Printf("[DEBUG] %s: %s Response: %s", s.name, method, string(jsonResult))
}
