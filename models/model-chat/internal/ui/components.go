package ui

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
	hx "maragu.dev/gomponents-htmx"
	"github.com/mslarkin/models/model-chat/internal/ai"
)

func Layout(title string, children ...Node) Node {
	return HTML(
		Head(
			TitleEl(Text(title)),
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Link(Rel("stylesheet"), Href("/static/styles.css")),
			Script(Src("https://unpkg.com/htmx.org@1.9.10")),
			// Add a font
			Link(Rel("preconnect"), Href("https://fonts.googleapis.com")),
			Link(Rel("preconnect"), Href("https://fonts.gstatic.com"), Attr("crossorigin", "")),
			Link(Rel("stylesheet"), Href("https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap")),
		),
		Body(
			Group(children),
		),
	)
}

func App(models []ai.Model, selectedModelID string) Node {
	return Div(
		Class("app-container"),
		// Sidebar
		Div(
			Class("sidebar"),
			H1(Class("app-title"), Text("Model Chat")),

			Div(
				Label(Class("label"), Text("Select Model")),
				ModelSelector(models, selectedModelID),
			),

			Div(Class("footer"),
				P(Text("Powered by Vertex AI")),
			),
		),
		// Main Chat Area
		Div(
			Class("main-area"),
			// Chat History
			Div(
				ID("chat-history"),
				Class("chat-history"),
				// Welcome message
				Message(false, "Select a model and start chatting!"),
			),
			// Input Area
			Div(
				Class("input-area"),
				FormEl(
					hx.Post("/chat"),
					hx.Target("#chat-history"),
					hx.Swap("beforeend"),
					hx.On("htmx:afterRequest", "this.reset()"), // Reset form after send
					Class("chat-form"),
					Input(
						Type("hidden"),
						Name("model_id"),
						ID("current-model-id"),
						Value(selectedModelID),
					),
					Input(
						Type("text"),
						Name("message"),
						Class("chat-input"),
						Placeholder("Type your message..."),
						Attr("autocomplete", "off"),
						Attr("autofocus", ""),
					),
					Button(
						Type("submit"),
						Class("send-btn"),
						Text("Send"),
					),
				),
			),
		),
	)
}

func ModelSelector(models []ai.Model, selected string) Node {
	return Select(
		Name("model_id_selector"),
		Class("model-select"),
		// On change, update the hidden input in the chat form
		Attr("onchange", "document.getElementById('current-model-id').value = this.value"),
		Group(Map(models, func(m ai.Model) Node {
			return Option(
				Value(m.ID),
				Text(m.DisplayName),
				If(m.ID == selected, Selected()),
			)
		})),
	)
}

func Message(isUser bool, content string) Node {
	wrapperClass := "message-wrapper ai"
	if isUser {
		wrapperClass = "message-wrapper user"
	}

	// Check if this is a "Thinking" block (rudimentary check)
	// In a real app we might parse markdown or XML tags

	return Div(
		Class(wrapperClass),
		Div(
			Class("message-bubble"),
			Text(content),
		),
	)
}


