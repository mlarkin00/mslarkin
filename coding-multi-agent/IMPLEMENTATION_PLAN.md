# Implementation Plan: mslarkin-ext Coding Agent Ecosystem

## **Overview**
The goal is to build a distributed multi-agent system on GKE where a **Planning Agent** orchestrates tasks by delegating to specialized agents (**Code**, **Design**, **Ops**, **Validation**, **Review**). All agents share a common Go codebase but run as distinct services.

### **Core Principles**
*   **Modularity**: Agents are independent services with clear responsibilities.
*   **Scalability**: Stateless design (where possible) to allow horizontal scaling.
*   **Observability**: End-to-end tracing and metrics.
*   **Reliability**: Robust error handling, retries, and circuit breakers.

## **Architecture & Design**
1.  **Error Handling Strategy**:
    *   Define consistent error types (e.g., `pkg/errors` with codes).
    *   Implement circuit breaker patterns for inter-agent communication (using `gobreaker` or similar).
    *   Add retry logic with exponential backoff for all external service calls (Vertex AI, MCP).

2.  **Security Considerations**:
    *   **Authentication/Authorization**: Implement mTLS for service-to-service communication (or use a mesh like Istio if available, otherwise strict network policies).
    *   **Input Validation**: Strict validation and sanitization for all agent inputs to prevent injection attacks.

3.  **Scalability Planning**:
    *   **Async Communication**: Prefer async message passing (Pub/Sub) for inter-agent communication where immediate consistency isn't required.
    *   **Rate Limiting**: Implement rate limiting for external API calls to avoid quota exhaustion.
    *   **Autoscaling**: Configure HPA (Horizontal Pod Autoscaler) based on CPU/Memory/Custom metrics.

## **Phase 1: Project Skeleton & Foundation**

1.  **Initialize/Refactor Go Module**:
    *   Ensure `go.mod` is set up in `models/adk-agent`.
    *   **Dependency Management**:
        *   Use `go workspaces` if managing multiple modules.
        *   Pin versions for critical dependencies.
        *   Regularly run `go mod tidy`.
    *   Add dependencies:
        *   `github.com/gin-gonic/gin` (HTTP Server).
        *   `google.golang.org/adk` (Google ADK).
        *   `go.opentelemetry.io/otel` (OpenTelemetry).
        *   `github.com/redis/go-redis/v9` (Redis client).
        *   `github.com/prometheus/client_golang/prometheus` (Metrics).
        *   `github.com/sony/gobreaker` (Circuit Breaker).

2.  **Define Core Interfaces (`pkg/agent`)**:
    *   Create `pkg/agent/types.go`: Define `AgentInput`, `AgentResponse`, `TokenMetrics`, `StatusEnum`.
    *   Create `pkg/agent/interface.go`: Define the `Agent` interface with `Process` and `StreamProcess` methods.
    *   **Context Usage**: Ensure `context.Context` is the first argument for all methods for tracing and cancellation.

3.  **Configuration Loading (`pkg/config`)**:
    *   Create `pkg/config/config.go`.
    *   **Validation**: Add validation for required configuration values.
    *   **Defaults**: Implement sensible default values.
    *   Support dynamic config updates (e.g., via config watching or restart-on-change).

4.  **Telemetry Setup (`pkg/telemetry`)**:
    *   Create `pkg/telemetry/trace.go`: Configure OpenTelemetry for tracing.
        *   Add span attributes for high-cardinality data (e.g., `agent_id`, `request_id`).
        *   Configure trace sampling.
    *   Create `pkg/telemetry/metrics.go`: Configure Prometheus metrics.
        *   Standard metrics: Request count, latency, error rate.
        *   Custom metrics: Token usage, model latency.

## **Phase 2: Core Agent Logic & MCP**

5.  **Base Agent Implementation**:
    *   Implement a `BaseAgent` struct in `internal/agent` that satisfies the `Agent` interface.
    *   Integrate `google.golang.org/adk` to handle Vertex AI interactions.
    *   Implement consistent log levels (Info, Debug, Error) using structured logging (e.g., `slog`).
    *   Add **Health Checks**: Implement `/healthz` and `/readyz` endpoints.

6.  **MCP Client (`pkg/mcp`)**:
    *   Create `pkg/mcp/client.go`.
    *   Implement a client to call MCP tools.
    *   **Connection Pooling**: manage connections efficiently.
    *   **Timeouts**: Configurable timeouts for tool calls.
    *   **Mocking**: Interface-based design to allow mock implementations for testing.
    *   **Sidecar/Remote Support**: Handle logic for both local sidecars and remote MCP servers (auth headers).

