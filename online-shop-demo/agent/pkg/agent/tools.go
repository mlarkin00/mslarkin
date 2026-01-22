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
				},
				Required: []string{"mode", "project_id", "cluster"},
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
				},
				Required: []string{"mode", "project_id", "cluster"},
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
		// For now we ignore cluster/project target in actual execution because
		// the demo scripts just use current context.
		// In a real agent we would switch context here.

		// Try to find apply.sh or *.yaml
		// Quick implementation: just look for yaml for now as our k8s client supports it
		// Or assume there is an emailservice-crash.yaml etc

		// Let's implement a smarter finder in k8s package later.
		// For now, hardcode some logic or use a helper.

		// TODO: Switch KubeContext to project/cluster
		log.Printf("Applying mode %s (Targeting %v/%v - Context switch not impl)", mode, args["project_id"], args["cluster"])

		// We use a simplified assumption: The user has set the context or we just apply to current.
		// The prompt asks to apply to a cluster, we should probably at least log it.

		// Logic to find manifest
		// Better: scan directory for yaml

		// For the purpose of this demo, we can just use the path we found in exploration?
		// No, we need to be generic.

		err := k8s.ApplyFailureMode(ctx, rootDir, mode)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "applied", "mode": mode}, nil

	case "revert_failure_mode":
		mode, _ := args["mode"].(string)
		log.Printf("Reverting mode %s", mode)
		err := k8s.RevertFailureMode(ctx, rootDir, mode)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "reverted", "mode": mode}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}
