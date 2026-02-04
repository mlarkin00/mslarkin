package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"k8s-status-backend/mcpclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: debug_onemcp <project> <location> <cluster>")
		os.Exit(1)
	}
	project := os.Args[1]
	location := os.Args[2]
	cluster := os.Args[3]

	ctx := context.Background()
	client, err := mcpclient.NewMCPClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	if client.OneMCPSession == nil {
		log.Fatal("OneMCP session is nil")
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)
	namespace := "default"

	// Test variations
	types := []string{"deployments", "Deployment", "services", "Service", "pods", "Pod"}

	for _, t := range types {
		fmt.Printf("--- Testing resourceType: %s ---\n", t)
		args := map[string]interface{}{
			"parent":       parent,
			"resourceType": t,
			"namespace":    namespace,
		}

		result, err := client.OneMCPSession.CallTool(ctx, &mcp.CallToolParams{
			Name:      "kube_get",
			Arguments: args,
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if len(result.Content) > 0 {
            // Dump content
            for i, c := range result.Content {
                if text, ok := c.(*mcp.TextContent); ok {
                    fmt.Printf("Content[%d] (Text): %s\n", i, text.Text)
                } else {
                    b, _ := json.Marshal(c)
                    fmt.Printf("Content[%d] (Other): %s\n", i, string(b))
                }
            }
		} else {
			fmt.Println("Result Content is empty")
		}

        if result.StructuredContent != nil {
             b, _ := json.Marshal(result.StructuredContent)
             fmt.Printf("StructuredContent: %s\n", string(b))
        }
        fmt.Println()
	}
}
