// Package mcpclient handles the connections to Model Context Protocol (MCP) servers.
package mcpclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s-status-backend/models"
	gcputils "github.com/mlarkin00/mslarkin/go-mslarkin-utils/gcputils"
	"gopkg.in/yaml.v3"
	"encoding/json"
)

// MCPSession defines the interface for an MCP client session.
type MCPSession interface {
	ListResources(ctx context.Context, params *mcp.ListResourcesParams) (*mcp.ListResourcesResult, error)
	CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
	ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	ListPrompts(ctx context.Context, params *mcp.ListPromptsParams) (*mcp.ListPromptsResult, error)
	InitializeResult() *mcp.InitializeResult
	Close() error
}

// MCPClient wraps connections to multiple MCP servers and provides
// high-level methods to interact with them, including caching.
// It manages sessions for both OneMCP (Google's MCP) and OSSMCP (Open Source MCP).
type MCPClient struct {
	// OneMCP is the client for the GKE OneMCP server.
	OneMCP        *mcp.Client
	OneMCPSession MCPSession

	// OSSMCP is the client for the GKE OSS MCP server.
	OSSMCP        *mcp.Client
	OSSMCPSession MCPSession

	cache map[string]cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

const (
	OneMCPEndpoint = "https://container.googleapis.com/mcp"
	OSSMCPEndpoint = "https://mcp.ai.mslarkin.com/sse"
	CacheTTL       = 30 * time.Second
)

// IDTokenTransport injects an OIDC ID token.
type IDTokenTransport struct {
	Base     http.RoundTripper
	Audience string
}

func (t *IDTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := gcputils.GetIDToken(req.Context(), t.Audience)
	if err != nil {
		return nil, fmt.Errorf("failed to get ID token: %w", err)
	}

	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+token)

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}

// AccessTokenTransport injects an OAuth2 access token.
type AccessTokenTransport struct {
	Base http.RoundTripper
}

func (t *AccessTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := gcputils.GetAccessToken(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+token)

	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}

// NewMCPClient initializes connections to both OneMCP and OSSMCP servers.
// It acts as a factory, setting up OIDC authentication and SSE transport
// for secure communication with the MCP servers.
func NewMCPClient(ctx context.Context) (*MCPClient, error) {
	client := &MCPClient{
		cache: make(map[string]cacheEntry),
	}

	// Initialize OneMCP with Access Token (OAuth2)
	// User suggested using streaminghttp transport.
	oneTrans := &mcp.StreamableClientTransport{
		Endpoint: OneMCPEndpoint,
		HTTPClient: &http.Client{
			Transport: &AccessTokenTransport{},
			Timeout:   120 * time.Second,
		},
	}
	client.OneMCP = mcp.NewClient(&mcp.Implementation{Name: "k8s-status-backend-onemcp", Version: "1.0.0"}, nil)
	oneSession, err := client.OneMCP.Connect(ctx, oneTrans, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OneMCP: %w", err)
	}
	log.Printf("[DEBUG] OneMCP: Connected successfully to %s", OneMCPEndpoint)
	client.OneMCPSession = NewLoggingSession(oneSession, "OneMCP")
	// Log Capabilities
	if initRes := client.OneMCPSession.InitializeResult(); initRes != nil {
		log.Printf("[INFO] OneMCP Capabilities: Tools=%v Prompts=%v Resources=%v",
			initRes.Capabilities.Tools != nil,
			initRes.Capabilities.Prompts != nil,
			initRes.Capabilities.Resources != nil,
		)
	}

	// Initialize OSSMCP with ID Token (OIDC)
	// Use the IAP Client ID as the audience for authentication.
	const ossMcpIAPClientID = "79309377625-i17s6rtmlmi6t3dg61b69nvfsvss8cdp.apps.googleusercontent.com"
	ossTrans := &mcp.StreamableClientTransport{
		Endpoint: OSSMCPEndpoint,
		HTTPClient: &http.Client{
			Transport: &IDTokenTransport{Audience: ossMcpIAPClientID},
			Timeout:   120 * time.Second,
		},
	}
	client.OSSMCP = mcp.NewClient(&mcp.Implementation{Name: "k8s-status-backend-ossmcp", Version: "1.0.0"}, nil)
	ossSession, err := client.OSSMCP.Connect(ctx, ossTrans, nil)
	if err != nil {
		fmt.Printf("Warning: failed to connect to OSSMCP: %v\n", err)
		// return nil, fmt.Errorf("failed to connect to OSSMCP: %w", err)
	} else {
		log.Printf("[DEBUG] OSSMCP: Connected successfully to %s", OSSMCPEndpoint)
		client.OSSMCPSession = NewLoggingSession(ossSession, "OSSMCP")
		// Log Capabilities
		if initRes := client.OSSMCPSession.InitializeResult(); initRes != nil {
			log.Printf("[INFO] OSSMCP Capabilities: Tools=%v Prompts=%v Resources=%v",
				initRes.Capabilities.Tools != nil,
				initRes.Capabilities.Prompts != nil,
				initRes.Capabilities.Resources != nil,
			)
		}
	}

	return client, nil
}

