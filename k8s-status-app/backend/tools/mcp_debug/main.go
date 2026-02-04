package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"k8s-status-backend/auth"
)

const (
	targetURL = "https://mcp.ai.mslarkin.com/sse"
	audience  = "79309377625-i17s6rtmlmi6t3dg61b69nvfsvss8cdp.apps.googleusercontent.com"
	outputFile = "mcp_test_output.txt"
)

func main() {
	ctx := context.Background()

	// 1. Get ID Token
	fmt.Printf("Getting ID Token for audience: %s\n", audience)
	token, err := auth.GetIDToken(ctx, audience)
	if err != nil {
		fmt.Printf("Error getting ID token: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Got ID Token.")

	// 2. Prepare Request
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")

	// 3. Dump Request
	reqDump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		fmt.Printf("Error dumping request: %v\n", err)
		os.Exit(1)
	}

	// 4. Send Request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// 5. Dump Response
	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		fmt.Printf("Error dumping response: %v\n", err)
		os.Exit(1)
	}

	// 6. Write to File
	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Fprintln(f, "=== REQUEST ===")
	// Redact token for safety in output file (optional, but good practice)
	// Actually, user asked to show request sent, I'll keep it raw or maybe truncate token.
	// For "show request sent", I'll show it but maybe not the full token if it's huge.
	// httputil dumper shows it.
	f.Write(reqDump)
	fmt.Fprintln(f, "\n\n=== RESPONSE ===")
	f.Write(respDump)

	fmt.Printf("Successfully wrote request/response to %s\n", outputFile)
	fmt.Printf("Response Status: %s\n", resp.Status)
}
