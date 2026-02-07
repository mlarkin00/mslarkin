package agent

import "time"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AgentInput struct {
	Task        string            `json:"task"`
	Context     []Message         `json:"context"` // Chat history
	Artifacts   map[string]string `json:"artifacts"` // Previous code/manifests
	Constraints []string          `json:"constraints"`
}

type TokenMetrics struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type StatusEnum string

const (
	StatusPendingReview StatusEnum = "PENDING_REVIEW"
	StatusApproved      StatusEnum = "APPROVED"
	StatusRejected      StatusEnum = "REJECTED"
	StatusCompleted     StatusEnum = "COMPLETED"
)

type AgentResponse struct {
	Content     string        `json:"content"`
	Rationale   string        `json:"rationale"` // For thinking models
	TokenUsage  TokenMetrics  `json:"token_usage"`
	Latency     time.Duration `json:"latency"`
	Status      StatusEnum    `json:"status"`
}

type AgentStreamUpdate struct {
	PartialContent string
	Step           string // e.g., "Drafting Code", "Running Validation"
}