// Close closes the underlying MCP sessions.
func (c *MCPClient) Close() {
	if c.OneMCPSession != nil {
		c.OneMCPSession.Close()
	}
	if c.OSSMCPSession != nil {
		c.OSSMCPSession.Close()
	}
}

// Helper for caching
func (c *MCPClient) getCached(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.cache == nil {
		return nil, false
	}
	entry, ok := c.cache[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Data, true
}

func (c *MCPClient) setCached(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = make(map[string]cacheEntry)
	}
	c.cache[key] = cacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(CacheTTL),
	}
}

// ListClusters retrieves the list of clusters from OneMCP.
// It uses caching to reduce latency by storing results for a short period.
// The projectID parameter specifies which Google Cloud project to query.
func (c *MCPClient) ListClusters(ctx context.Context, projectID string) ([]models.Cluster, error) {
	if c.OneMCPSession == nil {
		return nil, fmt.Errorf("OneMCP session is not available")
	}
	key := "clusters:" + projectID
	if data, ok := c.getCached(key); ok {
		return data.([]models.Cluster), nil
	}

	// Fetch from OneMCP using list_clusters tool
	result, err := c.OneMCPSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_clusters",
		Arguments: map[string]interface{}{
			"parent": fmt.Sprintf("projects/%s/locations/-", projectID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call list_clusters: %w", err)
	}

	var clusters []models.Cluster

	// Parse StructuredContent
	// content map expected to have "clusters" key
    var contentMap map[string]interface{}
    if result.StructuredContent != nil {
         if m, ok := result.StructuredContent.(map[string]interface{}); ok {
             contentMap = m
         } else if m, ok := result.StructuredContent.(map[string]any); ok {
             contentMap = m
         }
    }

    if contentMap != nil {
        // DEBUG: Log the raw content map keys and clusters list size
        keys := make([]string, 0, len(contentMap))
        for k := range contentMap {
            keys = append(keys, k)
        }
        log.Printf("DEBUG: ListClusters structure keys: %v", keys)
        if clustersAny, ok := contentMap["clusters"]; ok {
            if clustersList, ok := clustersAny.([]interface{}); ok {
                log.Printf("DEBUG: Found %d clusters in response", len(clustersList))
                for i, c := range clustersList {
                     log.Printf("DEBUG: Cluster %d: %+v", i, c)
                }
                for _, cAny := range clustersList {
                    if cMap, ok := cAny.(map[string]interface{}); ok {
                        name, _ := cMap["name"].(string)
                        location, _ := cMap["location"].(string)
                        status, _ := cMap["status"].(string)
                        // name might be short name, we can use it directly.

                        clusters = append(clusters, models.Cluster{
                            Name:      name,
                            ProjectID: projectID,
                            Location:  location,
                            Status:    status,
                        })
                    }
                }
            }
        }
    }

	c.setCached(key, clusters)
	return clusters, nil
}

// ListWorkloads retrieves the list of workloads using OneMCP (kube_get) with fallback to OSSMCP.
func (c *MCPClient) ListWorkloads(ctx context.Context, project, location, cluster, namespace string) ([]models.Workload, error) {
	// 1. Try OneMCP if we have the context
	if c.OneMCPSession != nil && project != "" && location != "" && cluster != "" {
		key := fmt.Sprintf("workloads:onemcp:%s:%s:%s:%s", project, location, cluster, namespace)
		if data, ok := c.getCached(key); ok {
			return data.([]models.Workload), nil
		}

		parent := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)

		// Fetch Deployments
		deps, err := c.fetchK8sResources(ctx, parent, namespace, "deployments")
		if err == nil {
			// Fetch Services
			svcs, err2 := c.fetchK8sResources(ctx, parent, namespace, "services")
			if err2 == nil {
				// Combine
				workloads := make([]models.Workload, 0, len(deps)+len(svcs))
				workloads = append(workloads, deps...)
				workloads = append(workloads, svcs...)
				c.setCached(key, workloads)
				return workloads, nil
			} else {
                 log.Printf("Warning: Failed to fetch services from OneMCP: %v", err2)
                 // If we got deployments, maybe return them? But let's fallback if partial success?
                 // Usually if deployments work, services should too unless permission issue.
                 // We'll proceed with fallback if we prefer completeness, or just use deployments.
                 // Let's fallback if any fail to ensure consistency.
            }
		} else {
             log.Printf("Warning: Failed to fetch deployments from OneMCP: %v", err)
        }
	}

	// 2. Fallback to OSSMCP
	if c.OSSMCP == nil { // Was checking session but client check is fine too, or just session.
        if c.OSSMCPSession == nil {
		    return nil, fmt.Errorf("OSSMCP session is not available (and OneMCP failed or context missing)")
        }
	} else if c.OSSMCPSession == nil {
         return nil, fmt.Errorf("OSSMCP session is not available")
    }

	key := fmt.Sprintf("workloads:ossmcp:%s:%s", cluster, namespace)
	if data, ok := c.getCached(key); ok {
		return data.([]models.Workload), nil
	}

	// Fetch from OSSMCP
	result, err := c.OSSMCPSession.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		return nil, err
	}

	var workloads []models.Workload
	for _, r := range result.Resources {
		workloads = append(workloads, models.Workload{
			Name:      r.Name,
			Namespace: namespace,
			Type:      "Deployment", // Defaulting to Deployment for demo
			Status:    "Ready",      // Mocked status
            Ready:     "1/1",        // Mocked readiness
            Age:       "1d",         // Mocked age
		})
	}

	c.setCached(key, workloads)
	return workloads, nil
}

