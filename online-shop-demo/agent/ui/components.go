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

func Dashboard(projects []string, currentProject string, clusters []string, failureModes []string) g.Node {
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
				Ul(Class("space-y-2"),
					g.Map(failureModes, func(mode string) g.Node {
						return Li(Class("flex items-center justify-between p-3 bg-gray-50 rounded border"),
							Span(Class("font-medium"), g.Text(mode)),
							Div(
								Button(
									Class("bg-red-500 text-white px-3 py-1 rounded text-sm mr-2 hover:bg-red-600"),
									g.Attr("hx-post", fmt.Sprintf("/apply?mode=%s&action=apply", mode)),
									g.Attr("hx-target", "#status-"+mode),
									g.Text("Apply"),
								),
								Button(
									Class("bg-green-500 text-white px-3 py-1 rounded text-sm hover:bg-green-600"),
									g.Attr("hx-post", fmt.Sprintf("/apply?mode=%s&action=revert", mode)),
									g.Attr("hx-target", "#status-"+mode),
									g.Text("Revert"),
								),
							),
							Div(ID("status-"+mode), Class("ml-4 text-sm text-gray-500")),
						)
					}),
				),
			),
		),
	)
}
