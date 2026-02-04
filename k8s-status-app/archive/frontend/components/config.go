package components

import "strings"

// BasePath is the root path for the application (e.g., "/k8s-status").
// It should be set at startup.
var BasePath string

// AppLink prepends the BasePath to the given path ensuring no double slashes.
func AppLink(path string) string {
	if BasePath == "" {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return BasePath + path
	}
	return BasePath + "/" + path
}
