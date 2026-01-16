package main

import (
	"context"
	"log"
	// "net/http"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	// "google.golang.org/adk/tool"
	// "google.golang.org/adk/tool/geminitool"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	// Set default environment variables for local development if not set
	if os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
		log.Println("GOOGLE_CLOUD_PROJECT not set, defaulting to 'mslarkin-ext'")
		os.Setenv("GOOGLE_CLOUD_PROJECT", "mslarkin-ext")
	}
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")

	if os.Getenv("GOOGLE_CLOUD_LOCATION") == "" {
		log.Println("GOOGLE_CLOUD_LOCATION not set, defaulting to 'us-west1'")
		os.Setenv("GOOGLE_CLOUD_LOCATION", "us-west1")
	}
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")

	if os.Getenv("GOOGLE_GENAI_USE_VERTEXAI") == "" {
		log.Println("GOOGLE_GENAI_USE_VERTEXAI not set, defaulting to 'true' (for ADC)")
		os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "true")
	}

	modelName := "gemini-2.5-pro"
	log.Printf("Initializing agent with project: %s, location: %s, model: %s", projectID, location, modelName)

	// Create Gemini model
	// Backend is now determined by GOOGLE_GENAI_USE_VERTEXAI environment variable
	model, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		Project:  projectID,
		Location: location,
	})
	if err != nil {
		log.Fatalf("Failed to create Gemini model: %v", err)
	}

	// Create LLM agent
	myAgent, err := llmagent.New(llmagent.Config{
		Name:        "GKEAssistant",
		Description: "A helpful AI assistant for the GKE Demo.",
		Model:       model,
		Instruction: "You are a helpful AI assistant for the GKE Demo. You can answer questions about GKE and the demo environment.",
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create a single agent loader
	loader := agent.NewSingleLoader(myAgent)

	// Configure the launcher/server
	// Initialize all required services with their InMemory implementations to avoid panics.
	cfg := &launcher.Config{
		AgentLoader: loader,
	}

	l := full.NewLauncher()

	if err = l.Execute(ctx, cfg, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
}

	// Create the HTTP handler
	// handler := adkrest.NewHandler(cfg)

	// port := os.Getenv("PORT")
	// if port == "" {
	// 	port = "8080"
	// }

	// log.Printf("Starting ADK agent server on :%s...", port)
	// if err := http.ListenAndServe(":"+port, handler); err != nil {
	// 	log.Fatal(err)
	// }
}
