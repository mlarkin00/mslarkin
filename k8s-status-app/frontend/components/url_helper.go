package components

import (
	"fmt"
	"net/http"
	"strings"
)

// ResolveURL builds a path relative to the public prefix (e.g. adding "/k8s-status" back in)
// It relies on X-Forwarded-Prefix being set by the Load Balancer (or Backend Service).
func ResolveURL(r *http.Request, path string) string {
	// 1. Get the prefix from the Load Balancer
	prefix := r.Header.Get("X-Forwarded-Prefix")

	// Clean up: Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 2. If there is no prefix, fall back to BasePath (if set) or return path
	if prefix == "" {
		if BasePath != "" {
			// Clean BasePath to ensure no double slashes if we combine
			cleanBase := strings.TrimRight(BasePath, "/")
			return cleanBase + path
		}
		return path
	}

	// 3. Combine them. Ensure we don't create double slashes
	return strings.TrimRight(prefix, "/") + path
}

// ResolveAbsoluteURL builds the full absolute URL including protocol and domain.
func ResolveAbsoluteURL(r *http.Request, path string) string {
	// 1. Determine Protocol (trust the LB first)
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	// 2. Determine Host (trust the LB first)
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	// 3. Get the path with the prefix corrected
	resolvedPath := ResolveURL(r, path)

	return fmt.Sprintf("%s://%s%s", proto, host, resolvedPath)
}
