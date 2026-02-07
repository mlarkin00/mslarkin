package agent

import (
	"context"
)

type Agent interface {
	Process(ctx context.Context, input AgentInput) (*AgentResponse, error)
	// StreamProcess allows real-time token streaming from the agent back to the orchestrator
	StreamProcess(ctx context.Context, input AgentInput, outputChan chan<- AgentStreamUpdate) error
}
