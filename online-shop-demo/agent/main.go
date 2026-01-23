package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mslarkin/online-shop-demo/agent/pkg/a2ui"
	"github.com/mslarkin/online-shop-demo/agent/pkg/agent"
	"github.com/mslarkin/online-shop-demo/agent/pkg/gcp"
	"github.com/mslarkin/online-shop-demo/agent/pkg/k8s"
)

func main() {
	rootDir := os.Getenv("APP_ROOT")
	if rootDir == "" {
		rootDir, _ = filepath.Abs("..")
	}

	// Initialize A2UI Server
	a2uiServer := a2ui.NewServer()

	// Push initial state
	project := "mslarkin-ext" // Default
	pushInitialState(a2uiServer, rootDir, project)

	// Handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "ui/client.html")
	})
	http.HandleFunc("/stream", a2uiServer.HandleStream)
	http.HandleFunc("/action", func(w http.ResponseWriter, r *http.Request) {
		a2uiServer.HandleAction(w, r, func(action a2ui.UserAction) {
			handleUserAction(a2uiServer, rootDir, action)
		})
	})

	log.Println("Agent starting on :8080 (A2UI)")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func pushInitialState(s *a2ui.Server, rootDir, project string) {
	// Build components
	k8sModes, err := k8s.GetFailureModes(rootDir)
	if err != nil {
		log.Printf("Error getting failure modes: %v", err)
	}

	ctx := context.Background()
	clusters, _ := gcp.ListClusters(ctx, project)
	clusterNames := []string{}
	for _, c := range clusters {
		clusterNames = append(clusterNames, c.Name)
	}

	// Construct A2UI Components
	var components []a2ui.ComponentWrapper

	// Root Column
	components = append(components, a2ui.ComponentWrapper{
		ID: "root",
		Component: a2ui.Component{
			Column: &a2ui.Column{
				Children: a2ui.Children{ExplicitList: []string{"header", "project_form", "cluster_select", "mode_list"}},
			},
		},
	})

	// Header
	components = append(components, a2ui.MakeText("header", "Failure Mode Simulator", "h1"))

	// Project Form
	components = append(components, a2ui.MakeText("project_form", "Project: "+project, "h3"))

	// Cluster Select
	targetCluster := ""
	if len(clusterNames) > 0 {
		targetCluster = clusterNames[0]
	}
	components = append(components, a2ui.MakeText("cluster_select", "Target Cluster: "+targetCluster, "h3"))

	// Modes
	var modeIDs []string
	for _, mode := range k8sModes {
		rowID := "mode_row_" + mode.ID
		nameID := "mode_name_" + mode.ID
		descID := "mode_desc_" + mode.ID
		applyBtnID := "btn_apply_" + mode.ID
		applyLabelID := "lbl_apply_" + mode.ID
		revertBtnID := "btn_revert_" + mode.ID
		revertLabelID := "lbl_revert_" + mode.ID

		modeIDs = append(modeIDs, rowID)

		// Row
		components = append(components, a2ui.ComponentWrapper{
			ID: rowID,
			Component: a2ui.Component{
				Column: &a2ui.Column{
					Children: a2ui.Children{ExplicitList: []string{nameID, descID, "actions_" + mode.ID}},
				},
			},
		})

		// Name & Desc
		components = append(components, a2ui.MakeText(nameID, mode.Name, "h3"))
		components = append(components, a2ui.MakeText(descID, mode.Description, "mono"))

		// Actions Row
		components = append(components, a2ui.ComponentWrapper{
			ID: "actions_" + mode.ID,
			Component: a2ui.Component{
				Row: &a2ui.Row{
					Children: a2ui.Children{ExplicitList: []string{applyBtnID, revertBtnID}},
				},
			},
		})

		// Buttons
		applyComps := a2ui.MakeButton(applyBtnID, applyLabelID, "Apply", "apply_mode", map[string]string{
			"mode":    mode.ID,
			"project": project,
			"cluster": targetCluster,
			"action":  "apply",
		})
		components = append(components, applyComps...)

		revertComps := a2ui.MakeButton(revertBtnID, revertLabelID, "Revert", "apply_mode", map[string]string{
			"mode":    mode.ID,
			"project": project,
			"cluster": targetCluster,
			"action":  "revert",
		})
		components = append(components, revertComps...)
	}

	components = append(components, a2ui.ComponentWrapper{
		ID: "mode_list",
		Component: a2ui.Component{
			Column: &a2ui.Column{
				Children: a2ui.Children{ExplicitList: modeIDs},
			},
		},
	})

	s.Broadcast(a2ui.Message{
		SurfaceUpdate: &a2ui.SurfaceUpdate{
			SurfaceID: "main",
			Components: components,
		},
	})

	s.Broadcast(a2ui.Message{
		BeginRendering: &a2ui.BeginRendering{Root: "root"},
	})
}

func handleUserAction(s *a2ui.Server, rootDir string, action a2ui.UserAction) {
	ctx := context.Background()
	if action.Name == "apply_mode" {
		mode := action.Context["mode"]
		project := action.Context["project"]
		cluster := action.Context["cluster"]
		act := action.Context["action"] // apply or revert

		// Find location
		location := "us-central1"
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

		statusID := "header" // Reuse header for status
		s.Broadcast(a2ui.Message{
			SurfaceUpdate: &a2ui.SurfaceUpdate{
				Components: []a2ui.ComponentWrapper{
					a2ui.MakeText(statusID, fmt.Sprintf("Processing %s %s...", act, mode), "h1"),
				},
			},
		})

		agentService, err := agent.NewAgent(ctx, project, "us-central1", "gemini-2.0-flash-001", rootDir)
		if err != nil {
			log.Printf("Agent init fail: %v", err)
			return
		}
		defer agentService.Close()

		var prompt string
		if act == "apply" {
			prompt = fmt.Sprintf("Please apply the '%s' failure mode to cluster '%s' in project '%s', location '%s'.", mode, cluster, project, location)
		} else {
			prompt = fmt.Sprintf("Please revert the '%s' failure mode on cluster '%s' in project '%s', location '%s'.", mode, cluster, project, location)
		}

		resp, err := agentService.Run(ctx, prompt)
		resText := resp
		if err != nil {
			resText = fmt.Sprintf("Error: %v", err)
		}

		// Update Status
		s.Broadcast(a2ui.Message{
			SurfaceUpdate: &a2ui.SurfaceUpdate{
				Components: []a2ui.ComponentWrapper{
					a2ui.MakeText(statusID, "Agent: "+resText, "h1"),
				},
			},
		})
	}
}
