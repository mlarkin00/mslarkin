package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mslarkin/online-shop-demo/agent/pkg/a2ui"
	"github.com/mslarkin/online-shop-demo/agent/pkg/agent"
	"github.com/mslarkin/online-shop-demo/agent/pkg/gcp"
	"github.com/mslarkin/online-shop-demo/agent/pkg/k8s"
)

type AppState struct {
	mu             sync.RWMutex
	CurrentProject string
	CurrentCluster string
	Clusters       []gcp.Cluster
}

var appState = &AppState{
	CurrentProject: "mslarkin-ext",
}

func main() {
	rootDir := os.Getenv("APP_ROOT")
	if rootDir == "" {
		rootDir, _ = filepath.Abs("..")
	}

	// Initialize A2UI Server
	a2uiServer := a2ui.NewServer()

	// Initial Render
	refreshClusters(context.Background())
	renderUI(a2uiServer, rootDir)

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

func refreshClusters(ctx context.Context) {
	appState.mu.Lock()
	defer appState.mu.Unlock()

	clusters, err := gcp.ListClusters(ctx, appState.CurrentProject)
	if err == nil {
		appState.Clusters = clusters
		// Set default if not set or invalid
		found := false
		for _, c := range clusters {
			if c.Name == appState.CurrentCluster {
				found = true
				break
			}
		}
		if !found && len(clusters) > 0 {
			appState.CurrentCluster = clusters[0].Name
		}
	} else {
		log.Printf("Error listing clusters: %v", err)
	}
}

func renderUI(s *a2ui.Server, rootDir string) {
	appState.mu.RLock()
	project := appState.CurrentProject
	targetCluster := appState.CurrentCluster
	clusters := appState.Clusters
	appState.mu.RUnlock()

	// Build components
	k8sModes, err := k8s.GetFailureModes(rootDir)
	if err != nil {
		log.Printf("Error getting failure modes: %v", err)
	}

	// Construct A2UI Components
	var components []a2ui.ComponentWrapper

	// Root Column
	components = append(components, a2ui.ComponentWrapper{
		ID: "root",
		Component: a2ui.Component{
			Column: &a2ui.Column{
				Children: a2ui.Children{ExplicitList: []string{"header", "project_form", "cluster_select_row", "mode_list"}},
			},
		},
	})

	// Header
	components = append(components, a2ui.MakeText("header", "Failure Mode Simulator (A2UI)", "h1"))

	// Project Form
	components = append(components, a2ui.MakeText("project_form", "Project: "+project, "h3"))

	// Cluster Select
	var clusterOptions []a2ui.Option
	for _, c := range clusters {
		clusterOptions = append(clusterOptions, a2ui.Option{ID: c.Name, Label: c.Name})
	}

	// Wrap select in a row or just use the ID
	// Using a Row to keep it clean if we add a refresh button later
	components = append(components, a2ui.ComponentWrapper{
		ID: "cluster_select_row",
		Component: a2ui.Component{
			Column: &a2ui.Column{
				Children: a2ui.Children{ExplicitList: []string{"cluster_label", "cluster_dropdown"}},
			},
		},
	})

	components = append(components, a2ui.MakeText("cluster_label", "Target Cluster:", "h3"))

	selectComps := a2ui.MakeSelect("cluster_dropdown", "Target Cluster", clusterOptions, targetCluster, "select_cluster", map[string]string{
		"project": project,
	})
	components = append(components, selectComps...)


	// Find location of target cluster to configure credentials
	location := "us-central1"
	for _, c := range clusters {
		if c.Name == targetCluster {
			location = c.Location
			break
		}
	}

	// Configure credentials once for this render pass (if targetCluster is set)
	if targetCluster != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := k8s.ConfigureCredentials(ctx, project, location, targetCluster); err != nil {
			log.Printf("Failed to configure credentials for %s: %v", targetCluster, err)
		}
		cancel()
	}

	// Modes
	var modeIDs []string
	for _, mode := range k8sModes {
		isActive, _ := k8s.IsFailureModeActive(context.Background(), mode.ID)

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
		statusText := ""
		if isActive {
			statusText = " (Active)"
		}
		components = append(components, a2ui.MakeText(nameID, mode.Name+statusText, "h3"))
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

		// Buttons Logic
		applyVariant := "success"
		applyLabel := "Apply"
		applyAction := "apply"

		revertVariant := "neutral"
		revertLabel := "Revert"
		revertAction := "revert"

		if isActive {
			applyVariant = "neutral"
			applyLabel = "Applied"
			// revertVariant logic: if active, revert is danger (red)
			revertVariant = "danger"
		}

		// Apply Button
		applyComps := a2ui.MakeButton(applyBtnID, applyLabelID, applyLabel, "apply_mode", map[string]string{
			"mode":    mode.ID,
			"project": project,
			"cluster": targetCluster,
			"action":  applyAction,
		})
		// Patch variant (MakeButton returns [Text, Button])
		for i := range applyComps {
			if applyComps[i].Component.Button != nil {
				applyComps[i].Component.Button.Variant = applyVariant
				break
			}
		}
		components = append(components, applyComps...)

		// Revert Button
		revertComps := a2ui.MakeButton(revertBtnID, revertLabelID, revertLabel, "apply_mode", map[string]string{
			"mode":    mode.ID,
			"project": project,
			"cluster": targetCluster,
			"action":  revertAction,
		})
		// Patch variant
		for i := range revertComps {
			if revertComps[i].Component.Button != nil {
				revertComps[i].Component.Button.Variant = revertVariant
				break
			}
		}
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

	if action.Name == "select_cluster" {
		val, ok := action.Context["selected_value"]
		if ok {
			appState.mu.Lock()
			appState.CurrentCluster = val
			appState.mu.Unlock()
			renderUI(s, rootDir)
		}
		return
	}

	if action.Name == "apply_mode" {
		mode := action.Context["mode"]
		project := action.Context["project"]
		// Trust the button context, OR use global state.
		// Button context was burned in at render time.
		// If we re-render on select, button context is fresh.
		cluster := action.Context["cluster"]
		act := action.Context["action"] // apply or revert

		// Find location using cached clusters
		appState.mu.RLock()
		clusters := appState.Clusters
		appState.mu.RUnlock()

		location := "us-central1"
		for _, c := range clusters {
			if c.Name == cluster {
				location = c.Location
				break
			}
		}

		statusID := "header" // Reuse header for status
		s.Broadcast(a2ui.Message{
			SurfaceUpdate: &a2ui.SurfaceUpdate{
				Components: []a2ui.ComponentWrapper{
					a2ui.MakeText(statusID, fmt.Sprintf("Processing %s %s on %s...", act, mode, cluster), "h1"),
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
