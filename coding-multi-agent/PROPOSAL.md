
# Goal:
Create a Golang application that hosts several AI Agents, each backed by Vertex AI.  The Planning Agent will be accessible via OpenAI-compatible API.  The other agents will communicated via A2A, but are only available to the Planning Agent (and each other).

# Verification
- Planning Agent accesible via https://model.ai.mslarkin.com (OpenAI-compatible API)
- Other agents accessible to the Planning Agent and each other via A2A

## Technical Stack:
- Golang
- Google ADK (https://github.com/google/adk-go)
- GKE
- Load Balancer: mslarkin-ext-lb
- Domain: https://model.ai.mslarkin.com

## Project:
mslarkin-ext

## Regions:
- us-west1 (preferred)
- global (preferred)
- us-central1 (fallback)

## Models:
- "qwen/qwen3-coder-480b-a35b-instruct-maas", DisplayName: "Qwen 3 Coder 480B (Instruct)", IsThinking: false, Region: "us-south1"
- "qwen/qwen3-next-80b-a3b-thinking-maas", DisplayName: "Qwen 3 Next 80B (Thinking)", IsThinking: true, Region: "global"
- "google/gemini-3.0-flash", DisplayName: "Gemini 3.0 Flash", IsThinking: true, Region: "global"
- "google/gemini-3.0-pro", DisplayName: "Gemini 3.0 Pro", IsThinking: true, Region: "global"
- "deepseek-ai/deepseek-r1", DisplayName: "DeepSeek R1", IsThinking: true, Region: "global"
- "moonshot-ai/kimi-k2-thinking-maas", DisplayName: "Kimi K2 Thinking", IsThinking: true, Region: "global"

## Agents
### General instructions
- All agents must send their responses to the Validation Agent for review and approval, before returning reponse to the Planning Agent.  If there is a conflict, disagreement, or uncertainty, all options being considered, including rationale, will be sent to the Review Agent, which will make the final decision.
- All agents must include the token usage metrics and latency/time spent working in the response to the Planning Agent.

### Planning Agent
- Model: deepseek-ai/deepseek-r1
- Purpose: Understand the user's request, develop a plan to implement, and route requests to the appropriate agents, depending on the task.
- Mandatory instructions
  - Receive responses from Code, Design, and Validation Agents, including token usage metrics and latency/time spent working.
  - Send necessary artifacts to the Ops Agent for actuation (ie. deployment)

### Code Agents
- Purpose: Generate code based on the user's request.
- MCP Servers: Context7, docs-onemcp (Google Cloud documentation)
- Mandatory instructions
  - Reference the Context7 MCP to verify proper formatting and usage.
  - Reference the docs-onemcp MCP to verify proper formatting and usage related to Google Cloud products and services.

#### Code Agent (Primary)
- Model: qwen/qwen3-coder-480b-a35b-instruct-maas
- Mandatory instructions
  - Generate 3 options for the code, including rationale.
  - Send the options to the secondary code agent for review.
  - If there is a conflict, disagreement, or uncertainty, all options being considered, including rationale, will be sent to the Validation Agent, which will make the final decision.

#### Code Agent (Secondary)
- Model: moonshot-ai/kimi-k2-thinking-maas
- Mandatory instructions
  - Review the options being considered, including rationale.
  - Send the options to the Validation Agent for review and approval, before returning reponse to the Planning Agent.  If there is a conflict, disagreement, or uncertainty, all options being considered, including rationale, will be sent to the Review Agent, which will make the final decision.

### Ops Agent
- Model: google/gemini-3.0-flash
- Purpose: Execute operations based on the user's request, including deploying generated code and configurations.
- MCP Servers: gke-onemcp (GKE tools), gke-oss-mcp (install locally: https://github.com/GoogleCloudPlatform/gke-mcp)
- Mandatory instructions
  - Prefer using gke-onemcp, with gke-oss-mcp as a fallback.

## Design Agent
- Model: google/gemini-3.0-pro
- Purpose: Generate configurations necessary to implement the user's request. For example, generating the necessary kubernetes manifests.
- MCP Servers: gke-onemcp (GKE tools), gke-oss-mcp (install locally: https://github.com/GoogleCloudPlatform/gke-mcp)
- Mandatory instructions
  - Prefer using gke-onemcp, with gke-oss-mcp as a fallback.

### Validation Agent
- Model: qwen/qwen3-next-80b-a3b-thinking-maas
- Purpose: Validate the generated code/configurations. Provide feedback to the generating agent for iteration/improvement.
- Mandatory instructions
  - If the Validation Agent and the generating agent cannot find agreement, the Validation Agent will send the options being considered, including rationale, to the Review Agent, which will make the final decision.

### Review Agent
- Model: google/gemini-3.0-pro
- Purpose: Review the generated code/configurations. Provide final approval or rejection.  Also acts as final reviewer once the task is complete, validating it is working as expected.
- MCP Servers: Context7, docs-onemcp (Google Cloud documentation)
- Mandatory instructions
  - Reference the Context7 MCP to verify proper formatting and usage.
  - Reference the docs-onemcp MCP to verify proper formatting and usage related to Google Cloud products and services.
  - Always web search for official documentation and references to verify proper formatting and usage (if not available via MCP)
  - Use whatever tools are necessary to verify that the user's request was completed successfully.  This includes curl, gcloud, kubectl, browser, etc.
