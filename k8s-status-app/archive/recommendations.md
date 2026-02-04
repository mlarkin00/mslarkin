# Codebase Recommendations

This document outlines suggested improvements for the `k8s-status-app` codebase, focusing on functionality, reliability, and maintainability.

## 1. Critical Fixes (Broken Functionality)

### Register Missing Frontend Handlers
**Location:** `frontend/main.go`
**Issue:** The functions `handlePartialsWorkloads` and `handleChatProxy` are defined but never registered to the HTTP router (`apiMux`).
**Impact:** The dashboard will fail to load workload data, and the AI Assistant chat widget will not function (404 errors).
**Recommendation:** Register these handlers in the `main` function.

```go
apiMux.HandleFunc("GET /partials/workloads", handlePartialsWorkloads)
apiMux.HandleFunc("POST /chat/proxy", handleChatProxy)
```

## 2. Reliability & Error Handling

### Stop Swallowing Initialization Errors
**Location:** `backend/mcpclient/client.go` (`NewMCPClient`)
**Issue:** Connection errors to MCP servers are logged as warnings, but the function returns `nil` error.
**Impact:** The backend starts successfully even if it cannot connect to upstream data sources. Calls to `ListClusters` will later fail with "OneMCP session is not available", making debugging difficult.
**Recommendation:** Return the error immediately to allow the application to implement a rigorous health check/reconnect mechanism.

```go
// Current
if err != nil {
    log.Printf("Warning: Failed to connect to OneMCP: %v", err)
}

// Recommended
if err != nil {
    return nil, fmt.Errorf("failed to connect to OneMCP: %w", err)
}
```

### Secure API Error Responses
**Location:** `backend/api/handlers.go`
**Issue:** Handlers return raw error strings to the client (e.g., `fmt.Sprintf("failed to list clusters: %v", err)`).
**Impact:** This leaks internal implementation details and stack traces, which is a security risk.
**Recommendation:** Log the detailed error on the server and return a generic error message to the client.

```go
// Recommended
log.Printf("Error listing clusters: %v", err)
http.Error(w, "Internal Server Error", http.StatusInternalServerError)
```

## 3. Maintainability & Code Quality

### Extract Inline JavaScript
**Location:** `frontend/components/layout.go`
**Issue:** The `Layout` function contains a large, complex JavaScript block embedded in a Go string literal.
**Impact:** No syntax highlighting, no linting, difficult to read, and prone to syntax errors.
**Recommendation:** Move this logic to `frontend/public/js/chat-widget.js` and load it via a `<script src="...">` tag.

### Simplify Frontend Routing
**Location:** `frontend/main.go`
**Issue:** The manual path cleaning (`path.Clean`) and slash-trapping logic is complex and brittle. It attempts to work around `http.StripPrefix` behavior and Load Balancer path rewriting.
**Recommendation:**
1. Rely on standard `http.StripPrefix`.
2. Configure the Load Balancer or Reverse Proxy to handle path stripping if possible.
3. If application-level stripping is required, use a robust middleware pattern rather than ad-hoc path manipulation in `main`.

## 4. Technical Debt

### Remove Hardcoded Data
**Location:** `backend/api/handlers.go` (`ListProjects`)
**Issue:** The project list is hardcoded (`[]string{"mslarkin-ext", "mslarkin-demo"}`).
**Recommendation:** Fetch this list dynamically from a configuration file, environment variable, or an MCP endpoint (e.g., Resource Manager).

### Improve Mocked Data
**Location:** `backend/mcpclient/client.go` (`ListWorkloads`)
**Issue:** Workload details (Type, Status, Ready, Age) are hardcoded mocks.
**Recommendation:** Parse the actual resource data returned by the MCP server to populate these fields.
