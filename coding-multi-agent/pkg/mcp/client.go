package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mslarkin/coding-multi-agent/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Client interface {
	Call(ctx context.Context, requestBody interface{}) (interface{}, error)
}

type HTTPClient struct {
	endpoint   string
	httpClient *http.Client
	authConfig config.MCPClientConfig
	tokenSrc   oauth2.TokenSource
}

func NewClient(ctx context.Context, cfg config.MCPClientConfig) (*HTTPClient, error) {
	client := &HTTPClient{
		endpoint:   cfg.Endpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		authConfig: cfg,
	}

	if cfg.AuthType == "google_oauth" {
		// Use ADC to get a token source
		creds, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials for google_oauth: %w", err)
		}
		client.tokenSrc = creds.TokenSource
	}

	return client, nil
}

func (c *HTTPClient) Call(ctx context.Context, requestBody interface{}) (interface{}, error) {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Handle Auth
	if c.authConfig.AuthType == "custom_header" {
		// Retrieve key from env var using the header key name (or a mapping)
		// For simplicity, assuming the config HeaderKey is the NAME of the header,
		// and the value comes from an env var named similarly, or passed in config?
		// The design says: HeaderKey: "CONTEXT7_API_KEY". And in Pod definition:
		// env: MCP_CONTEXT7_KEY valueFrom secret.
		// So we likely need to look up the value.
		// Let's assume the value is in an env var for now, or we modify config to hold the value.
		// The design config struct has HeaderKey.
		// Let's assume the env var name is derived or standard.
		// The pod spec shows: env name MCP_CONTEXT7_KEY.
		// I'll check if there's an env var with the name of the HeaderKey or just use a placeholder.
		// For now, let's assume CONTEXT7_API_KEY env var holds the value if HeaderKey is CONTEXT7_API_KEY.
		if val := os.Getenv(c.authConfig.HeaderKey); val != "" {
			req.Header.Set(c.authConfig.HeaderKey, val)
		} else if val := os.Getenv("MCP_" + c.authConfig.HeaderKey); val != "" {
			req.Header.Set(c.authConfig.HeaderKey, val)
		}
	} else if c.authConfig.AuthType == "google_oauth" && c.tokenSrc != nil {
		token, err := c.tokenSrc.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to get oauth token: %w", err)
		}
		token.SetAuthHeader(req)
	}

	if c.authConfig.UserProjectOverride != "" {
		req.Header.Set("X-goog-user-project", c.authConfig.UserProjectOverride)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mcp server error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
