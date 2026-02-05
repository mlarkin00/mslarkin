package status

import (
	"context"
	"fmt"
    "sync"
    "strings"
    "time"

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
    Age       string `json:"age"`
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

func formatAge(t metav1.Time) string {
    if t.IsZero() {
        return ""
    }
    d := time.Since(t.Time)
    if d.Hours() > 24 {
        return fmt.Sprintf("%dd", int(d.Hours()/24))
    }
    if d.Hours() > 1 {
        return fmt.Sprintf("%dh", int(d.Hours()))
    }
    if d.Minutes() > 1 {
        return fmt.Sprintf("%dm", int(d.Minutes()))
    }
    return fmt.Sprintf("%ds", int(d.Seconds()))
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

    // Get Deployments
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
            // Filter system namespaces if desired (currently keeping all as per logic)
            if strings.HasPrefix(d.Namespace, "kube-") {
               continue
            }

            status.Workloads = append(status.Workloads, WorkloadStatus{
                Name:      d.Name,
                Namespace: d.Namespace,
                Kind:      "Deployment",
                Desired:   d.Status.Replicas,
                Ready:     d.Status.ReadyReplicas,
                Status:    s,
                Message:   msg,
                Age:       formatAge(d.CreationTimestamp),
            })
        }
    } else {
        if status.Error == "" {
             status.Error = fmt.Sprintf("Failed to list deployments: %v", err)
        } else {
             status.Error += fmt.Sprintf("; Failed to list deployments: %v", err)
        }
    }

    // Get Services
    svcs, err := client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
    if err == nil {
        for _, svc := range svcs.Items {
             // Filter system namespaces
             if strings.HasPrefix(svc.Namespace, "kube-") {
                continue
             }

             s := "Healthy"
             msg := string(svc.Spec.Type)
             // Simple service status check? (maybe LoadBalancer IP pending?)
             if svc.Spec.Type == "LoadBalancer" {
                 if len(svc.Status.LoadBalancer.Ingress) == 0 {
                     s = "Progressing"
                     msg += " (Pending IP)"
                 } else {
                     msg += fmt.Sprintf(" (%s)", svc.Status.LoadBalancer.Ingress[0].IP)
                 }
             }

             status.Workloads = append(status.Workloads, WorkloadStatus{
                Name:      svc.Name,
                Namespace: svc.Namespace,
                Kind:      "Service",
                Desired:   1, // Service is always 1 logical entity?
                Ready:     1,
                Status:    s,
                Message:   msg,
                Age:       formatAge(svc.CreationTimestamp),
            })
        }
    } else {
        if status.Error == "" {
             status.Error = fmt.Sprintf("Failed to list services: %v", err)
        } else {
             status.Error += fmt.Sprintf("; Failed to list services: %v", err)
        }
    }

    return status
}

type PodStatus struct {
	Name   string `json:"name"`
	Phase  string `json:"phase"`
	IP     string `json:"pod_ip"`
	Node   string `json:"node_name"`
	Age    string `json:"age"`
}

func (a *Aggregator) GetWorkloadPods(ctx context.Context, cluster gke.ClusterInfo, namespace, workloadName string) ([]PodStatus, error) {
	client, err := a.clients.GetClient(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	// 1. Get Deployment to find selector
	// Note: We are assuming Deployment for now as per previous logic.
	// If we support Services/StatefulSets later we need generic logic or a switch.
	dep, err := client.AppsV1().Deployments(namespace).Get(ctx, workloadName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s: %v", workloadName, err)
	}

	// 2. List Pods using selector
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("invalid selector: %v", err)
	}

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}

	var results []PodStatus
	for _, p := range pods.Items {
		results = append(results, PodStatus{
			Name:  p.Name,
			Phase: string(p.Status.Phase),
			IP:    p.Status.PodIP,
			Node:  p.Spec.NodeName,
			Age:   formatAge(p.CreationTimestamp),
		})
	}

	return results, nil
}
