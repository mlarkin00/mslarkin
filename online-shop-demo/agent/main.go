package main

import (
	"fmt"
	"log"
	"net/http"
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

	rootDir, _ := filepath.Abs("..") // Assuming we run from agent/
	failureModes, err := k8s.GetFailureModes(rootDir)
	if err != nil {
		log.Printf("Error getting failure modes: %v", err)
	}

	page := ui.Layout("Failure Mode Simulator", ui.Dashboard([]string{project}, project, clusterNames, failureModes))
	w.Header().Set("Content-Type", "text/html")
	_ = page.Render(w)
}

func handleApply(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	action := r.URL.Query().Get("action")
	project := r.URL.Query().Get("project")
	cluster := r.URL.Query().Get("cluster")

	if project == "" {
		project = "mslarkin-ext"
	}

	rootDir, _ := filepath.Abs("..")

	// Initialize Agent
	ctx := r.Context()
	agentService, err := agent.NewAgent(ctx, project, "us-central1", "gemini-2.0-flash-001", rootDir)
	if err != nil {
		fmt.Fprintf(w, "<span class='text-red-500'>Failed to init agent: %v</span>", err)
		return
	}
	defer agentService.Close()

	var prompt string
	if action == "apply" {
		prompt = fmt.Sprintf("Please apply the '%s' failure mode to cluster '%s' in project '%s'.", mode, cluster, project)
	} else {
		prompt = fmt.Sprintf("Please revert the '%s' failure mode on cluster '%s' in project '%s'.", mode, cluster, project)
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
