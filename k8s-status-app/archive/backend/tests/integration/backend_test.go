package integration

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s-status-backend/mcpclient"
	"k8s-status-backend/models"
)

// MockMCPSession implements mcpclient.MCPSession
type MockMCPSession struct {
	Clusters  []models.Cluster
	Workloads []models.Workload
}

func (m *MockMCPSession) ListResources(ctx context.Context, params *mcp.ListResourcesParams) (*mcp.ListResourcesResult, error) {
	var resources []*mcp.Resource
	for _, c := range m.Clusters {
		// Mimic OneMCP response which returns resources using valid Name format for parsing logic.
		// Logic expects: //container.googleapis.com/projects/{project}/locations/{location}/clusters/{cluster}
		// or simple suffix checking depending on impl.

		name := c.Name
		if c.Location != "" {
			// Construct a "valid" OneMCP name for testing the parser
			name = "//container.googleapis.com/projects/" + c.ProjectID + "/locations/" + c.Location + "/clusters/" + c.Name
		}

		resources = append(resources, &mcp.Resource{
			Name: name,
		})
	}
	for _, w := range m.Workloads {
		// Mimic OSSMCP response which returns resources.
		resources = append(resources, &mcp.Resource{
			Name: w.Name,
		})
		// Also add a "Pod" resource to test ListPods mock logic
		// Mock logic in client.go: if resource name starts with workloadName + "-", treat as its pod.
		podName := w.Name + "-pod-123"
		resources = append(resources, &mcp.Resource{
			Name: podName,
		})
	}
	return &mcp.ListResourcesResult{Resources: resources}, nil
}

func (m *MockMCPSession) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	if params.Name == "list_clusters" {
		var clusters []interface{}
		for _, c := range m.Clusters {
			clusters = append(clusters, map[string]interface{}{
				"name":     c.Name,
				"location": c.Location,
				"status":   "RUNNING",
			})
		}
		return &mcp.CallToolResult{
			StructuredContent: map[string]interface{}{
				"clusters": clusters,
			},
		}, nil
	}
	return &mcp.CallToolResult{}, nil
}

func (m *MockMCPSession) ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{}, nil
}

func (m *MockMCPSession) ListPrompts(ctx context.Context, params *mcp.ListPromptsParams) (*mcp.ListPromptsResult, error) {
	return &mcp.ListPromptsResult{}, nil
}

func (m *MockMCPSession) InitializeResult() *mcp.InitializeResult {
	// Mock basic capabilities
	return &mcp.InitializeResult{
		Capabilities: &mcp.ServerCapabilities{
			Prompts:   &mcp.PromptCapabilities{},
			Resources: &mcp.ResourceCapabilities{},
			Tools:     &mcp.ToolCapabilities{},
		},
	}
}

func (m *MockMCPSession) Close() error {
	return nil
}

func TestClusterLocation_Logic(t *testing.T) {
	// Setup Mock Data with explicit location
	mockClusters := []models.Cluster{
		{Name: "ai-auto-cluster", ProjectID: "mslarkin-ext", Location: "us-central1"},
	}

	mockSession := &MockMCPSession{
		Clusters: mockClusters,
	}

	ctx := context.Background()
	client := &mcpclient.MCPClient{
		OneMCPSession: mockSession,
	}

	backendClusters, err := client.ListClusters(ctx, "mslarkin-ext")
	if err != nil {
		t.Fatalf("Failed to list clusters: %v", err)
	}

	t.Logf("Fetched %d clusters from Backend", len(backendClusters))

	if len(backendClusters) == 0 {
		t.Fatal("Expected clusters, got none")
	}

	found := false
	for _, bc := range backendClusters {
		// The Name returned by client might be the full Resource Name depending on implementation.
		// Current impl just puts r.Name into Cluster.Name.
		// So we should check if it contains our cluster name.
		match := bc.Location == "us-central1"
		t.Logf("Backend Cluster: Name=%q Location=%q | Ground Truth Location: us-central1 | Match: %v", bc.Name, bc.Location, match)

		if match {
			found = true
		}
	}
	if !found {
		t.Error("Backend did not correctly parse location 'us-central1' from resource name")
	}
}

func TestWorkloadList_Logic(t *testing.T) {
	mockWorkloads := []models.Workload{
		{Name: "checkout-service", Namespace: "default", Type: "Deployment", Status: "Ready"},
	}

	mockSession := &MockMCPSession{
		Workloads: mockWorkloads,
	}

	ctx := context.Background()
	client := &mcpclient.MCPClient{
		OSSMCPSession: mockSession,
	}

	workloads, err := client.ListWorkloads(ctx, "", "", "ai-auto-cluster", "default")
	if err != nil {
		t.Fatalf("Failed to list workloads: %v", err)
	}

	t.Logf("Fetched %d workloads from Backend", len(workloads))

	found := false
	for _, w := range workloads {
		if w.Name == "checkout-service" {
			found = true

			matchNs := w.Namespace == "default"
			t.Logf("Backend Workload: Name=%q Namespace=%q | Ground Truth: Name=checkout-service Namespace=default | Match: %v", w.Name, w.Namespace, matchNs)

			if !matchNs {
				t.Errorf("Backend returned namespace %q, expected %q", w.Namespace, "default")
			}
		} else {
			t.Logf("Backend returned extra workload: %q", w.Name)
		}
	}
	if !found {
		t.Error("Workload checkout-service not found in backend response")
	}
}

func TestListPods_Logic(t *testing.T) {
	mockWorkloads := []models.Workload{
		{Name: "checkout-service", Namespace: "default"},
	}

	mockSession := &MockMCPSession{
		Workloads: mockWorkloads,
	}

	ctx := context.Background()
	client := &mcpclient.MCPClient{
		OSSMCPSession: mockSession,
	}

	pods, err := client.ListPods(ctx, "", "", "ai-auto-cluster", "default", "checkout-service")
	if err != nil {
		t.Fatalf("Failed to list pods: %v", err)
	}

	t.Logf("Fetched %d pods from Backend", len(pods))

	if len(pods) == 0 {
		t.Fatal("Expected pods, got none")
	}

	// Expect at least one pod starting with checkout-service-
	found := false
	for _, p := range pods {
		isMatch := p.Status == "Running"
		t.Logf("Backend Pod: Name=%q Status=%q | Ground Truth Pattern: checkout-service-* Status=Running | Match: %v", p.Name, p.Status, isMatch)

		if isMatch { // Check a field we mocked
			found = true
		}
	}
	if !found {
		t.Error("Expected valid pod in response")
	}
}
