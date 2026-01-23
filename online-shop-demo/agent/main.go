package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mslarkin/online-shop-demo/agent/pkg/agent"
	"github.com/mslarkin/online-shop-demo/agent/pkg/gcp"
	"github.com/mslarkin/online-shop-demo/agent/pkg/k8s"
	"github.com/mslarkin/online-shop-demo/agent/ui"
)

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/apply", handleApply)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "mslarkin-ext" // Default
	}

	var clusterNames []string
	if project != "" {
		clusters, err := gcp.ListClusters(r.Context(), project)
		if err != nil {
			log.Printf("Error listing clusters: %v", err)
			// Don't fail completely, just show empty list or error
		} else {
			for _, c := range clusters {
				clusterNames = append(clusterNames, c.Name)
			}
		}
	}

	rootDir := os.Getenv("APP_ROOT")
	if rootDir == "" {
		rootDir, _ = filepath.Abs("..") // Assuming we run from agent/ locally
	}
	// failureModes is now []k8s.FailureMode
	k8sModes, err := k8s.GetFailureModes(rootDir)
	if err != nil {
		log.Printf("Error getting failure modes: %v", err)
	}

	var uiModes []ui.FailureMode
	for _, m := range k8sModes {
		uiModes = append(uiModes, ui.FailureMode{
			Name:        m.Name,
			Description: m.Description,
		})
	}

	page := ui.Layout("Failure Mode Simulator", ui.Dashboard([]string{project}, project, clusterNames, uiModes))
	w.Header().Set("Content-Type", "text/html")
	_ = page.Render(w)
}

func handleApply(w http.ResponseWriter, r *http.Request) {
	mode := r.FormValue("mode")
	action := r.FormValue("action")
	project := r.FormValue("project")
	cluster := r.FormValue("cluster")

	if project == "" {
		project = "mslarkin-ext"
	}

	rootDir := os.Getenv("APP_ROOT")
	if rootDir == "" {
		rootDir, _ = filepath.Abs("..")
	}

	// Initialize Agent
	ctx := r.Context()

	// Lookup cluster location since valid prompt needs it for tool call
	var location string
	if project != "" && cluster != "" {
		clusters, err := gcp.ListClusters(ctx, project)
		if err == nil {
			for _, c := range clusters {
				if c.Name == cluster {
					location = c.Location
					break
				}
			}
		}
	}
	if location == "" {
		location = "us-central1" // Fallback or let agent guess/ask (but agent has no chat UI)
		// Or maybe we should error? But let's try fallback.
	}

	agentService, err := agent.NewAgent(ctx, project, "us-central1", "gemini-2.0-flash-001", rootDir)
	if err != nil {
		fmt.Fprintf(w, "<span class='text-red-500'>Failed to init agent: %v</span>", err)
		return
	}
	defer agentService.Close()

	var prompt string
	if action == "apply" {
		prompt = fmt.Sprintf("Please apply the '%s' failure mode to cluster '%s' in project '%s', location '%s'.", mode, cluster, project, location)
	} else {
		prompt = fmt.Sprintf("Please revert the '%s' failure mode on cluster '%s' in project '%s', location '%s'.", mode, cluster, project, location)
	}

	response, err := agentService.Run(ctx, prompt)
	if err != nil {
		log.Printf("Agent failed: %v", err)
		fmt.Fprintf(w, "<span class='text-red-500'>Agent Error: %v</span>", err)
		return
	}

	// Success response from Agent
	fmt.Fprintf(w, "<span class='text-green-500'>Agent: %s</span>", response)
}
