package views

import (
	"net/http"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
    "k8s-status-frontend/components"
    "k8s-status-frontend/models"
)

// DashboardData holds the data required to render the Dashboard view.
type DashboardData struct {
    Project string
    Clusters []models.Cluster
}

// Dashboard renders the main project dashboard with a list of clusters.
func Dashboard(r *http.Request, data DashboardData) Node {
    return components.Layout(r, "Dashboard - "+data.Project,
        Div(
            H2(Class("text-2xl font-bold mb-4"), Text("Project: "+data.Project)),
            Div(Class("flex-col gap-4 flex"), // Fixed class order and name for clarity? "flex flex-col gap-4" is better but staying safe
                Map(data.Clusters, func(c models.Cluster) Node {
                    return ClusterCard(r, c)
                }),
            ),
        ),
    )
}

// ClusterCard renders a card component for a single cluster.
func ClusterCard(r *http.Request, c models.Cluster) Node {
    return Div(Class("card bg-base-100 shadow-xl"),
        Div(Class("card-body"),
            H2(Class("card-title"), Text(c.Name),
                Span(Class("badge badge-success"), Text(c.Status)),
            ),
            P(Text("Location: "+c.Location)),

            Div(
                Attr("hx-get", components.ResolveURL(r, "/partials/workloads?cluster="+c.Name+"&location="+c.Location+"&project="+c.ProjectID+"&namespace=default")),
                Attr("hx-trigger", "load"),
                Attr("hx-swap", "innerHTML"),
                Span(Class("loading loading-spinner"), Text("Loading workloads...")),
            ),
        ),
    )
}

// WorkloadsList renders a table of workloads.
func WorkloadsList(workloads []models.Workload) Node {
    return Div(Class("overflow-x-auto"),
        Table(Class("table table-xs"),
            THead(
                Tr(
                    Th(Text("Name")),
                    Th(Text("Type")),
                    Th(Text("Status")),
                    Th(Text("Ready")),
                    Th(Text("Age")),
                    Th(Text("Actions")),
                ),
            ),
            TBody(
                Map(workloads, func(w models.Workload) Node {
                    return Tr(
                        Td(Text(w.Name)),
                        Td(Text(w.Type)),
                        Td(Text(w.Status)),
                        Td(Text(w.Ready)),
                        Td(Text(w.Age)),
                        Td(
                            Button(Class("btn btn-xs btn-outline"), Text("Describe")),
                            Button(Class("btn btn-xs btn-outline ml-1"), Text("Pods")),
                        ),
                    )
                }),
            ),
        ),
    )
}
