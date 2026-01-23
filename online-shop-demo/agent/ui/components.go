package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func Layout(title string, body g.Node) g.Node {
	return HTML(
		Head(
			TitleEl(g.Text(title)),
			Script(Src("https://cdn.tailwindcss.com")),
			Script(Src("https://unpkg.com/htmx.org@1.9.10")),
		),
		Body(Class("bg-gray-100 p-8"),
			Div(Class("max-w-4xl mx-auto bg-white shadow rounded-lg p-6"),
				H1(Class("text-2xl font-bold mb-4"), g.Text(title)),
				body,
			),
		),
	)
}

// We need to import the k8s package to use the struct, OR just define an interface or struct here.
// To avoid cyclic deps if ui imports k8s (and main imports both), we should define the struct here or pass a generic struct.
// For simplicity in this demo, let's redefine the struct or interface, or just pass a struct with Name/Description.
// Since Go doesn't like cyclic imports, and main -> ui, main -> k8s. ui should NOT import k8s.
// We'll change the signature to accept a slice of structs defined in UI or just interface{}.
// Let's pass a struct defined in UI package for loose coupling, or just standard types.
// Actually, let's just use a local struct alias or interface for now to keep it simple.

type FailureMode struct {
	Name        string
	Description string
}

func Dashboard(projects []string, currentProject string, clusters []string, failureModes []FailureMode) g.Node {
	return Div(
		FormEl(Method("get"), Action("/"), Class("mb-6 flex gap-4 items-end"),
			Div(
				Label(For("project"), Class("block text-sm font-medium text-gray-700"), g.Text("Project ID")),
				Input(Type("text"), Name("project"), ID("project"), Value(currentProject), Class("mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border")),
			),
			Button(Type("submit"), Class("bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"), g.Text("Load Clusters")),
		),
		g.If(len(clusters) > 0,
			Div(
				H2(Class("text-xl font-semibold mb-2"), g.Text("Target Cluster")),
				Select(Name("cluster"), ID("cluster"), Class("block w-full rounded-md border-gray-300 shadow-sm p-2 border mb-6"),
					g.Map(clusters, func(c string) g.Node {
						return Option(Value(c), g.Text(c))
					}),
				),
				H2(Class("text-xl font-semibold mb-2"), g.Text("Failure Modes")),
				Ul(Class("space-y-4"),
					g.Map(failureModes, func(mode FailureMode) g.Node {
						return Li(Class("p-4 bg-gray-50 rounded border"),
							Div(Class("flex items-center justify-between mb-2"),
								Span(Class("font-bold text-lg"), g.Text(mode.Name)),
								Div(
									Button(
										Class("bg-red-500 text-white px-3 py-1 rounded text-sm mr-2 hover:bg-red-600"),
										g.Attr("hx-post", fmt.Sprintf("/apply?mode=%s&action=apply", mode.Name)),
										g.Attr("hx-target", "#status-"+mode.Name),
										g.Attr("hx-include", "#project, #cluster"),
										g.Text("Apply"),
									),
									Button(
										Class("bg-green-500 text-white px-3 py-1 rounded text-sm hover:bg-green-600"),
										g.Attr("hx-post", fmt.Sprintf("/apply?mode=%s&action=revert", mode.Name)),
										g.Attr("hx-target", "#status-"+mode.Name),
										g.Attr("hx-include", "#project, #cluster"),
										g.Text("Revert"),
									),
								),
							),
							P(Class("text-gray-600 text-sm whitespace-pre-wrap"), g.Text(mode.Description)),
							Div(ID("status-"+mode.Name), Class("mt-2 text-sm text-gray-500 font-mono")),
						)
					}),
				),
			),
		),
	)
}
