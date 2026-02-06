# **Detailed Technical Design: mslarkin-ext Agent Ecosystem**

## **1\. System Architecture**

### **1.1 High-Level Topology**

The application will be deployed as a distributed system on Google Kubernetes Engine (GKE).

* **External Entry Point**: mslarkin-ext-lb (Global HTTPS Load Balancer).
  * **Authentication**: Protected by **Identity-Aware Proxy (IAP)** or API Key validation middleware.
  * **Target**: **Planning Agent** Service.
* **Internal Network**: A private ClusterIP network where agents communicate via gRPC/REST.
* **Persistence Layer**: **Cloud Memorystore (Redis)** instance mslarkin-redis for maintaining conversation state and agent-to-agent intermediate artifacts.

### **1.2 Service Decomposition**

While distinct agents, they will share a common Golang foundation (pkg/agent).

1. **Gateway Service (Planning Agent)**:
   * Publicly exposed.
   * Implements POST /v1/chat/completions (OpenAI format).
   * **Streaming Support**: Must implement Server-Sent Events (SSE) to stream partial "thoughts" and keep the connection alive during long multi-agent loops.
   * Orchestrates the lifecycle of a request.
2. **Internal Agent Services**:
   * Code (Primary), Code (Secondary), Design, Ops, Validation, Review.
   * Not exposed publicly.
   * Accessible only via K8s DNS within the agent-ns namespace (e.g., http://agent-code-primary.agent-ns.svc.cluster.local).

## **2\. Go Application Design**

### **2.1 Dependency Injection & Configuration**

The application uses google/adk-go as the wrapper for Vertex AI interactions.

**Configuration Structure (config.yaml):**

models:
  code\_primary:
    model\_id: "qwen/qwen3-coder-480b-a35b-instruct-maas"
    location: "us-south1"
    is\_thinking: false
  code\_secondary:
    model\_id: "moonshot-ai/kimi-k2-thinking-maas"
    location: "global"
    is\_thinking: true
  planning:
    model\_id: "deepseek-ai/deepseek-r1"
    location: "global"
    is\_thinking: true
  ops:
    model\_id: "google/gemini-3.0-flash"
    location: "global"
  design:
    model\_id: "google/gemini-3.0-pro"
    location: "global"
  validation:
    model\_id: "qwen/qwen3-next-80b-a3b-thinking-maas"
    location: "global"
  review:
    model\_id: "google/gemini-3.0-pro"
    location: "global"

storage:
  redis:
    \# Managed Memorystore Instance: mslarkin-redis
    \# Host IP is injected via environment variable or secret
    host: "${REDIS\_HOST}"
    port: 6379

mcp\_clients:
  context7:
    endpoint: "\[https://mcp.context7.com/mcp\](https://mcp.context7.com/mcp)"
    auth\_type: "custom\_header"
    header\_key: "CONTEXT7\_API\_KEY"
  docs\_onemcp:
    endpoint: "\[https://developerknowledge.googleapis.com/mcp\](https://developerknowledge.googleapis.com/mcp)"
    auth\_type: "google\_oauth"
    user\_project\_override: "mslarkin-ext"
  gke\_onemcp:
    endpoint: "\[https://container.googleapis.com/mcp\](https://container.googleapis.com/mcp)"
    auth\_type: "google\_oauth"
    user\_project\_override: "mslarkin-ext"
  gke\_oss:
    endpoint: "http://localhost:8080/mcp" \# UPDATED: Latest release uses /mcp for Streamable HTTP

### **2.2 The Agent Interface**

All agents must implement the standard Agent interface to allow the Planning Agent to treat them polymorphically.

type AgentInput struct {
    Task        string            \`json:"task"\`
    Context     \[\]Message         \`json:"context"\` // Chat history
    Artifacts   map\[string\]string \`json:"artifacts"\` // Previous code/manifests
    Constraints \[\]string          \`json:"constraints"\`
}

type AgentResponse struct {
    Content     string        \`json:"content"\`
    Rationale   string        \`json:"rationale"\` // For thinking models
    TokenUsage  TokenMetrics  \`json:"token\_usage"\`
    Latency     time.Duration \`json:"latency"\`
    Status      StatusEnum    \`json:"status"\` // PENDING\_REVIEW, APPROVED, REJECTED
}

type Agent interface {
    Process(ctx context.Context, input AgentInput) (\*AgentResponse, error)
    // StreamProcess allows real-time token streaming from the agent back to the orchestrator
    StreamProcess(ctx context.Context, input AgentInput, outputChan chan\<- AgentStreamUpdate) error
}

type AgentStreamUpdate struct {
    PartialContent string
    Step           string // e.g., "Drafting Code", "Running Validation"
}

### **2.3 Observability & Distributed Tracing**

To visualize the complex multi-agent workflows, the application must implement **Distributed Tracing** using **Google Cloud Trace**.

* **Instrumentation**: Use the OpenTelemetry Go SDK (go.opentelemetry.io/otel).
* **Exporter**: Configure the texporter (Google Cloud Trace Exporter) to send spans to the GCP project.
* **Context Propagation**:
  * The **Planning Agent** initiates the root span upon receiving the external OpenAI request.
  * Trace Context (Trace ID, Span ID) must be propagated to downstream agents (Code, Design, Validation) via HTTP headers (W3C Trace Context standard).
  * **Vertex AI Integration**: Spans must be created for every call to the Vertex AI API (Gemini/Qwen models) to track token generation latency.
  * **MCP Integration**: Spans must track the execution time of tool calls to sidecar and remote MCP servers.
* **Sampling**: Set to 100% in development/staging; adaptive in production.

## **3\. Workflow Logic & State Machines**

### **3.1 The "Planning" Orchestrator**

The Planning Agent acts as the router. Upon receiving an OpenAI-compatible request:

1. **Auth Check**: Verify API Key or IAP Header.
2. **Parse**: Analyze intent using deepseek-r1.
3. **Route & Stream**:
   * Initiate the specific pipeline (Coding/Design/Ops).
   * **Crucial**: If stream=true, immediately flush HTTP 200 headers.
   * As internal agents change state (e.g., "Primary finished", "Validation started"), emit a specially formatted "Reasoning Content" chunk to the client so the user sees progress.
4. **Aggregate**: Collect metrics and content.
5. **Respond**: Finalize the stream or return JSON body.

### **3.2 The Code Generation Pipeline (Complex Flow)**

This specific logic must be implemented in the pkg/pipelines/code.go controller.

1. **Step 1: Generation (Code Primary)**
   * *Input*: User prompt.
   * *Constraint*: Must generate exactly 3 distinct options.
   * *Output*: Option A, Option B, Option C \+ Rationale.
2. **Step 2: Peer Review & Iteration (Code Secondary)**
   * *Input*: The 3 options from Primary.
   * *Action*: Analyze security, performance, and adherence to Context7 MCP.
   * *Decision Logic*:
     * **Refinement Needed**: If the review finds flaws or missed requirements, return feedback to **Code Primary** (Re-trigger Step 1 with critique). *Limit: 3 Iterations*.
     * **Proceed**: If satisfied, select the preferred option.
   * *Output*: Preferred Option recommendation \+ Rationale.
3. **Step 3: Validation (Validation Agent)**
   * *Input*: Secondary's final recommendation \+ Primary's original options.
   * *Action*: Final correctness check.
   * *Branch*:
     * *If Agreed*: Return Code.
     * *If Conflict*: Forward entire context to **Review Agent**.
4. **Step 4: Arbitration (Review Agent \- Only if Conflict)**
   * *Action*: Uses docs-onemcp and Web Search to break the tie.
   * *Output*: Final Binding Decision.

## **4\. MCP (Model Context Protocol) Integration**

The system uses a hybrid deployment model for MCP servers: **Sidecars** for local tools and **Remote Servers** for shared knowledge bases.

### **4.1 Deployment Strategy**

| MCP Server | Type | Access Method | Auth Mechanism | Headers |
| :---- | :---- | :---- | :---- | :---- |
| **gke-oss** | Sidecar | http://localhost:8080/mcp | None (Localhost trust) | N/A |
| **Context7** | Remote (Public) | https://mcp.context7.com/mcp | **Custom Key** | CONTEXT7\_API\_KEY: \<key\> |
| **docs-onemcp** | Remote (Google) | https://developerknowledge.googleapis.com/mcp | **OAuth2** (ADC) | X-goog-user-project: mslarkin-ext |
| **gke-onemcp** | Remote (Google) | https://container.googleapis.com/mcp | **OAuth2** (ADC) | X-goog-user-project: mslarkin-ext |

### **4.2 Pod Definition Example (Ops Agent)**

The Ops Agent pod includes the gke-oss sidecar, but accesses others remotely.

apiVersion: v1
kind: Pod
metadata:
  name: agent-ops
  namespace: agent-ns
spec:
  serviceAccountName: agent-ksa \# Kubernetes Service Account
  containers:
    \- name: agent-app
      image: gcr.io/mslarkin-ext/agent:latest
      env:
        \# Remote MCP Configuration
        \- name: MCP\_CONTEXT7\_URL
          value: "\[https://mcp.context7.com/mcp\](https://mcp.context7.com/mcp)"
        \- name: MCP\_CONTEXT7\_KEY
          valueFrom:
            secretKeyRef:
              name: mcp-secrets
              key: context7-api-key
        \- name: MCP\_DOCS\_URL
          value: "\[https://developerknowledge.googleapis.com/mcp\](https://developerknowledge.googleapis.com/mcp)"
        \# Local Sidecar Configuration
        \- name: MCP\_GKE\_OSS\_URL
          value: "http://localhost:8080/mcp"
        \- name: GOOGLE\_CLOUD\_PROJECT
          value: "mslarkin-ext"
        \# Persistence Configuration
        \- name: REDIS\_HOST
          value: "10.0.0.5" \# Example IP of mslarkin-redis
    \# Sidecar for GKE OSS
    \- name: mcp-sidecar-gke-oss
      image: gcr.io/mslarkin-ext/gke-oss-mcp:latest
      ports:
        \- containerPort: 8080
      \# Use compiled binary in HTTP mode (required for remote access/sidecar)
      command: \["gke-mcp"\]
      args: \["--server-mode", "http", "--server-port", "8080"\]  

## **5. Annotated Implementation Plan**

### **5.1 Single Binary, Multiple Roles**
To simplify development and deployment, the "Internal Agent Services" and "Gateway Service" should be implemented as a **single Go binary** capable of assuming different roles.
*   **Implementation**: Use configuration parameters:
    *   `AGENT_ROLE` (env var): Determines if the instance starts as `planning`, `code-primary`, `ops`, etc.
    *   `AGENT_MODEL` (env var): Specifies the Vertex AI Model ID to use (e.g., `gemini-3.0-pro`, `deepseek-r1`). This allows overriding the default model for any role.
*   **Benefit**: Shared codebase, simpler CI/CD (one container image), easier local testing, and flexibility to swap models without code changes.

### **5.2 Inter-Agent Communication (A2A)**
The internal communication between agents should use standard HTTP/JSON (REST) to reduce complexity compared to gRPC, unless strict schema enforcement is critical.
*   **Protocol**: HTTP/1.1 or HTTP/2.
*   **Discovery**: Kubernetes DNS (e.g., `http://agent-code-primary.agent-ns.svc.cluster.local:8080`).
*   **Data Format**: JSON using the `AgentInput` and `AgentResponse` structs defined in section 2.2.

### **5.3 MCP Client Implementation**
The MCP client must handle both **Sidecar (Localhost)** and **Remote (Auth-protected)** connections transparently.
*   **Sidecar**: Connect to `http://localhost:<port>/mcp`. No auth required.
*   **Remote (Public)**: Connect to HTTPS URL. Inject `CONTEXT7_API_KEY` header.
*   **Remote (Google)**: Connect to HTTPS URL. Generate OIDC token via `google.golang.org/api/idtoken` or standard `golang.org/x/oauth2/google` ADC and inject as `Authorization: Bearer ...`.

### **5.4 Directory Structure Suggestion**
Refactor `models/adk-agent` to:
*   `cmd/agent/main.go`: Entry point, parses config, initializes specific agent role.
*   `pkg/agent/`: Core `Agent` interface, `AgentInput/Response` types.
*   `pkg/config/`: Configuration loading (Env vars + YAML).
*   `pkg/mcp/`: MCP client implementation (handling auth and transport).
*   `pkg/telemetry/`: OpenTelemetry setup.
*   `internal/planning/`: Planning agent logic (HTTP server, routing).
*   `internal/worker/`: Generic worker logic for Code, Design, Ops agents.
*   `k8s/`: Kubernetes manifests (Deployment, Service, Ingress).

### **5.5 State Management**
*   **Redis**: Use `go-redis/v9` for the client.
*   **Schema**: Store conversation history with a key pattern like `session:{session_id}:history` (List) and artifacts as `session:{session_id}:artifacts:{artifact_id}` (Hash).
