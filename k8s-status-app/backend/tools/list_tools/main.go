package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
    "time"

	"k8s-status-backend/mcpclient"
)

func main() {
	// Set project ID for testing
	os.Setenv("GOOGLE_CLOUD_PROJECT", "mslarkin-ext")

	ctx := context.Background()
	log.Println("Initializing MCP Client...")
	client, err := mcpclient.NewMCPClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create MCP Client: %v", err)
	}
	defer client.Close()

    // Wait a bit for connection
    time.Sleep(2 * time.Second)

    log.Println("Listing Tools from OneMCP...")
    if client.OneMCPSession == nil {
        log.Fatal("OneMCP session is nil")
    }

    toolsRes, err := client.OneMCPSession.ListTools(ctx, nil)
    if err != nil {
        log.Fatalf("Failed to list tools: %v", err)
    }

    for _, tool := range toolsRes.Tools {
        if tool.Name == "kube_get" {
            fmt.Printf("Tool: %s\nDescription: %s\n", tool.Name, tool.Description)
            argsJSON, _ := json.MarshalIndent(tool.InputSchema, "  ", "  ")
            fmt.Printf("Input Schema: %s\n", string(argsJSON))
        }
    }
}
