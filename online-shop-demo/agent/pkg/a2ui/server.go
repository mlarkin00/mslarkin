package a2ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Server handles A2UI streaming and actions
type Server struct {
	mu           sync.RWMutex
	clients      map[chan Message]bool
	currentState []Message // Replay buffer for new clients (simplified state)
}

func NewServer() *Server {
	return &Server{
		clients: make(map[chan Message]bool),
	}
}

// Broadcast sends a message to all connected clients
func (s *Server) Broadcast(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentState = append(s.currentState, msg)

	for ch := range s.clients {
		select {
		case ch <- msg:
		default:
			// Client blocked, drop or disconnect (simple implementation drops)
		}
	}
}

// HandleStream serves the Server-Sent Events stream
func (s *Server) HandleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan Message, 10)

	s.mu.Lock()
	s.clients[ch] = true
	// Replay current state
	initMsgs := make([]Message, len(s.currentState))
	copy(initMsgs, s.currentState)
	s.mu.Unlock()

	// Send initial state
	for _, msg := range initMsgs {
		s.writeMessage(w, msg)
	}
	flusher.Flush()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
		close(ch)
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			s.writeMessage(w, msg)
			flusher.Flush()
		}
	}
}

func (s *Server) writeMessage(w http.ResponseWriter, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	// A2UI spec: JSONL stream. SSE usually uses "data: ...\n\n"
	// To be purely JSONL over HTTP, we wouldn't use "data: " prefix of SSE.
	// But spec mentions SSE as a transport.
	// Standard SSE format:
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// HandleAction receives user actions from the client
func (s *Server) HandleAction(w http.ResponseWriter, r *http.Request, handler func(UserAction)) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}


	// The client sends { "userAction": { ... } } or just the action?
	// Spec says: { "userAction": { "name": ..., "context": ... } }
	var container struct {
		UserAction UserAction `json:"userAction"`
	}

	if err := json.NewDecoder(r.Body).Decode(&container); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Async handle to not block HTTP response
	go handler(container.UserAction)

	w.WriteHeader(http.StatusOK)
}

// Helper to create a literal text component
func MakeText(id, content, usage string) ComponentWrapper {
	return ComponentWrapper{
		ID: id,
		Component: Component{
			Text: &Text{
				Text: BoundValue{LiteralString: content},
				UsageHint: usage,
			},
		},
	}
}

// Helper to create a button
func MakeButton(id, labelID, labelText, actionName string, context map[string]string) []ComponentWrapper {
	// Create label text component
	textComp := MakeText(labelID, labelText, "")

	ctxVars := []ContextVariable{}
	for k, v := range context {
		ctxVars = append(ctxVars, ContextVariable{Key: k, Value: BoundValue{LiteralString: v}})
	}

	btnComp := ComponentWrapper{
		ID: id,
		Component: Component{
			Button: &Button{
				Child: labelID,
				Action: Action{
					Name:    actionName,
					Context: ctxVars,
				},
			},
		},
	}
	return []ComponentWrapper{textComp, btnComp}
}

// Helper to create a select dropdown
func MakeSelect(id, label string, options []Option, selected string, actionName string, context map[string]string) []ComponentWrapper {
	ctxVars := []ContextVariable{}
	for k, v := range context {
		ctxVars = append(ctxVars, ContextVariable{Key: k, Value: BoundValue{LiteralString: v}})
	}

	return []ComponentWrapper{
		{
			ID: id,
			Component: Component{
				Select: &Select{
					Label:    label,
					Options:  options,
					Selected: selected,
					Action: Action{
						Name:    actionName,
						Context: ctxVars,
					},
				},
			},
		},
	}
}
