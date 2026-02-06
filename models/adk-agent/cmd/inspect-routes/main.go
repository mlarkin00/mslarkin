package main

import (
	"fmt"
	"time"
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/server/adkrest"
)

func main() {
	config := &launcher.Config{}
	handler := adkrest.NewHandler(config, 30*time.Second)

	r, ok := handler.(*mux.Router)
	if !ok {
		fmt.Printf("Not a mux.Router: %T\n", handler)
		return
	}

	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			fmt.Println("ROUTE:", pathTemplate)
		}
		queries, err := route.GetQueriesTemplates()
		if err == nil {
			fmt.Println("  Queries:", queries)
		}
		methods, err := route.GetMethods()
		if err == nil {
			fmt.Println("  Methods:", methods)
		}
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

    // Also print some standard paths if they match
    printMatch(r, "/api/agents")
    printMatch(r, "/agents")
    printMatch(r, "/models")
}

func printMatch(r *mux.Router, path string) {
    var match mux.RouteMatch
    req, _ := http.NewRequest("GET", path, nil)
    if r.Match(req, &match) {
        fmt.Printf("MATCH %s -> %v\n", path, match.Route.GetName())
    } else {
        fmt.Printf("NO MATCH %s\n", path)
    }
}
