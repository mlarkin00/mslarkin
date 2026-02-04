package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"k8s-status-backend/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AccessTokenTransport (copied from client.go for standalone test)
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

func main() {
	ctx := context.Background()
	// 1. Setup Transport
	endpoint := "https://container.googleapis.com/mcp"
	transport := &mcp.StreamableClientTransport{
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Transport: &AccessTokenTransport{},
			Timeout:   30 * time.Second,
		},
	}

	// 2. Connect
	log.Printf("Connecting to OneMCP (%s)...", endpoint)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-mcp", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()
	log.Println("Connected!")

	// 3. Ping
	// 3. Call list_clusters
	log.Println("Calling list_clusters...")
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_clusters",
		Arguments: map[string]interface{}{
			"parent": "projects/mslarkin-demo/locations/-",
		},
	})
	if err != nil {
		log.Printf("CallTool list_clusters failed: %v", err)
	} else {
		log.Printf("CallTool list_clusters result: %+v", result)
	}

	// 4. List Tools
	log.Println("Listing Tools...")
	toolsReq := &mcp.ListToolsParams{}
	toolsRes, err := session.ListTools(ctx, toolsReq)
	if err != nil {
		log.Printf("Failed to ListTools: %v", err)
	} else {
		log.Printf("Found %d tools:", len(toolsRes.Tools))
		for _, t := range toolsRes.Tools {
			fmt.Printf("- Tool: %s\n", t.Name)
		}
	}

	// 5. List Resources (Skipped)
	log.Println("Skipping ListResources (not supported by OneMCP)")
}
// Helper to prevent unused var error if we commented out stuff
var _ = fmt.Printf

