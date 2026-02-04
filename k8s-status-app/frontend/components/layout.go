package components

import (
	"net/http"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func Layout(r *http.Request, title string, body Node) Node {
	return HTML(
		Head(
			TitleEl(Text(title)),
			Script(Attr("src", ResolveURL(r, "/static/js/htmx.min.js"))),
			Link(Attr("rel", "stylesheet"), Attr("href", ResolveURL(r, "/static/css/daisyui.min.css"))),
			Script(Attr("src", ResolveURL(r, "/static/js/tailwindcss.js"))),
			Script(Raw(`
                window.APP_CONFIG = {
                    basePath: "`+BasePath+`"
                };
            `)),
            Script(Type("module"), Attr("src", ResolveURL(r, "/static/js/chat-widget.js"))),
		),
		Body(
			Class("bg-gray-100 min-h-screen"),
			Navbar(),
			Container(
				body,
			),
			ChatWidget(),
		),
	)
}

func Container(children ...Node) Node {
	return Div(Class("container mx-auto p-4"), Group(children))
}

func Navbar() Node {
	return Div(Class("navbar bg-base-100 shadow-lg mb-4"),
		Div(Class("flex-1"),
			A(Class("btn btn-ghost text-xl"), Text("GKE Status")),
		),
	)
}

func ChatWidget() Node {
	return Div(Class("fixed bottom-4 right-4 w-96 bg-white shadow-xl rounded-lg border border-gray-200 overflow-hidden z-50 flex flex-col"),
		Div(Class("bg-blue-600 text-white p-3 font-bold flex justify-between items-center"),
			Text("AI Assistant"),
		),
		Div(Class("h-96 overflow-y-auto p-3 bg-gray-50 flex flex-col gap-2"),
			ID("chat-messages"),
		),
		FormEl(
			Class("p-3 border-t flex"),
			Attr("onsubmit", "sendMessage(event)"),
			Input(Type("text"), Name("message"), Class("input input-bordered w-full mr-2"), Placeholder("Ask about GKE...")),
			Button(Class("btn btn-primary"), Text("Send")),
		),
	)
}
