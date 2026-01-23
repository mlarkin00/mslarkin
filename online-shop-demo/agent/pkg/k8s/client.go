package k8s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ApplyManifest applies the given manifest file using kubectl.
// We use kubectl exec for simplicity as parsing arbitrary YAMLs and mapping to GVRs via client-go is complex.
func ApplyManifest(ctx context.Context, manifestPath string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", manifestPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply manifest %s: %w", manifestPath, err)
	}
	return nil
}

// DeleteManifest deletes the resources defined in the manifest file using kubectl.
func DeleteManifest(ctx context.Context, manifestPath string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "delete", "-f", manifestPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete manifest %s: %w", manifestPath, err)
	}
	return nil
}

// ConfigureCredentials configures kubectl to communicate with the specified cluster.
func ConfigureCredentials(ctx context.Context, projectID, location, cluster string) error {
	// gcloud container clusters get-credentials CLUSTER --region REGION --project PROJECT
	cmd := exec.CommandContext(ctx, "gcloud", "container", "clusters", "get-credentials", cluster,
		"--region", location,
		"--project", projectID,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to get-credentials for %s: %w", cluster, err)
	}
	return nil
}

// FailureMode represents a chaos scenario
type FailureMode struct {
	Name        string
	Description string
}

// GetFailureModes returns a list of available failure modes by scanning the failure-modes directory.
func GetFailureModes(rootDir string) ([]FailureMode, error) {
	modesDir := filepath.Join(rootDir, "failure-modes")
	entries, err := os.ReadDir(modesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read failure modes directory: %w", err)
	}

	var modes []FailureMode
	for _, e := range entries {
		if e.IsDir() {
			descPath := filepath.Join(modesDir, e.Name(), "description.txt")
			descBytes, _ := os.ReadFile(descPath)
			description := string(descBytes)
			if description == "" {
				description = "No description available."
			}
			modes = append(modes, FailureMode{
				Name:        e.Name(),
				Description: description,
			})
		}
	}
	return modes, nil
}

// ApplyFailureMode applies the failure mode by finding the relevant manifest or script.
func ApplyFailureMode(ctx context.Context, rootDir, mode string) error {
	modeDir := filepath.Join(rootDir, "failure-modes", mode)

	// Check for apply.sh first
	scriptPath := filepath.Join(modeDir, "apply.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		// For now, we prefer to find the YAML if possible for k8s ApplyManifest compatibility,
        // but if we implemented ExecuteScript we could use that.
        // Let's pass for now and look for YAML.
	}

	// Find the first YAML file
	entries, err := os.ReadDir(modeDir)
	if err != nil {
		return fmt.Errorf("failed to read mode directory: %w", err)
	}

	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".yaml" || filepath.Ext(e.Name()) == ".yml" {
			manifestPath := filepath.Join(modeDir, e.Name())
			return ApplyManifest(ctx, manifestPath)
		}
	}

	return fmt.Errorf("no suitable manifest found for mode %s", mode)
}

// RevertFailureMode reverts the failure mode.
// For this demo, we apply the baseline manifest to reset state.
func RevertFailureMode(ctx context.Context, rootDir, mode string) error {
	baselinePath := filepath.Join(rootDir, "baseline", "release", "kubernetes-manifests.yaml")
	return ApplyManifest(ctx, baselinePath)
}
