# Backend Design

## Overview
The backend is a stateless REST API service written in Go. Its primary responsibility is to act as a gateway and aggregator for the "Model Context Protocol" (MCP). It connects to upstream MCP servers to retrieve Kubernetes cluster and workload data and serves this data to the frontend in a simplified JSON format. It also hosts a conversational agent service.

## Architecture

### Layers
1.  **API Handler Layer (`api/handlers.go`)**:
    -   Standard `net/http` handlers.
    -   Parses query parameters and request bodies.
    -   Delegates business logic to the `Server` struct dependencies.
    -   Marshals responses to JSON.

2.  **Service Layer**:
    -   **MCPClient (`mcpclient/client.go`)**: Manages connections to upstream MCP servers (e.g., "OneMCP" for clusters, "OSSMCP" for workloads). Handles authentication and tool execution.
    -   **ChatService (`chat/agent.go`)**: Manages the conversational AI agent lifecycle and streams responses.

3.  **Data Models (`models/types.go`)**:
    -   Simple struct definitions (DTOs) for `Cluster`, `Workload`, and `Pod`.
    -   Shared contract with the frontend.

## API Specification

### 1. List Projects
*   **Endpoint**: `GET /api/projects`
*   **Description**: Returns a list of available GCP projects.
*   **Response**:
    ```json
    ["mslarkin-ext", "mslarkin-demo"]
    ```

### 2. List Clusters
*   **Endpoint**: `GET /api/clusters`
*   **Query Params**: `project` (required)
*   **Description**: Fetches clusters from the MCP provider.
*   **Response**:
    ```json
    [
      {
        "name": "autopilot-cluster-1",
        "projectId": "mslarkin-demo",
        "location": "us-central1",
        "status": "RUNNING"
      }
    ]
    ```

### 3. List Workloads
*   **Endpoint**: `GET /api/workloads`
*   **Query Params**: `cluster`, `namespace` (required)
*   **Description**: Fetches workloads (Deployments, StatefulSets, etc.) for the specific cluster.
*   **Response**:
    ```json
    [
      {
        "name": "frontend",
        "namespace": "default",
        "type": "Deployment",
        "status": "Ready",
        "ready": "3/3",
        "upToDate": "3",
        "available": "3",
        "age": "2d"
      }
    ]
    ```

### 4. Get Workload Details
*   **Endpoint**: `GET /api/workloads/{name}`
*   **Query Params**: `cluster`, `namespace` (required)
*   **Response**: A single `Workload` object.

### 5. Chat
*   **Endpoint**: `POST /api/chat`
*   **Content-Type**: `application/json`
*   **Request**:
    ```json
    {
      "message": "Why is the frontend crashing?",
      "session": "user-session-id"
    }
    ```
*   **Response**: Server-Sent Events (SSE) stream.
    ```text
    data: {"type": "text", "content": "Checking the logs..."}
    
    data: {"type": "tool_use", "tool": "list_logs", "input": {...}}
    ```

## External Communication (MCP)

The backend does **not** use the Kubernetes client-go library directly. Instead, it uses the **Model Context Protocol**.

### OneMCP (Cluster Metadata)
-   **Purpose**: Listing GKE clusters across projects.
-   **Tool Used**: `list_clusters`
-   **Authentication**: OIDC / Google Default Application Credentials.

### OSSMCP (Kubernetes Data)
-   **Purpose**: Fetching in-cluster resources (Pods, Deployments).
-   **Tool Used**: `list_resources` (generic) or specific tools like `kubectl_get`.
-   **Authentication**: Bearer token or local kubeconfig context (depending on deployment).

## Error Handling
-   Standard HTTP status codes are used (200, 400, 500).
-   Internal errors are logged to stdout/stderr.
-   The backend attempts to normalize MCP errors into user-friendly HTTP responses.