func (c *MCPClient) fetchK8sResources(ctx context.Context, parent, namespace, resourceType string) ([]models.Workload, error) {
    if c.OneMCPSession == nil {
        return nil, fmt.Errorf("OneMCP session nil")
    }

    args := map[string]interface{}{
		"parent":       parent,
		"resourceType": resourceType,
    }
    if namespace != "" {
        args["namespace"] = namespace
    }

	result, err := c.OneMCPSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "kube_get",
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}

    // Result should contain text content with YAML
    var yamlContent string
    if len(result.Content) > 0 {
        if text, ok := result.Content[0].(*mcp.TextContent); ok {
            yamlContent = text.Text
        }
    }

    if yamlContent == "" {
        // Try struct content fallback or error
        return nil, fmt.Errorf("empty response from kube_get")
    }

    // Parse YAML
    type K8sMetadata struct {
        Name string `yaml:"name"`
        Namespace string `yaml:"namespace"`
        CreationTimestamp string `yaml:"creationTimestamp,omitempty"`
    }
    type K8sStatus struct {
        Phase string `yaml:"phase"`
        Replicas int `yaml:"replicas,omitempty"`
        ReadyReplicas int `yaml:"readyReplicas,omitempty"`
        AvailableReplicas int `yaml:"availableReplicas,omitempty"`
    }
    type K8sItem struct {
        Kind string `yaml:"kind"`
        Metadata K8sMetadata `yaml:"metadata"`
        Status K8sStatus `yaml:"status"`
    }
    type K8sList struct {
        Items []K8sItem `yaml:"items"`
    }

    var list K8sList

	// OneMCP may return a JSON object with a list of YAML strings
	type OneMCPResponse struct {
		ResourcesYaml []string `json:"resourcesYaml"`
	}
	var resp OneMCPResponse
	if err := json.Unmarshal([]byte(yamlContent), &resp); err == nil && len(resp.ResourcesYaml) > 0 {
		for _, r := range resp.ResourcesYaml {
			var item K8sItem
			if err := yaml.Unmarshal([]byte(r), &item); err == nil {
				list.Items = append(list.Items, item)
			}
		}
	} else {
		// Fallback to standard K8s List YAML
		if err := yaml.Unmarshal([]byte(yamlContent), &list); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	}

    var workloads []models.Workload
    for _, item := range list.Items {
        status := "Unknown"
        ready := ""

        switch item.Kind { // Or resourceType
        case "Deployment":
             if item.Status.Replicas == item.Status.ReadyReplicas {
                 status = "Ready"
             } else {
                 status = "Progressing" // Simplified
             }
             ready = fmt.Sprintf("%d/%d", item.Status.ReadyReplicas, item.Status.Replicas)
        case "Service":
             status = "Active" // Services usually active
             ready = "N/A"
        default:
             // Fallback if kind missing (sometimes lists don't set it on items?)
             // Usually they do.
             if resourceType == "services" {
                 item.Kind = "Service"
                 status = "Active"
                 ready = "N/A"
             } else {
                 item.Kind = "Deployment"
                 status = "Ready" // fallback
                 ready = fmt.Sprintf("%d/%d", item.Status.ReadyReplicas, item.Status.Replicas)
             }
        }

        workloads = append(workloads, models.Workload{
            Name: item.Metadata.Name,
            Namespace: item.Metadata.Namespace,
            Type: item.Kind,
            Status: status,
            Ready: ready,
            Age: "1d", // difficult to calc relative time without extra lib or parsing logic, keeping simple
        })
    }
    return workloads, nil
}

