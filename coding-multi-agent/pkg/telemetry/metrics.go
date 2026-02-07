package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "agent_request_duration_seconds",
			Help: "Duration of agent requests in seconds.",
		},
		[]string{"agent_type", "status"},
	)

	TokenUsage = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_token_usage_total",
			Help: "Total number of tokens used by agents.",
		},
		[]string{"agent_type", "model_id", "token_type"}, // token_type: prompt, completion
	)

	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_requests_total",
			Help: "Total number of agent requests.",
		},
		[]string{"agent_type", "status"},
	)
)

func InitMetrics() {
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(TokenUsage)
	prometheus.MustRegister(RequestCount)
}
