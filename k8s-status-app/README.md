# K8s Status App

This application provides a status dashboard for GKE clusters, leveraging the Model Context Protocol (MCP) to fetch data from GKE and OSS sources. It also includes an AI-powered chat assistant.

## Architecture

The system consists of two main components:

1.  **Backend (`k8s-status-backend`)**:
    *   **Role**: Acts as the central data aggregator and MCP Client.
    *   **Functionality**:
        *   Connects to GKE OneMCP (`container.googleapis.com/mcp`) and GKE OSS MCP (`mcp.ai.mslarkin.com`).
        *   Fetches cluster and workload data via MCP.
        *   Hosts an AI agent using Google ADK and Vertex AI for chat capabilities.
        *   Exposes a REST API for the frontend.
    *   **Tech Stack**: Go, `github.com/modelcontextprotocol/go-sdk`, `google.golang.org/adk`.

2.  **Frontend (`k8s-status-frontend`)**:
    *   **Role**: User Interface.
    *   **Functionality**:
        *   Renders the dashboard using Server-Side Rendering (SSR).
        *   Uses HTMX for dynamic content loading (e.g., workload lists).
        *   Provides a chat widget for interacting with the backend agent.
    *   **Tech Stack**: Go, `maragu.dev/gomponents`, HTMX, TailwindCSS, DaisyUI.

## Directory Structure

*   `backend/`: Source code for the backend service.
    *   `auth/`: OIDC authentication logic.
    *   `mcpclient/`: MCP client implementation and caching.
    *   `chat/`: ADK agent setup.
    *   `api/`: HTTP handlers.
*   `frontend/`: Source code for the frontend application.
    *   `components/`: Reusable UI components (Layout, Navbar, etc.).
    *   `views/`: Page definitions (Landing, Dashboard).
    *   `models/`: Shared data structures.
*   `deploy/`: Kubernetes manifests for deployment.

## Prerequisites

*   Go 1.24+
*   Google Cloud Project with Vertex AI and GKE API enabled.
*   `kubectl` configured for the target cluster.
*   `gcloud` CLI.

## Setup & Deployment

1.  **Configure Environment**:
    Edit `setup.sh` to match your Project ID and Cluster details.

2.  **Run Setup Script**:
    This script creates the necessary Service Accounts, Workload Identity bindings, and applies the Kubernetes manifests.
    ```bash
    ./setup.sh
    ```

3.  **Build Images**:
    You need to build and push the Docker images to your container registry (e.g., GCR or Artifact Registry).
    *   Backend: `gcr.io/YOUR_PROJECT/k8s-status-backend:latest`
    *   Frontend: `gcr.io/YOUR_PROJECT/k8s-status-frontend:latest`

## Local Development

To run locally, you need to set environment variables and ensure you have valid credentials (ADC).

**Backend:**
```bash
cd backend
export GOOGLE_CLOUD_PROJECT=mslarkin-ext
go run main.go
```

**Frontend:**
```bash
cd frontend
export BACKEND_URL=http://localhost:8080
go run main.go
```

Access the frontend at `http://localhost:8081`.
