package gcp

import (
	"context"
	"fmt"

	container "google.golang.org/api/container/v1"
	"google.golang.org/api/option"
)

type Cluster struct {
	Name     string
	Location string
	Status   string
}

// ListClusters lists all GKE clusters in the given project.
func ListClusters(ctx context.Context, projectID string) ([]Cluster, error) {
	svc, err := container.NewService(ctx, option.WithScopes(container.CloudPlatformScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create container service: %w", err)
	}

	// Parent format: "projects/{project_id}/locations/-" to list in all locations
	parent := fmt.Sprintf("projects/%s/locations/-", projectID)
	resp, err := svc.Projects.Locations.Clusters.List(parent).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var clusters []Cluster
	for _, c := range resp.Clusters {
		clusters = append(clusters, Cluster{
			Name:     c.Name,
			Location: c.Location,
			Status:   c.Status,
		})
	}

	return clusters, nil
}
