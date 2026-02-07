package planning

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mslarkin/coding-multi-agent/pkg/agent"
	"github.com/mslarkin/coding-multi-agent/pkg/config"
	"github.com/mslarkin/coding-multi-agent/pkg/llm"
)

type PlanningAgent struct {
	ModelConfig config.ModelConfig
	LLMClient   llm.Client
	// Configuration for downstream agents
	CodeAgentURL   string
	OpsAgentURL    string
	DesignAgentURL string
}

func NewPlanningAgent(cfg config.ModelConfig, llmClient llm.Client) *PlanningAgent {
	// Defaults for internal DNS
	return &PlanningAgent{
		ModelConfig:    cfg,
		LLMClient:      llmClient,
		CodeAgentURL:   "http://agent-code-primary:8080",
		OpsAgentURL:    "http://agent-ops:8080",
		DesignAgentURL: "http://agent-design:8080",
	}
}

func (p *PlanningAgent) Process(ctx context.Context, input agent.AgentInput) (*agent.AgentResponse, error) {
	// 1. Analyze Intent (Simple keyword matching for now, or use LLM)
	// In a real system, we'd use the LLM to decide the plan.

	targetURL := ""
	taskLower := strings.ToLower(input.Task)
	if strings.Contains(taskLower, "code") || strings.Contains(taskLower, "implement") {
		targetURL = p.CodeAgentURL
	} else if strings.Contains(taskLower, "deploy") || strings.Contains(taskLower, "ops") {
		targetURL = p.OpsAgentURL
	} else {
		// Default to Design or handle directly
		targetURL = p.DesignAgentURL
	}

	// 2. Call Downstream Agent
	return p.callRemoteAgent(ctx, targetURL, input)
}

func (p *PlanningAgent) StreamProcess(ctx context.Context, input agent.AgentInput, outputChan chan<- agent.AgentStreamUpdate) error {
	// 1. Analyze Intent
	outputChan <- agent.AgentStreamUpdate{Step: "Planning", PartialContent: "Analyzing request..."}

	targetURL := ""
	taskLower := strings.ToLower(input.Task)
	if strings.Contains(taskLower, "code") || strings.Contains(taskLower, "implement") {
		targetURL = p.CodeAgentURL
	} else if strings.Contains(taskLower, "deploy") || strings.Contains(taskLower, "ops") {
		targetURL = p.OpsAgentURL
	} else {
		targetURL = p.DesignAgentURL
	}

	outputChan <- agent.AgentStreamUpdate{Step: "Delegating", PartialContent: fmt.Sprintf("Delegating task to %s...", targetURL)}

	// 2. Call Downstream Agent
	resp, err := p.callRemoteAgent(ctx, targetURL, input)
	if err != nil {
		return err
	}

	// 3. Send final response
	outputChan <- agent.AgentStreamUpdate{Step: "Completed", PartialContent: resp.Content}
	return nil
}

func (p *PlanningAgent) callRemoteAgent(ctx context.Context, url string, input agent.AgentInput) (*agent.AgentResponse, error) {
	jsonBody, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url+"/process", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote agent returned status %d", resp.StatusCode)
	}

	var agentResp agent.AgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		return nil, err
	}

	return &agentResp, nil
}
