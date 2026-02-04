# Frontend Design

## Overview
The frontend is a Server-Side Rendering (SSR) application built with Go, `gomponents`, and HTMX. It serves as the user interface for the K8s Status App, providing a dashboard to view cluster resources and an interactive chat interface powered by an AI agent.

## Architecture

### Server
- **Entry Point**: `main.go`
- **Routing**: `http.ServeMux` handles incoming HTTP requests.
- **Port**: Defaults to `8081`.
- **Backend Communication**: Proxies API requests to the backend service (defaulting to `http://localhost:8080`).

### View Layer
The view layer uses [gomponents](https://github.com/maragudk/gomponents) to write HTML components in pure Go code. This ensures type safety and consistency.

#### Key Components
1.  **Layout (`components/layout.go`)**:
    -   Wraps every page.
    -   Includes global CSS (DaisyUI/Tailwind) and JS (HTMX, A2UI).
    -   Provides the navigation bar and footer.

2.  **Dashboard (`views/dashboard.go`)**:
    -   The main application view.
    -   Renders a grid of `ClusterCard` components.
    -   **HTMX Integration**: The `ClusterCard` initially loads with a spinner or skeleton state. It uses `hx-get="/partials/workloads?..."` and `hx-trigger="load"` to asynchronously fetch and inject the workload data.

3.  **Workloads Partial (`views/partials.go` / `handlePartialsWorkloads`)**:
    -   A specialized view that renders *only* the `<tbody>` or list of rows for the workloads table.
    -   Designed to be swapped into the DOM by HTMX.

4.  **A2UI Shell (`views/a2ui_shell.go`)**:
    -   A specialized view for the AI Chat experience.
    -   Renders a container for the `<a2ui-root>` Web Component.
    -   Includes client-side JavaScript to handle the chat streaming protocol and bridge it to the UI components.

## Lifecycle & Data Flow

### 1. Initial Page Load (Dashboard)
1.  User requests `GET /dashboard?project=my-project`.
2.  Server renders the `Layout` and `Dashboard` view.
3.  The `Dashboard` view iterates over the project's clusters (passed via context or fetched).
4.  HTML is returned to the browser.

### 2. Async Data Fetching (Workloads)
1.  Browser parses HTML. HTMX finds elements with `hx-trigger="load"`.
2.  Browser sends `GET /partials/workloads?cluster=c1&namespace=default` to the **Frontend Server**.
3.  **Frontend Server** acts as a proxy/aggregator:
    -   Makes an HTTP GET request to `Backend Service`: `GET /api/workloads?cluster=c1&namespace=default`.
    -   Receives JSON response: `[{"name": "nginx", "status": "Ready", ...}]`.
4.  **Frontend Server** passes this data to the `WorkloadsList` gomponent.
5.  **Frontend Server** renders the component to HTML and returns it.
6.  Browser (HTMX) swaps the HTML into the target container.

### 3. Chat Interaction (A2UI)
1.  User types a message in the chat interface.
2.  Client-side JS sends `POST /chat/proxy`.
3.  **Frontend Server** proxies the request body to `Backend Service` (`POST /api/chat`).
4.  **Backend** returns a Server-Sent Events (SSE) stream.
5.  **Frontend Server** streams the chunks back to the client.
6.  Client-side JS parses the SSE events and updates the `<a2ui-root>` component.

## URL Structure

| Path | Method | Description |
| :--- | :--- | :--- |
| `/` | GET | Landing page |
| `/dashboard` | GET | Main dashboard view. Query param: `project` |
| `/partials/workloads` | GET | HTML fragment for workload list. Query params: `cluster`, `namespace` |
| `/chat/proxy` | POST | Proxy for chat requests to backend |
| `/static/*` | GET | Static assets (CSS, JS) |

## Dependencies
-   **Go 1.22+**
-   **gomponents**: HTML generation.
-   **HTMX**: Client-side interactivity.
-   **DaisyUI / Tailwind**: Styling (via CDN or static file).