7.  **Redis State Management (`pkg/state`)**:
    *   Create `pkg/state/redis.go`.
    *   **TTL**: Set Expiration for session data to avoid stale data accumulation.
    *   **Clustering**: Support Redis Cluster configuration for HA.
    *   **Backup**: Document strategy for RDB/AOF if persistence is critical.

## **Phase 3: Specific Agent Implementations**
*Suggestion: implement a plugin architecture or registration pattern in `main.go` to easily add new agents.*

8.  **Planning Agent (`internal/planning`)**:
    *   **Router**: Analyze user input (using `deepseek-ai/deepseek-r1-0528-maas` [Region: `us-central1`]).
    *   **Streaming**: SSE for real-time updates.

9.  **Code Agent - Primary (`internal/code/primary.go`)**:
    *   Default Model: `qwen/qwen3-coder-480b-a35b-instruct-maas` [Region: `us-south1`].
    *   Logic: Generate 3 code options.

10. **Code Agent - Secondary (`internal/code/secondary.go`)**:
    *   Default Model: `moonshotai/kimi-k2-thinking-maas` [Region: `global`].
    *   Logic: Critique/Select best option.

11. **Validation Agent (`internal/validation`)**:
    *   Default Model: `qwen/qwen3-next-80b-a3b-thinking-maas` [Region: `global`].
    *   Logic: Validate selected option.

12. **Review Agent (`internal/review`)**:
    *   Default Model: `google/gemini-3-pro-preview` [Region: `global`].
    *   Logic: Final arbiter. Verify with external tools.

13. **Design & Ops Agents**:
    *   Use `google/gemini-3-pro-preview` and `google/gemini-3-flash-preview`.
    *   Ops Agent uses `gke-onemcp` / `gke-oss-mcp`.

## **Phase 4: API & Entrypoint**

14. **Main Entrypoint (`cmd/agent/main.go`)**:
    *   Parse Environment Variables (`AGENT_ROLE`, `AGENT_MODEL`).
    *   **Graceful Shutdown**: Handle SIGTERM/SIGINT to flush traces and close connections.
    *   Initialize specific agent implementation.

15. **HTTP Server & Routing**:
    *   **Middleware**:
        *   Request/Response Logging.
        *   Panic Recovery.
        *   Request ID generation.
        *   **CORS**: Configure for web clients.
    *   **API Versioning**: Prefix routes with `/v1`.
    *   **Validation**: Add request body validation middleware.

## **Phase 5: Deployment & Infrastructure**

16. **Kubernetes Manifests (`k8s/`)**:
    *   **Resource Management**: Add `resources.requests` and `resources.limits` for all containers.
    *   **Probes**: Add `livenessProbe` and `readinessProbe` to all deployments.
    *   **Availability**: Add `PodDisruptionBudget`.
    *   **Services**: Use K8s Services for discovery.
    *   **Helm**: Consider packaging as a Helm chart for easier management (optional but recommended).

17. **Sidecar Configuration**:
    *   Ensure `gke-oss-mcp` sidecar is correctly configured for agents needing it.

18. **Secrets Management**:
    *   Avoid hardcoding secrets. Use K8s Secrets.
    *   Consider External Secrets Operator (ESO) to sync with Google Secret Manager.
    *   **Rotation**: Plan for secret rotation (API keys).

## **Testing Strategy**
*   **Unit Tests**: High coverage for business logic (`go test -race ./...`).
*   **Integration Tests**: Test agent-to-agent communication (mocking the transport).
*   **Contract Tests**: Verify API contracts between agents.
*   **Performance/Load**: Use k6 or similar to test throughput and latency under load.
*   **Chaos Testing**: Simulate pod failures/network partitions to verify resilience.

## **Documentation & Maintenance**
1.  **API Documentation**: Generate OpenAPI/Swagger spec.
2.  **Runbooks**: Create guides for:
    *   Debugging agent failures.
    *   Rolling back deployments.
    *   Adding a new agent type.
3.  **Architecture Diagrams**: Visualize the interaction flow.
4.  **CI/CD**:
    *   Automated testing pipeline (GitHub Actions / Cloud Build).
    *   Container image scanning (Trivy/Clair).

## **Minor Technical Notes**
*   **Context**: Use `context.Context` everywhere.
*   **Shared Utils**: Create a shared library for common utilities (logging, http client wrappers) to avoid duplication.
*   **Golden Paths**: Define "golden paths" for common successful workflows to streamline debugging.
