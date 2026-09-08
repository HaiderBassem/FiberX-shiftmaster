package assistant

import "encoding/json"

// Event is one unit of the assistant's streamed reply. The handler serialises
// events as SSE; the frontend renders them in order. Everything the user sees
// in a structured card (tool data, approval summaries) is server-built —
// model-authored content only ever appears in "text" events.
type Event struct {
	Type string `json:"type"` // status | text | tool | approval | done | error

	// status: a short machine phase for the UI spinner ("thinking", "tool").
	// text: a model-authored chunk of the reply.
	Text string `json:"text,omitempty"`

	// tool events: which tool ran and its (bounded) result or error.
	Tool    string          `json:"tool,omitempty"`
	OK      bool            `json:"ok,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`

	// approval: the staged pending action for the approval card.
	Action json.RawMessage `json:"action,omitempty"`

	// done: closing metadata.
	ConversationID string `json:"conversation_id,omitempty"`
}

// Emit is how the orchestrator hands events to the transport. Implementations
// must be cheap; a slow client must not stall tool execution indefinitely
// (the handler buffers via the HTTP write path).
type Emit func(Event)