// GetWorkload retrieves specific workload details by name.
// It currently filters the results from ListWorkloads.
func (c *MCPClient) GetWorkload(ctx context.Context, project, location, cluster, namespace, name string) (*models.Workload, error) {
	// Reusing ListWorkloads and filtering for simplicity
	workloads, err := c.ListWorkloads(ctx, project, location, cluster, namespace)
	if err != nil {
		return nil, err
	}
	for _, w := range workloads {
		if w.Name == name {
			return &w, nil
		}
	}
	return nil, fmt.Errorf("workload not found")
}

// ListPods retrieves pods for a specific workload.
func (c *MCPClient) ListPods(ctx context.Context, project, location, cluster, namespace, workloadName string) ([]models.Pod, error) {
    if c.OneMCPSession != nil && project != "" && location != "" && cluster != "" {
         parent := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)
         // Assuming fetchPods helper
         pods, err := c.fetchK8sPods(ctx, parent, namespace, workloadName)
         if err == nil {
             return pods, nil
         }
         log.Printf("Warning: Failed to fetch pods from OneMCP: %v", err)
    }

	if c.OSSMCPSession == nil {
		return nil, fmt.Errorf("OSSMCP session is not available")
	}

	key := fmt.Sprintf("pods:%s:%s", cluster, namespace)
	if data, ok := c.getCached(key); ok {
		return data.([]models.Pod), nil
	}

	// Fetch from OSSMCP
	result, err := c.OSSMCPSession.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		return nil, err
	}

	var pods []models.Pod
	for _, r := range result.Resources {
		// Mock logic: if resource name starts with workloadName + "-", treat as its pod.
		if strings.HasPrefix(r.Name, workloadName+"-") {
			pods = append(pods, models.Pod{
				Name:   r.Name,
				Status: "Running", // Mocked
				Age:    "1h",      // Mocked
			})
		}
	}

	c.setCached(key, pods)
	return pods, nil
}

