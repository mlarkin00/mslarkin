package main

import (
	"context"

	"log"
	"os"

	"k8s-status-backend/mcpclient"
)

func main() {
	log.Println("Starting integration test for mcpclient logs...")
	ctx := context.Background()

	// 1. Initialize Client
	client, err := mcpclient.NewMCPClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create MCP Client: %v", err)
	}
	defer client.Close()
	log.Println("MCP Client initialized.")

	// 2. Test ListClusters (OneMCP)
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = "mslarkin-demo"
	}
	log.Printf("Testing ListClusters for project: %s...", projectID)

	clusters, err := client.ListClusters(ctx, projectID)
	if err != nil {
		log.Printf("ListClusters failed: %v", err)
	} else {
		log.Printf("Successfully retrieved %d clusters.", len(clusters))
	}
}
