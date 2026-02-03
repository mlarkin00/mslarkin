// Package mcpclient handles the connections to Model Context Protocol (MCP) servers.
package mcpclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s-status-backend/auth"
	"k8s-status-backend/models"
)

// MCPClient wraps connections to multiple MCP servers and provides
// high-level methods to interact with them, including caching.
type MCPClient struct {
	// OneMCP is the client for the GKE OneMCP server.
	OneMCP        *mcp.Client
	OneMCPSession *mcp.ClientSession

	// OSSMCP is the client for the GKE OSS MCP server.
	OSSMCP        *mcp.Client
	OSSMCPSession *mcp.ClientSession

	cache map[string]cacheEntry
	mu    sync.RWMutex
}

type cacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

const (
	OneMCPEndpoint = "https://container.googleapis.com/mcp"
	OSSMCPEndpoint = "https://mcp.ai.mslarkin.com"
	CacheTTL       = 30 * time.Second
)

// IDTokenTransport injects an OIDC ID token.
type IDTokenTransport struct {
	Base     http.RoundTripper
	Audience string
}

func (t *IDTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := auth.GetIDToken(req.Context(), t.Audience)
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
	token, err := auth.GetAccessToken(req.Context())
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
// It sets up OIDC authentication and SSE transport.
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
		log.Printf("Warning: Failed to connect to OneMCP: %v", err)
	} else {
		client.OneMCPSession = oneSession
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
		log.Printf("Warning: Failed to connect to OSSMCP: %v", err)
	} else {
		client.OSSMCPSession = ossSession
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
	entry, ok := c.cache[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Data, true
}

func (c *MCPClient) setCached(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = cacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(CacheTTL),
	}
}

// ListClusters retrieves the list of clusters from OneMCP.
// It uses caching to reduce latency.
func (c *MCPClient) ListClusters(ctx context.Context, projectID string) ([]models.Cluster, error) {
	if c.OneMCPSession == nil {
		return nil, fmt.Errorf("OneMCP session is not available")
	}
	key := "clusters:" + projectID
	if data, ok := c.getCached(key); ok {
		return data.([]models.Cluster), nil
	}

	// Fetch from OneMCP
	// We list resources.
	result, err := c.OneMCPSession.ListResources(ctx, nil)
	if err != nil {
		return nil, err
	}

	var clusters []models.Cluster
	for _, r := range result.Resources {
		// Simple mapping. In reality, we'd parse URI or Metadata.
		clusters = append(clusters, models.Cluster{
			Name:      r.Name,
			ProjectID: projectID, // Assumed
			Location:  "us-west1", // Default/Unknown
			Status:    "UNKNOWN",
		})
	}

	c.setCached(key, clusters)
	return clusters, nil
}

// ListWorkloads retrieves the list of workloads from OSSMCP.
// It uses caching to reduce latency.
func (c *MCPClient) ListWorkloads(ctx context.Context, cluster, namespace string) ([]models.Workload, error) {
	if c.OSSMCPSession == nil {
		return nil, fmt.Errorf("OSSMCP session is not available")
	}
	key := fmt.Sprintf("workloads:%s:%s", cluster, namespace)
	if data, ok := c.getCached(key); ok {
		return data.([]models.Workload), nil
	}

	// Fetch from OSSMCP
	result, err := c.OSSMCPSession.ListResources(ctx, nil)
	if err != nil {
		return nil, err
	}

	var workloads []models.Workload
	for _, r := range result.Resources {
        // Filter by namespace?
		workloads = append(workloads, models.Workload{
			Name:      r.Name,
			Namespace: namespace,
			Type:      "Deployment", // Mock type or infer from URI
			Status:    "Ready",
            Ready:     "1/1",
            Age:       "1d",
		})
	}

	c.setCached(key, workloads)
	return workloads, nil
}
