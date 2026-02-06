package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/web"
	"google.golang.org/adk/cmd/launcher/web/webui"
)

func main() {
	port := flag.Int("port", 8000, "Port to listen on")
	flag.Parse()

	// Initialize config
	config := &launcher.Config{}

	webuiLauncher := webui.NewLauncher()

	// Parse webui args (pass all remaining args)
	// webui expects its specific flags
	// We skip the first arg if it is "webui" to fail gracefully or just ignore it
	args := flag.Args()
	if len(args) > 0 && args[0] == "webui" {
		args = args[1:]
	}

	if _, err := webuiLauncher.Parse(args); err != nil {
		log.Fatalf("Failed to parse webui args: %v", err)
	}

	// Build router
	router := web.BuildBaseRouter()

	// Setup webui routes
	if err := webuiLauncher.SetupSubrouters(router, config); err != nil {
		log.Fatalf("Failed to setup subrouters: %v", err)
	}

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("Starting ADK Web UI on %s...", addr)

	// Call UserMessage hook if available (optional)
	webuiLauncher.UserMessage(fmt.Sprintf("http://localhost:%d", *port), log.Println)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
