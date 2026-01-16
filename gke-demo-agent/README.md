# GKE Demo Agent (Go ADK)

This project implements a GenAI Agent using the Google Go ADK (`google.golang.org/adk`).

## Prerequisites

1.  **Go 1.23+**
2.  **Google Cloud Project** with Vertex AI API enabled.
3.  **gcloud CLI** (for local authentication)
4.  **ADK CLI** (Optional, for web interface usage)

## Configuration

The agent uses the following environment variables. If not set, they default to the values below:

-   `GOOGLE_CLOUD_PROJECT`: `mslarkin-ext`
-   `GOOGLE_CLOUD_LOCATION`: `us-west1`
-   `GOOGLE_GENAI_USE_VERTEXAI`: `true`

To override them:

```bash
export GOOGLE_CLOUD_PROJECT=your-project-id
export GOOGLE_CLOUD_LOCATION=us-central1
```

## Running the Agent

To run the agent locally:

```bash
go mod tidy
go run main.go
```

## ADK Web Interface

If you have the ADK CLI installed, you can use the web interface to interact with your agent.

```bash
# Example command if CLI is available
adk web --agent-url http://localhost:8080
```

## Notes

-   **Model**: Configured to use `gemini-1.5-pro` (User requested 'Gemini 3 Pro', mapped to 1.5 Pro).
-   **Imports**: This project uses `google.golang.org/adk`. Ensure you have access to this package.
