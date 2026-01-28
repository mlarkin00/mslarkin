package a2ui

// Component represents a generic A2UI component.
type Component struct {
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type"`
	Props    map[string]interface{} `json:"props,omitempty"`
	Children []Component            `json:"children,omitempty"`
}

// Helper functions to create common components

func Text(content string) Component {
	return Component{
		Type: "text",
		Props: map[string]interface{}{
			"content": content,
		},
	}
}

func Container(children ...Component) Component {
	return Component{
		Type:     "container",
		Children: children,
	}
}

func Card(title string, children ...Component) Component {
	return Component{
		Type: "card",
		Props: map[string]interface{}{
			"title": title,
		},
		Children: children,
	}
}

func Button(label string, action string) Component {
	return Component{
		Type: "button",
		Props: map[string]interface{}{
			"label": label,
			"onClick": map[string]string{
				"action": "send_prompt",
				"prompt": action,
			},
		},
	}
}

func Table(headers []string, rows [][]string) Component {
	// A2UI table structure might vary, this is a simplified guess based on common patterns.
	// In a real A2UI spec, we'd check the schema.
    // Assuming a simple "table" component with "headers" and "rows" props.
	return Component{
		Type: "table",
		Props: map[string]interface{}{
			"headers": headers,
			"rows":    rows,
		},
	}
}
