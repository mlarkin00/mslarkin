package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
    "log"
)

// Mock components
var BasePath string

func ResolveURL(r *http.Request, path string) string {
	prefix := r.Header.Get("X-Forwarded-Prefix")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if prefix == "" {
		if BasePath != "" {
			cleanBase := strings.TrimRight(BasePath, "/")
			return cleanBase + path
		}
		return path
	}
	return strings.TrimRight(prefix, "/") + path
}

func main() {
    BasePath = "/k8s-status"

    internalPath := "/k8s-status-app"

    apiMux := http.NewServeMux()
    apiMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.NotFound(w, r)
            return
        }
        fmt.Fprint(w, "Landing Page")
    })
    apiMux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
         if r.URL.Query().Get("project") == "" {
             target := ResolveURL(r, "/")
             http.Redirect(w, r, target, http.StatusFound)
             return
         }
         fmt.Fprint(w, "Dashboard")
    })

    cleaner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cleaned := path.Clean(r.URL.Path)
        if cleaned == "." {
            cleaned = "/"
        }
        r.URL.Path = cleaned
        apiMux.ServeHTTP(w, r)
    })

    appHandler := http.StripPrefix(internalPath, cleaner)

    topMux := http.NewServeMux()
    topMux.Handle(internalPath+"/", appHandler)
    topMux.HandleFunc(internalPath, func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Trapped missing slash: %s", r.URL.Path)
        r.URL.Path = internalPath + "/"
        appHandler.ServeHTTP(w, r)
    })

    server := httptest.NewServer(topMux)
    defer server.Close()

    // Test Cases
    cases := []struct {
        Name string
        Path string
        Headers map[string]string
        WantCode int
        WantBody string
        WantLoc string
    }{
        {
            Name: "Root Path with Slash (Standard LB behavior)",
            Path: internalPath + "/",
            Headers: map[string]string{"X-Forwarded-Prefix": "/k8s-status"},
            WantCode: 200,
            WantBody: "Landing Page",
        },
        {
            Name: "Root Path Missing Slash (Trap)",
            Path: internalPath,
            Headers: map[string]string{"X-Forwarded-Prefix": "/k8s-status"},
            WantCode: 200,
            WantBody: "Landing Page",
        },
        {
            Name: "Dashboard Redirect (No Project)",
            Path: internalPath + "/dashboard",
            Headers: map[string]string{"X-Forwarded-Prefix": "/k8s-status"},
            WantCode: 302,
            WantLoc: "/k8s-status/",
        },
        {
            Name: "Dashboard OK",
            Path: internalPath + "/dashboard?project=foo",
            Headers: map[string]string{"X-Forwarded-Prefix": "/k8s-status"},
            WantCode: 200,
            WantBody: "Dashboard",
        },
    }

    for _, tc := range cases {
        req, _ := http.NewRequest("GET", server.URL + tc.Path, nil)
        for k, v := range tc.Headers {
            req.Header.Set(k, v)
        }

        // Handle client not following redirects automatically to check Location
        transport := &http.Transport{}
        client := &http.Client{
            Transport: transport,
            CheckRedirect: func(req *http.Request, via []*http.Request) error {
                return http.ErrUseLastResponse
            },
        }

        resp, err := client.Do(req)
        if err != nil {
            log.Fatalf("Failed %s: %v", tc.Name, err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != tc.WantCode {
            fmt.Printf("[FAIL] %s: Want Code %d, Got %d\n", tc.Name, tc.WantCode, resp.StatusCode)
            continue
        }
        if tc.WantLoc != "" {
            loc := resp.Header.Get("Location")
            if loc != tc.WantLoc {
                 fmt.Printf("[FAIL] %s: Want Loc %s, Got %s\n", tc.Name, tc.WantLoc, loc)
                 continue
            }
        }
        fmt.Printf("[PASS] %s\n", tc.Name)
    }
}
