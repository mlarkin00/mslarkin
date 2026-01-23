package agent

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/vertexai/genai"
	"github.com/mslarkin/online-shop-demo/agent/pkg/gcp"
	"github.com/mslarkin/online-shop-demo/agent/pkg/k8s"
)

// Define the tool schema
var clusterTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{
		{
			Name:        "list_clusters",
			Description: "List GKE clusters in the specified project.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"project_id": {
						Type:        genai.TypeString,
						Description: "The Google Cloud Project ID.",
					},
				},
				Required: []string{"project_id"},
			},
		},
		{
			Name:        "apply_failure_mode",
			Description: "Apply a specific failure mode to a cluster.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"mode": {
						Type:        genai.TypeString,
						Description: "The name of the failure mode to apply (e.g., 'crashloop', 'image-pull', 'oom').",
					},
					"project_id": {
						Type:        genai.TypeString,
						Description: "The Google Cloud Project ID.",
					},
					"cluster": {
						Type:        genai.TypeString,
						Description: "The name of the target GKE cluster.",
					},
					"location": {
						Type:        genai.TypeString,
						Description: "The location (region/zone) of the target GKE cluster.",
					},
				},
				Required: []string{"mode", "project_id", "cluster", "location"},
			},
		},
		{
			Name:        "revert_failure_mode",
			Description: "Revert (delete/fix) a specific failure mode on a cluster.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"mode": {
						Type:        genai.TypeString,
						Description: "The name of the failure mode to revert.",
					},
					"project_id": {
						Type:        genai.TypeString,
						Description: "The Google Cloud Project ID.",
					},
					"cluster": {
						Type:        genai.TypeString,
						Description: "The name of the target GKE cluster.",
					},
					"location": {
						Type:        genai.TypeString,
						Description: "The location (region/zone) of the target GKE cluster.",
					},
				},
				Required: []string{"mode", "project_id", "cluster", "location"},
			},
		},
	},
}

// executeTool handles the actual execution of tools
func executeTool(ctx context.Context, name string, args map[string]interface{}, rootDir string) (map[string]interface{}, error) {
	switch name {
	case "list_clusters":
		projectID, _ := args["project_id"].(string)
		clusters, err := gcp.ListClusters(ctx, projectID)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"clusters": clusters}, nil

	case "apply_failure_mode":
		mode, _ := args["mode"].(string)
		projectID, _ := args["project_id"].(string)
		cluster, _ := args["cluster"].(string)
		location, _ := args["location"].(string)

		// Configure credentials for kubectl
		if err := k8s.ConfigureCredentials(ctx, projectID, location, cluster); err != nil {
			return nil, fmt.Errorf("failed to configure credentials: %w", err)
		}

		log.Printf("Applying mode %s to %s/%s in %s", mode, projectID, cluster, location)
		err := k8s.ApplyFailureMode(ctx, rootDir, mode)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "applied", "mode": mode}, nil

	case "revert_failure_mode":
		mode, _ := args["mode"].(string)
		projectID, _ := args["project_id"].(string)
		cluster, _ := args["cluster"].(string)
		location, _ := args["location"].(string)

		// Configure credentials for kubectl
		if err := k8s.ConfigureCredentials(ctx, projectID, location, cluster); err != nil {
			return nil, fmt.Errorf("failed to configure credentials: %w", err)
		}

		log.Printf("Reverting mode %s on %s/%s in %s", mode, projectID, cluster, location)
		err := k8s.RevertFailureMode(ctx, rootDir, mode)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "reverted", "mode": mode}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}
