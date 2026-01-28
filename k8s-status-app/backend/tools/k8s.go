package tools

import (
	"context"
	"fmt"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"k8s-status-backend/a2ui"
	"k8s-status-backend/mcpclient"
)

type K8sTools struct {
	Client *mcpclient.MCPClient
}

func NewK8sTools(client *mcpclient.MCPClient) *K8sTools {
	return &K8sTools{Client: client}
}

func (t *K8sTools) GetTools() []tool.Tool {
	listClusters, _ := functiontool.New(functiontool.Config{
		Name: "list_clusters",
		Description: "List available GKE clusters. Returns A2UI cards.",
	}, t.ListClusters)

	listWorkloads, _ := functiontool.New(functiontool.Config{
		Name: "list_workloads",
		Description: "List workloads in a cluster. Returns A2UI components.",
	}, t.ListWorkloads)

	return []tool.Tool{listClusters, listWorkloads}
}

type ListClustersArgs struct {
	ProjectID string `json:"project_id"`
}

func (t *K8sTools) ListClusters(ctx tool.Context, args ListClustersArgs) (a2ui.Component, error) {
	projectID := args.ProjectID
	if projectID == "" {
		projectID = "mslarkin-ext"
	}

	clusters, err := t.Client.ListClusters(context.Background(), projectID)
	if err != nil {
		return a2ui.Text("Error listing clusters"), fmt.Errorf("failed to list clusters: %w", err)
	}

	var clusterCards []a2ui.Component
	for _, c := range clusters {
		clusterCards = append(clusterCards, a2ui.Card(
			c.Name,
			a2ui.Text("Location: "+c.Location),
			a2ui.Text("Status: "+c.Status),
			a2ui.Container(
				a2ui.Button("View Workloads", fmt.Sprintf("Show workloads for cluster %s", c.Name)),
			),
		))
	}

	return a2ui.Container(clusterCards...), nil
}

type ListWorkloadsArgs struct {
	Cluster string `json:"cluster"`
	Namespace string `json:"namespace"`
}

func (t *K8sTools) ListWorkloads(ctx tool.Context, args ListWorkloadsArgs) (a2ui.Component, error) {
	if args.Cluster == "" {
		return a2ui.Text("Cluster name required"), fmt.Errorf("cluster argument required")
	}
	namespace := args.Namespace
	if namespace == "" {
		namespace = "default"
	}

	workloads, err := t.Client.ListWorkloads(context.Background(), args.Cluster, namespace)
	if err != nil {
		return a2ui.Text("Error listing workloads"), fmt.Errorf("failed to list workloads: %w", err)
	}

	var workloadCards []a2ui.Component
	for _, w := range workloads {
		workloadCards = append(workloadCards, a2ui.Card(
			w.Name,
			a2ui.Text(fmt.Sprintf("%s | %s", w.Type, w.Status)),
			a2ui.Container(
				a2ui.Button("Pods", fmt.Sprintf("Show pods for workload %s in cluster %s", w.Name, args.Cluster)),
				a2ui.Button("Describe", fmt.Sprintf("Describe workload %s in cluster %s", w.Name, args.Cluster)),
			),
		))
	}

	return a2ui.Container(
		a2ui.Text(fmt.Sprintf("Workloads in %s/%s", args.Cluster, namespace)),
		a2ui.Container(workloadCards...),
	), nil
}
