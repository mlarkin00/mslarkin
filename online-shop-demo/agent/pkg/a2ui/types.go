package a2ui

// Message represents a top-level A2UI message in the stream
type Message struct {
	SurfaceUpdate   *SurfaceUpdate   `json:"surfaceUpdate,omitempty"`
	DataModelUpdate *DataModelUpdate `json:"dataModelUpdate,omitempty"`
	BeginRendering  *BeginRendering  `json:"beginRendering,omitempty"`
	ClientEvent     *ClientEvent     `json:"clientEvent,omitempty"`
	UserAction      *UserAction      `json:"userAction,omitempty"`
	Error           *Error           `json:"error,omitempty"`
}

type BeginRendering struct {
	Root string `json:"root"` // ID of the root component
}

type SurfaceUpdate struct {
	SurfaceID  string             `json:"surfaceId,omitempty"`
	Components []ComponentWrapper `json:"components"`
}

type ComponentWrapper struct {
	ID        string    `json:"id"`
	Component Component `json:"component"`
}

type Component struct {
	Column *Column `json:"Column,omitempty"`
	Row    *Row    `json:"Row,omitempty"`
	Text   *Text   `json:"Text,omitempty"`
	Button *Button `json:"Button,omitempty"`
	Image  *Image  `json:"Image,omitempty"`
	Card   *Card   `json:"Card,omitempty"`
	Select *Select `json:"Select,omitempty"`
}

type Select struct {
	Label    string   `json:"label,omitempty"`
	Options  []Option `json:"options"`
	Selected string   `json:"selected,omitempty"`
	Action   Action   `json:"action,omitempty"`
}

type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Column struct {
	Children  Children `json:"children,omitempty"`
	Alignment string   `json:"alignment,omitempty"` // start, center, end
}

type Row struct {
	Children  Children `json:"children,omitempty"`
	Alignment string   `json:"alignment,omitempty"` // start, center, end
}

type Text struct {
	Text      BoundValue `json:"text"`
	UsageHint string     `json:"usageHint,omitempty"` // h1, h2, h3, etc.
}

type Button struct {
	Child   string `json:"child"` // Component ID of the button label
	Action  Action `json:"action,omitempty"`
	Variant string `json:"variant,omitempty"` // e.g., "primary", "danger", "neutral", "success"
}

type Image struct {
	URL BoundValue `json:"url"`
}

type Card struct {
	Child string `json:"child"`
}

type Children struct {
	ExplicitList []string `json:"explicitList,omitempty"` // List of Component IDs
}

type Action struct {
	Name    string            `json:"name"`
	Context []ContextVariable `json:"context,omitempty"`
}

type ContextVariable struct {
	Key   string     `json:"key"`
	Value BoundValue `json:"value"`
}

// BoundValue represents a value that can be literal or bound to data model
type BoundValue struct {
	LiteralString string `json:"literalString,omitempty"`
	Path          string `json:"path,omitempty"` // JSON pointer to data model
}

type DataModelUpdate struct {
	Path     string            `json:"path,omitempty"`
	Contents map[string]string `json:"contents"` // Simplified for this demo
}

// Client-to-Server types

type UserAction struct {
	Name    string            `json:"name"`
	Context map[string]string `json:"context,omitempty"` // Resolved context
}

type ClientEvent struct {
	// Generic event
}

type Error struct {
	Message string `json:"message"`
}
