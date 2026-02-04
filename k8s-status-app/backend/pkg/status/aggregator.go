package status

import (
	"context"
	"fmt"
    "sync"
    "strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    gke "k8s-status-backend/pkg/gke"
    "k8s-status-backend/pkg/k8s"
)

type WorkloadStatus struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
    Kind      string `json:"kind"`
    Desired   int32  `json:"desired"`
    Ready     int32  `json:"ready"`
    Status    string `json:"status"` // "Healthy", "Degraded", "Progressing"
    Message   string `json:"message"`
}

type ClusterStatus struct {
    ClusterName string           `json:"cluster_name"`
    ProjectID   string           `json:"project_id"`
    Location    string           `json:"location"`
    NodeCount   int              `json:"node_count"`
    NodesReady  int              `json:"nodes_ready"`
    Workloads   []WorkloadStatus `json:"workloads"`
    Error       string           `json:"error,omitempty"`
}

type Aggregator struct {
    clients *k8s.ClientManager
}

func NewAggregator(cm *k8s.ClientManager) *Aggregator {
    return &Aggregator{clients: cm}
}

func (a *Aggregator) FetchAll(ctx context.Context, clusters []gke.ClusterInfo) []ClusterStatus {
    var wg sync.WaitGroup
    results := make([]ClusterStatus, len(clusters))

    for i, c := range clusters {
        wg.Add(1)
        go func(idx int, cluster gke.ClusterInfo) {
            defer wg.Done()
            results[idx] = a.GetClusterStatus(ctx, cluster)
        }(i, c)
    }
    wg.Wait()
    return results
}

func (a *Aggregator) GetClusterStatus(ctx context.Context, cluster gke.ClusterInfo) ClusterStatus {
    status := ClusterStatus{
        ClusterName: cluster.Name,
        ProjectID:   cluster.ProjectID,
        Location:    cluster.Location,
    }

    client, err := a.clients.GetClient(ctx, cluster)
    if err != nil {
        status.Error = fmt.Sprintf("Failed to create client: %v", err)
        return status
    }

    // Get Nodes
    nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
    if err != nil {
        status.Error = fmt.Sprintf("Failed to list nodes: %v", err)
        return status
    }
    status.NodeCount = len(nodes.Items)
    for _, node := range nodes.Items {
        for _, cond := range node.Status.Conditions {
            if cond.Type == "Ready" && cond.Status == "True" {
                status.NodesReady++
                break
            }
        }
    }

    // Get Deployments (across all namespaces for now, or filter?)
    // User requested "workload information".
    // Let's list Deployments in all namespaces.
    deps, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
    if err == nil {
        for _, d := range deps.Items {
            // Simple status check
            s := "Healthy"
            msg := ""
            if d.Status.Replicas != d.Status.ReadyReplicas {
                s = "Degraded"
                msg = fmt.Sprintf("%d/%d ready", d.Status.ReadyReplicas, d.Status.Replicas)
            }
            // Ignore kube-system? Maybe distinct?
            // For status app, maybe show everything but filter in UI.
            if strings.HasPrefix(d.Namespace, "kube-") {
                continue // Skip system namespaces for cleaner view? OR keep?
                // Let's keep system out for demo clarity unless requested.
            }

            status.Workloads = append(status.Workloads, WorkloadStatus{
                Name:      d.Name,
                Namespace: d.Namespace,
                Kind:      "Deployment",
                Desired:   d.Status.Replicas,
                Ready:     d.Status.ReadyReplicas,
                Status:    s,
                Message:   msg,
            })
        }
    } else {
        // Just log/append error to msg
        if status.Error == "" {
             status.Error = fmt.Sprintf("Failed to list deployments: %v", err)
        } else {
             status.Error += fmt.Sprintf("; Failed to list deployments: %v", err)
        }
    }

    return status
}