func (c *MCPClient) fetchK8sPods(ctx context.Context, parent, namespace, workloadName string) ([]models.Pod, error) {
    if c.OneMCPSession == nil {
        return nil, fmt.Errorf("OneMCP session nil")
    }

    args := map[string]interface{}{
		"parent":       parent,
		"resourceType": "pods",
        "namespace": namespace,
        // labelSelector might be needed but for now filtering by name prefix or assuming we get all and filter
        // Ideally: "labelSelector": "app="+workloadName
    }

	result, err := c.OneMCPSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "kube_get",
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}

    var yamlContent string
    if len(result.Content) > 0 {
        if text, ok := result.Content[0].(*mcp.TextContent); ok {
			yamlContent = text.Text
		}
    }
    if yamlContent == "" {
        return nil, fmt.Errorf("empty response")
    }

    type K8sMetadata struct {
        Name string `yaml:"name"`
    }
    type K8sStatus struct {
        Phase string `yaml:"phase"`
    }
    type K8sItem struct {
        Metadata K8sMetadata `yaml:"metadata"`
        Status K8sStatus `yaml:"status"`
    }
    type K8sList struct {
        Items []K8sItem `yaml:"items"`
    }

    var list K8sList

	// OneMCP may return a JSON object with a list of YAML strings
	type OneMCPResponse struct {
		ResourcesYaml []string `json:"resourcesYaml"`
	}
	var resp OneMCPResponse
	if err := json.Unmarshal([]byte(yamlContent), &resp); err == nil && len(resp.ResourcesYaml) > 0 {
		for _, r := range resp.ResourcesYaml {
			var item K8sItem
			if err := yaml.Unmarshal([]byte(r), &item); err == nil {
				list.Items = append(list.Items, item)
			}
		}
	} else {
		if err := yaml.Unmarshal([]byte(yamlContent), &list); err != nil {
			return nil, fmt.Errorf("failed to parse: %w", err)
		}
	}

    var pods []models.Pod
    for _, item := range list.Items {
        // Simple filter if not using label selector
        // In reality, pods might handle different naming but usually start with workload if generated by it.
        // Better to use label selector in future task.
        if strings.HasPrefix(item.Metadata.Name, workloadName) {
            pods = append(pods, models.Pod{
                Name: item.Metadata.Name,
                Status: item.Status.Phase,
                Age: "1h",
            })
        }
    }
    return pods, nil
}

// ListTools returns a combined list of tools from all connected MCP servers (OneMCP and OSSMCP).
// This allows the backend to expose a unified toolset to agents or other clients.
func (c *MCPClient) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	var allTools []*mcp.Tool

	// 1. OneMCP
	if c.OneMCPSession != nil {
		res, err := c.OneMCPSession.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			log.Printf("Warning: Failed to list tools from OneMCP: %v", err)
		} else {
			allTools = append(allTools, res.Tools...)
		}
	}

	// 2. OSSMCP
	if c.OSSMCPSession != nil {
		res, err := c.OSSMCPSession.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			log.Printf("Warning: Failed to list tools from OSSMCP: %v", err)
		} else {
			allTools = append(allTools, res.Tools...)
		}
	}

	return allTools, nil
}

