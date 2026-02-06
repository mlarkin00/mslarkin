# Implementation Plan: mslarkin-ext Coding Agent Ecosystem

## **Overview**
The goal is to build a distributed multi-agent system on GKE where a **Planning Agent** orchestrates tasks by delegating to specialized agents (**Code**, **Design**, **Ops**, **Validation**, **Review**). All agents share a common Go codebase but run as distinct services.

## **Phase 1: Project Skeleton & Foundation**

1.  **Initialize/Refactor Go Module**:
    *   Ensure `go.mod` is set up in `models/adk-agent`.
    *   Add necessary dependencies:
        *   `github.com/gin-gonic/gin` or standard `net/http` (Standard lib preferred for simplicity unless complex routing needed).
        *   `google.golang.org/adk` (Google ADK).
        *   `go.opentelemetry.io/otel` (OpenTelemetry).
        *   `github.com/redis/go-redis/v9` (Redis client).
        *   `github.com/sashabaranov/go-openai` (for OpenAI compatibility types, optional but helpful).

2.  **Define Core Interfaces (`pkg/agent`)**:
    *   Create `pkg/agent/types.go`: Define `AgentInput`, `AgentResponse`, `TokenMetrics`, `StatusEnum`.
    *   Create `pkg/agent/interface.go`: Define the `Agent` interface with `Process` and `StreamProcess` methods.

3.  **Configuration Loading (`pkg/config`)**:
    *   Create `pkg/config/config.go`.
    *   Implement loading from `config.yaml` (embed or read from file) and environment variables (`AGENT_ROLE`, `AGENT_MODEL`, `PORT`, `REDIS_HOST`, `PROJECT_ID`).
    *   Define the `Config` struct to hold model IDs, locations, and MCP endpoints.

4.  **Telemetry Setup (`pkg/telemetry`)**:
    *   Create `pkg/telemetry/trace.go`.
    *   Implement OpenTelemetry setup to export traces to Google Cloud Trace.
    *   Ensure a `Shutdown` function is provided to flush traces on exit.

## **Phase 2: Core Agent Logic & MCP**

5.  **Base Agent Implementation**:
    *   Implement a `BaseAgent` struct in `internal/agent` that satisfies the `Agent` interface.
    *   Integrate `google.golang.org/adk` to handle Vertex AI interactions.
    *   Implement the `Process` method to send prompts to Vertex AI and parse responses.

6.  **MCP Client (`pkg/mcp`)**:
    *   Create `pkg/mcp/client.go`.
    *   Implement a client that can call MCP tools.
    *   **Sidecar Support**: Handle `http://localhost:8080/mcp`.
    *   **Remote Support**:
        *   Handle `Context7` (Header: `CONTEXT7_API_KEY`).
        *   Handle `docs-onemcp` & `gke-onemcp` (Google Auth: OIDC/ADC).
    *   Ensure the client can list tools and call tools.

7.  **Redis State Management (`pkg/state`)**:
    *   Create `pkg/state/redis.go`.
    *   Implement functions to `SaveSession`, `GetSession`, `SaveArtifact`, `GetArtifact`.

## **Phase 3: Specific Agent Implementations**

8.  **Planning Agent (`internal/planning`)**:
    *   Implement the orchestration logic.
    *   **Router**: Analyze user input (using `deepseek-ai/deepseek-r1-0528-maas` [Region: `us-central1`]) to decide which pipeline to trigger.
    *   **Streaming**: Implement SSE (Server-Sent Events) to stream status updates to the client.

9.  **Code Agent - Primary (`internal/code/primary.go`)**:
    *   Default Model: `qwen/qwen3-coder-480b-a35b-instruct-maas` [Region: `us-south1`].
    *   Logic: Generate 3 distinct code options based on input.

10. **Code Agent - Secondary (`internal/code/secondary.go`)**:
    *   Default Model: `moonshotai/kimi-k2-thinking-maas` [Region: `global`].
    *   Logic: Review the 3 options from Primary. Critique or select the best one.

11. **Validation Agent (`internal/validation`)**:
    *   Default Model: `qwen/qwen3-next-80b-a3b-thinking-maas` [Region: `global`].
    *   Logic: Validate the selected option. If valid, approve. If not, reject or escalate.

12. **Review Agent (`internal/review`)**:
    *   Default Model: `google/gemini-3-pro-preview` [Region: `global`].
    *   Logic: Final arbiter in case of conflict. Uses `Context7`, `docs-onemcp` and web search to verify.

13. **Design & Ops Agents**:
    *   Implement `internal/design` and `internal/ops` using their respective models (`google/gemini-3-pro-preview` [Region: `global`], `google/gemini-3-flash-preview` [Region: `global`]).
    *   Ops Agent must prefer `gke-onemcp` tool, with `gke-oss-mcp` (https://github.com/GoogleCloudPlatform/gke-mcp) as a fallback.

## **Phase 4: API & Entrypoint**

14. **Main Entrypoint (`cmd/agent/main.go`)**:
    *   Parse `AGENT_ROLE` env var.
    *   Parse `AGENT_MODEL` env var.
    *   Initialize the specific agent implementation based on the role and selected model.
    *   Start the HTTP server.

15. **HTTP Server & Routing**:
    *   **Planning Agent Mode**:
        *   Expose `POST /v1/chat/completions`.
        *   Handle OpenAI-compatible request body.
        *   Orchestrate calls to other agents via HTTP.
    *   **Worker Agent Mode (Code, Ops, etc.)**:
        *   Expose `POST /process` (accepts `AgentInput`, returns `AgentResponse`).

## **Phase 5: Kubernetes Manifests & Infrastructure**

16. **Kubernetes Manifests (`k8s/`)**:
    *   **Common**: `configmap.yaml` (if needed for non-sensitive config).
    *   **Deployments**: Create a base deployment template and generate:
        *   `deployment-planning.yaml`
        *   `deployment-code-primary.yaml`
        *   `deployment-code-secondary.yaml`
        *   `deployment-ops.yaml`
        *   `deployment-design.yaml`
        *   `deployment-validation.yaml`
        *   `deployment-review.yaml`
    *   **Services**:
        *   `service-planning.yaml` (Type: NodePort or ClusterIP, pointed to by Ingress).
        *   `service-internal.yaml` (Headless or ClusterIP for internal A2A).
    *   **Ingress**:
        *   `ingress.yaml` (Expose Planning Agent via `mslarkin-ext-lb`).

17. **Sidecar Configuration**:
    *   Ensure the `deployment-ops.yaml` and others that need local MCP include the `gke-oss-mcp` sidecar container definition.

18. **Setup Script (`setup.sh`)**:
    *   Script to apply all manifests: `kubectl apply -f k8s/`.
    *   Instructions to create secrets for `CONTEXT7_API_KEY`.

## **Verification Plan**
*   **Unit Tests**: Run `go test ./...`
*   **Local Test**: Run `main.go` locally with `AGENT_ROLE=planning` and mock other agents.
*   **Integration**: Deploy to GKE `mslarkin-ext` and use `curl` to hit the external endpoint.
