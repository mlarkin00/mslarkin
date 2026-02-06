package ui

import (
	"fmt"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
	hx "maragu.dev/gomponents-htmx"
	"github.com/mslarkin/models/model-chat/internal/ai"
	"github.com/sashabaranov/go-openai"
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
			Script(Raw(`
				document.addEventListener('keydown', function(e) {
					if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
						const form = document.querySelector('form');
						if (form) htmx.trigger(form, 'submit');
					}
				});
						if (form) htmx.trigger(form, 'submit');
					}
				});
				// No thinking indicator for now as it's complex with grid view partial updates
			`)),
		),
		Body(
			Group(children),
		),
	)
}

func App(models []ai.Model, selectedModelIDs []string) Node {
	return Div(
		Class("app-container"),
		// Sidebar
		Div(
			Class("sidebar"),
			H1(Class("app-title"), Text("Model Chat")),

			Div(
				Label(Class("label"), Text("Select Models")),
				ModelSelector(models, selectedModelIDs),
			),

			Div(
				Class("mcp-servers-section"),
				H3(Class("section-title"), Text("Available MCP Servers")),
				Ul(
					Class("mcp-list"),
					Li(Text("context7-mcp")),
					Li(Text("docs-onemcp")),
					Li(Text("gke-onemcp")),
					Li(Text("gke-oss")),
				),
				Div(
					Class("external-links"),
					A(Href("https://github.com/GoogleCloudPlatform/agent-development-kit"), Class("nav-link"), Text("Google ADK Dev UI"), Target("_blank")), // Updated Link
				),
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
				Message(true, "System", "Select models and start chatting!", nil),
			),
			// Input Area
			Div(
				Class("input-area"),
				FormEl(
					hx.Post("/chat"),
					hx.Target("#chat-history"),
					hx.Swap("beforeend"),
					hx.On("htmx:afterRequest", "this.reset()"), // Reset form after send
					hx.Include("input[name='model_id']"),       // Include selected models
					Class("chat-form"),
					// Hidden inputs for model_ids are now handled by the checkboxes having the name "model_id" directly
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

func ModelSelector(models []ai.Model, selectedIDs []string) Node {
	return Div(
		Class("model-selector-checkboxes"),
		Group(Map(models, func(m ai.Model) Node {
			isSelected := false
			for _, id := range selectedIDs {
				if id == m.ID {
					isSelected = true
					break
				}
			}
			return Div(
				Class("checkbox-item"),
				Input(
					Type("checkbox"),
					Name("model_id"), // Use same name for array binding in backend
					ID("model-"+m.ID),
					Value(m.ID),
					If(isSelected, Checked()),
				),
				Label(For("model-"+m.ID), Text(m.DisplayName)),
			)
		})),
	)
}

func Message(isUser bool, sender string, content string, metrics *ai.ChatResponse) Node {
	wrapperClass := "message-wrapper ai"
	if isUser {
		wrapperClass = "message-wrapper user"
	}

	var metricsNodes Node
	if metrics != nil && metrics.Usage != (openai.Usage{}) {
		inTokens := fmt.Sprintf("In: %d", metrics.Usage.PromptTokens)
		outTokens := fmt.Sprintf("Out: %d", metrics.Usage.CompletionTokens)
		totalTokens := fmt.Sprintf("Total: %d", metrics.Usage.TotalTokens)

		metricsNodes = Div(Class("message-metrics"),
			Span(Text(inTokens)),
			Span(Text("|")),
			Span(Text(outTokens)),
			Span(Text("|")),
			Span(Text(totalTokens)),
			If(metrics.Thinking != "", Group([]Node{
				Span(Text("|")),
				Span(Text("Think: Yes")),
			})),
		)
	} else {
		metricsNodes = Group(nil) // Render nothing if no metrics
	}

	return Div(
		Class(wrapperClass),
		Div(
			Class("message-bubble"),
			Div(Class("message-sender"), Text(sender)),
			Div(Class("message-content"), Text(content)),
			metricsNodes,
		), // Close of message-bubble Div
	) // Close of wrapperClass Div
}

func ComparisonView(userMsg string, responses map[string]*ai.ChatResponse, errors map[string]error) Node {
	return Div(
		Class("comparison-row"),
		// User Message (Full Width)
		Message(true, "User", userMsg, nil),
		// Responses Grid
		Div(
			Class("responses-grid"),
			Style("display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem;"),
			MapKeys(responses, func(modelID string) Node {
				resp := responses[modelID]
				if resp == nil {
					return Message(false, modelID, "No response (nil)", nil)
				}
				return Message(false, modelID, resp.Content, resp)
			}),
			MapKeys(errors, func(modelID string) Node {
				err := errors[modelID]
				return Message(false, modelID, fmt.Sprintf("Error: %v", err), nil)
			}),
		),
	)
}

// MapKeys is a helper to map over map keys (since Go doesn't have consistent map iteration order, strictly speaking we should sort, but for this demo random is okay-ish or we can sort keys)
func MapKeys[V any](m map[string]V, f func(string) Node) Node {
	// For consistent ordering, let's sort keys if possible, but for now just iterate
	// Ideally pass models slice to preserve order.
	// Refactor: ComparisonView should probably take []Struct{ModelID, Response, Error} to preserve order
	return Group(MapResult(m, f))
}

func MapResult[V any](m map[string]V, f func(string) Node) []Node {
	var nodes []Node
	for k := range m {
		nodes = append(nodes, f(k)) // Note: Random order
	}
	return nodes
}