// ListPrompts returns a combined list of prompts from all connected MCP servers.
func (c *MCPClient) ListPrompts(ctx context.Context) ([]*mcp.Prompt, error) {
	var allPrompts []*mcp.Prompt

	// 1. OneMCP
	if c.OneMCPSession != nil {
		// Check capabilities
		initRes := c.OneMCPSession.InitializeResult()
		if initRes != nil && initRes.Capabilities != nil && initRes.Capabilities.Prompts != nil {
			res, err := c.OneMCPSession.ListPrompts(ctx, &mcp.ListPromptsParams{})
			if err != nil {
				log.Printf("Debug: OneMCP ListPrompts: %v", err)
			} else {
				allPrompts = append(allPrompts, res.Prompts...)
			}
		}
	}

	// 2. OSSMCP
	if c.OSSMCPSession != nil {
		initRes := c.OSSMCPSession.InitializeResult()
		if initRes != nil && initRes.Capabilities != nil && initRes.Capabilities.Prompts != nil {
			res, err := c.OSSMCPSession.ListPrompts(ctx, &mcp.ListPromptsParams{})
			if err != nil {
				log.Printf("Warning: Failed to list prompts from OSSMCP: %v", err)
			} else {
				allPrompts = append(allPrompts, res.Prompts...)
			}
		}
	}

	return allPrompts, nil
}

// ListResources returns a combined list of resources from all connected MCP servers.
func (c *MCPClient) ListResources(ctx context.Context) ([]*mcp.Resource, error) {
	var allResources []*mcp.Resource

	// 1. OneMCP
	if c.OneMCPSession != nil {
		initRes := c.OneMCPSession.InitializeResult()
		if initRes != nil && initRes.Capabilities != nil && initRes.Capabilities.Resources != nil {
			res, err := c.OneMCPSession.ListResources(ctx, &mcp.ListResourcesParams{})
			if err != nil {
				log.Printf("Debug: OneMCP ListResources: %v", err)
			} else {
				allResources = append(allResources, res.Resources...)
			}
		}
	}

	// 2. OSSMCP
	if c.OSSMCPSession != nil {
		initRes := c.OSSMCPSession.InitializeResult()
		if initRes != nil && initRes.Capabilities != nil && initRes.Capabilities.Resources != nil {
			res, err := c.OSSMCPSession.ListResources(ctx, &mcp.ListResourcesParams{})
			if err != nil {
				log.Printf("Warning: Failed to list resources from OSSMCP: %v", err)
			} else {
				allResources = append(allResources, res.Resources...)
			}
		}
	}

	return allResources, nil
}

// CallGenericTool calls a tool on the appropriate server.
// It attempts to call the tool on OneMCP first, and falls back to OSSMCP if not found or if OneMCP fails.
func (c *MCPClient) CallGenericTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Try OneMCP first
	if c.OneMCPSession != nil {
		res, err := c.OneMCPSession.CallTool(ctx, &mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		})
		// If success, return. If error strictly implies tool not found, define logic?
		// For now we assume if no error, it worked.
		// If error is JSONRPC code -32601 (Method not found), we try next.
		// But SDK might wrap error.
		// Simple approach: Use ListTools to find where it is? No, expensive.
		// Just try both? OneMCP is highly likely to contain GKE tools.
		if err == nil {
			return res, nil
		}
		// Check if error string contains "not found" or similar if possible.
		// But for now, let's just Log and try OSSMCP if OneMCP fails.
		// Actually, if OneMCP fails with "Internal Error" we shouldn't try OSSMCP?
		// "StatelessServer" of OneMCP returns error if tool not found?
	}

	if c.OSSMCPSession != nil {
		return c.OSSMCPSession.CallTool(ctx, &mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		})
	}

	return nil, fmt.Errorf("tool execution failed: no available session handled the request")
}
