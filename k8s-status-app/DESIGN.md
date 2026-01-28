# GKE Status Web App: Design & Implementation Guide (MCP Edition)

## 1. System Architecture Overview

The system follows a client-server architecture deployed on GKE.

* **Backend (MCP Client):** A Golang service acting as a Model Context Protocol (MCP) Client. It connects to external MCP Servers to retrieve cluster and workload context.
* **Frontend:** A Golang application using **Go, HTMX, and Gomponents** to render the data aggregated by the backend.

---

## 2. Backend Design Specifications

The Backend functions as the central data aggregator. It must be implemented as an **MCP Client** that connects to specific **MCP Servers** rather than wrapping CLI tools like `kubectl`.

### 2.1. Protocol & Connections

* **Protocol:** Model Context Protocol (MCP).
* **Language:** Golang.
* **MCP Server Connections:** The backend must initialize connections to the following MCP Servers to access configuration, status, and metrics:
    1. **GKE OneMCP Server:** `https://container.googleapis.com/mcp`.
    2. **GKE OSS MCP Server:** `https://mcp.ai.mslarkin.com`.

### 2.2. Data Retrieval via MCP

The backend maps frontend requests to specific MCP primitives (Resources or Tools) provided by the upstream servers.

#### **A. Cluster Discovery (via GKE OneMCP)**

* **Source:** GKE OneMCP Server.
* **Mechanism:** Query the server for the list of projects and available clusters.
* **MCP Interaction:** Use the protocol to list available contexts/resources associated with the user's Project ID input.
* **Data Extracted:** Project names, Cluster names, and statuses for the frontend landing page.

#### **B. Workload Aggregation (via GKE OSS MCP)**

* **Source:** GKE OSS MCP Server.
* **Mechanism:** Retrieve workload lists for specific namespaces (e.g., `agent-ns`) and types (deployments, services).
* **MCP Interaction:** Access resources corresponding to "List" operations for Kubernetes types.
* **Data Transformation:**
    * Aggregate the data into lists separated by namespace and type.
    * Extract specific columns to match standard CLI output (e.g., `NAME`, `READY`, `UP-TO-DATE`, `AVAILABLE`, `AGE` for deployments).

#### **C. Workload Inspection (via GKE OSS MCP)**

The backend must expose capabilities to fetch detailed views for the "Describe" and "Pods" buttons.

* **Feature: Describe Resource**
    * **Trigger:** Frontend requests description for a specific resource (e.g., `deployment go-flexible-workload`).
    * **MCP Interaction:** Call the specific MCP Tool or read the Resource URI that provides the equivalent of `kubectl describe`.

* **Feature: List Pods**
    * **Trigger:** Frontend requests pods for a specific workload.
    * **MCP Interaction:** Query the MCP Server for pod resources linked to the workload, ensuring status is included.

### 2.3. AI & Agent Framework Specifications

The backend implementation must adhere to specific framework and model requirements for modern agentic capabilities.

* **Framework:** **Google Gen AI Agent Development Kit (ADK)**.
    * **Requirement:** The backend agent logic and tool orchestration must be built using the Google ADK to leverage standardized patterns for reliability and observability.
* **Model:** **Vertex AI Gemini 3 Pro**.
    * **Requirement:** The **Chat Assistant** feature must use the Vertex AI Gemini 3 Pro model to answer user questions about the environment. Core dashboard rendering remains deterministic.

### 2.4. Identity & Permissions

* **Service Account:** The Backend service must run as a dedicated Service Account (e.g., `k8s-status-backend-sa`).
* **Permissions:**
    * **GKE MCP Access:** Must have permissions to authenticate and communicate with the GKE OneMCP (`https://container.googleapis.com/mcp`) and GKE OSS MCP (`https://mcp.ai.mslarkin.com`) servers.
    * **Kubernetes Reader:** Must have `view` or equivalent read-only RoleBindings in the target clusters to fetch workload status, metrics, and logs.
    * **Monitoring Viewer:** Must have `roles/monitoring.viewer` (or equivalent) if fetching metrics from Google Cloud Monitoring.

### 2.5. API Interface

Since the Frontend and Backend are decoupled, the Backend must expose a structured API.

* **Protocol:** RESTful JSON API using standard HTTP verbs (GET, POST).
* **Endpoints:**
    * `GET /api/projects`: List available projects.
    * `GET /api/clusters`: List clusters (optionally filtered by project).
    * `GET /api/workloads`: List workloads for a specific cluster/namespace.
    * `GET /api/workload/{name}`: Get detailed description for a workload.
    * `GET /api/workload/{name}/pods`: Get pods for a workload.
    * `POST /api/chat`: Send user questions to the AI assistant.

---

## 3. Frontend Design Specifications



The Frontend is a standard **Golang** web application using **Server-Side Rendering (SSR)**.

* **Stack:**
    * **Language:** Golang.
    * **Templating:** `github.com/maragudk/gomponents` (Type-safe HTML components).
    * **Interactivity:** `htmx` (for dynamic updates without full page reloads).
* **Architecture:** The Frontend acts as a consumer of the Backend's JSON API. It fetches data server-side and renders HTML to the client.

### 3.1. User Interface Structure

* **Landing Page:** Input field for Project ID(s).
* **Dashboard:**
    * **Tabs:** One tab per Project.
    * **Cards:** One card per Cluster within the project tab.
    * **Chat Widget:** A floating or docked chat interface for querying the AI assistant.

### 3.2. Cluster Card Layout

Each Cluster Card visualizes the data retrieved from the MCP Servers:

* **Organization:** Group workloads by **Namespace** and **Type** (e.g., a list for Deployments, a list for Services).
* **Rows:** Each workload is a row displaying:
    * Name.
    * Status.
    * Resource-specific columns (mirroring standard CLI output).
* **Actions:**
    * **"Describe" Button:** Opens a view with the detailed resource description.
    * **"Pods" Button:** Opens a list of pods associated with that workload.

### 3.3. Identity & Permissions

* **Service Account:** The Frontend application must run as a dedicated Service Account (e.g., `k8s-status-frontend-sa`).
* **Permissions:**
    * **Backend Access:** Must have permissions to reach the Backend service (e.g., via Cloud Run invoker roles or internal GKE network policies).
    * **Least Privilege:** Should NOT have direct access to the MCP Servers or Kubernetes API; it must rely entirely on the Backend.

---

## 4. Implementation Notes for Agents

* **No CLI Wrappers:** Do not shell out to `kubectl` or `gcloud`. All data **must** be sourced via the Model Context Protocol connections defined in Section 2.1.
* **MCP SDK:** Use a standard Golang SDK for Model Context Protocol to handle the client-server handshake and message framing.
* **Deployment:** The final artifacts must be containerizable for deployment to GKE.

---

## 5. Deployment Specifications

### 5.1. Target Environment

* **Project:** `mslarkin-ext`
* **Cluster:** `ai-auto-cluster`
* **Scope:** Both Frontend and Backend services must be deployed to this specific cluster.

### 5.2. Network Exposure

* **Frontend:**
    * **Mechanism:** Must be exposed via **Global Load Balancing** using a Standalone Network Endpoint Group (NEG).
    * **Annotation:** The Frontend Service manifest must include the following annotation:
        ```yaml
        cloud.google.com/neg: '{"exposed_ports": {"80":{"name": "frontend-neg"}}}'
        ```
    * ** Backend:** Should remain internal, accessible only by the Frontend (and potentially other authorized internal services).
