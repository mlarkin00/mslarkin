// Package auth provides authentication utilities for the Backend.
package auth

import (
	"context"
	"fmt"

	"os/exec"
	"strings"

	"cloud.google.com/go/compute/metadata"
	"google.golang.org/api/idtoken"
	"golang.org/x/oauth2/google"
)

// GetIDToken fetches an OIDC ID token for the given audience.
// It tries to use the metadata server first (when running on GCE/GKE),
// falling back to ADC (Application Default Credentials) if running locally.
//
// Parameters:
//   - ctx: Context for the request.
//   - audience: The audience (URL) the token is intended for.
//
// Returns:
//   - string: The ID token.
//   - error: Any error encountered.
func GetIDToken(ctx context.Context, audience string) (string, error) {
	if metadata.OnGCE() {
		// Use metadata server
		token, err := metadata.Get("instance/service-accounts/default/identity?audience=" + audience + "&format=full")
		if err != nil {
			return "", fmt.Errorf("failed to get token from metadata: %w", err)
		}
		return token, nil
	}

	// Local development fallback using ADC
	ts, err := idtoken.NewTokenSource(ctx, audience)
	if err == nil {
		token, err := ts.Token()
		if err == nil {
			return token.AccessToken, nil
		}
	}

	// Fallback to gcloud if available (common for User Credentials)
	// This is useful when running locally with 'gcloud auth login' but without SA key.
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "print-identity-token", "--audiences="+audience)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("idtoken library failed and gcloud fallback failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil

}

// GetAccessToken fetches an OAuth2 access token with the cloud-platform scope.
func GetAccessToken(ctx context.Context) (string, error) {
	// Use google.FindDefaultCredentials which handles ADC and GKE Metadata automatically.
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to find default credentials: %w", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	return token.AccessToken, nil
}
