package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"k8s-status-backend/mcpclient"
)

func main() {
	ctx := context.Background()
	// Set env if needed
	os.Setenv("GOOGLE_CLOUD_PROJECT", "mslarkin-ext")

	client, err := mcpclient.NewMCPClient(ctx)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
        // If it fails due to auth or connection, we might exit, but let's see.
        return
	}
	defer client.Close()

	projects := []string{"mslarkin-ext", "mslarkin-demo"}
	for _, p := range projects {
		fmt.Printf("Listing clusters for project: %s\n", p)
		clusters, err := client.ListClusters(ctx, p)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}
        fmt.Printf("Found %d clusters\n", len(clusters))
		for _, c := range clusters {
			jsonB, _ := json.Marshal(c)
			fmt.Println(string(jsonB))
		}
	}
}
