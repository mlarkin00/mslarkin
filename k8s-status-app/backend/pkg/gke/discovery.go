package gke

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	containerpb "cloud.google.com/go/container/apiv1/containerpb"
)

// ClusterInfo contains minimal info needed to connect to a cluster
type ClusterInfo struct {
	Name      string `json:"name"`
	Location  string `json:"location"`
	Endpoint  string `json:"endpoint"`
	ProjectID string `json:"project_id"`
	Status    string `json:"status"`
    // CaCert is the base64 encoded cluster CA certificate
    CaCert    string `json:"-"`
}

type DiscoveryClient struct {
	client *container.ClusterManagerClient
}

func NewDiscoveryClient(ctx context.Context) (*DiscoveryClient, error) {
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster manager client: %w", err)
	}
	return &DiscoveryClient{client: c}, nil
}

func (d *DiscoveryClient) Close() error {
	return d.client.Close()
}

// ListClusters returns a list of clusters for the given project IDs
func (d *DiscoveryClient) ListClusters(ctx context.Context, projectIDs []string) ([]ClusterInfo, error) {
	var clusters []ClusterInfo

	for _, pid := range projectIDs {
		// List clusters in the project (all locations)
		parent := fmt.Sprintf("projects/%s/locations/-", pid)
		req := &containerpb.ListClustersRequest{
			Parent: parent,
		}

		resp, err := d.client.ListClusters(ctx, req)
		if err != nil {
			// Return error for now, maybe log and continue in future if partial success needed
			return nil, fmt.Errorf("failed to list clusters in project %s: %w", pid, err)
		}

		for _, c := range resp.Clusters {
            caCert := ""
            if c.MasterAuth != nil {
                caCert = c.MasterAuth.ClusterCaCertificate
            }

			clusters = append(clusters, ClusterInfo{
				Name:      c.Name,
				Location:  c.Location,
				Endpoint:  c.Endpoint,
				ProjectID: pid,
				Status:    c.Status.String(),
                CaCert:    caCert,
			})
		}
	}

	return clusters, nil
}
