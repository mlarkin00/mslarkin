package views

import (
	"net/http"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
    "k8s-status-frontend/components"
)

// Landing renders the home page with a form to select a project.
func Landing(r *http.Request) Node {
    return components.Layout(r, "GKE Status - Home",
        Div(Class("hero min-h-screen bg-base-200"),
            Div(Class("hero-content text-center"),
                Div(Class("max-w-md"),
                    H1(Class("text-5xl font-bold"), Text("GKE Status Dashboard")),
                    P(Class("py-6"), Text("Monitor your GKE clusters and workloads via MCP.")),
                    FormEl(Action(components.ResolveURL(r, "/dashboard")), Method("GET"),
                        Input(Type("text"), Name("project"), Value("mslarkin-ext"), Class("input input-bordered w-full max-w-xs mb-4"), Placeholder("Project ID")),
                        Br(),
                        Button(Type("submit"), Class("btn btn-primary"), Text("View Dashboard")),
                    ),
                ),
            ),
        ),
    )
}
