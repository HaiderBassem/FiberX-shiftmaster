// Package llm defines the narrow contract the assistant has with a language
// model provider, and the Anthropic implementation of it.
//
// The interface is deliberately small: one blocking completion call. The model
// is treated as an untrusted planner — nothing in this package executes tools,
// touches the database, or knows about authorization. It converts conversation
// state to provider wire format and back, and that is all.
package llm

import (
	"context"
	"encoding/json"
	"errors"
)

// Content block types used across the conversation transcript.
const (
	BlockText       = "text"
	BlockToolUse    = "tool_use"
	BlockToolResult = "tool_result"
)

// Message roles. The Messages API only distinguishes user and assistant; tool
// results travel inside a user message.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Stop reasons the orchestrator dispatches on.
const (
	StopEndTurn   = "end_turn"
	StopToolUse   = "tool_use"
	StopMaxTokens = "max_tokens"
)

// ContentBlock is one unit of conversation content. Exactly one shape is
// populated depending on Type; the JSON tags carry Anthropic's wire names so
// the same struct round-trips through the API and through our own storage.
type ContentBlock struct {
	Type string `json:"type"`

	// BlockText
	Text string `json:"text,omitempty"`

	// BlockToolUse (authored by the model)
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// BlockToolResult (authored by us, echoing ID as ToolUseID)
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// TextBlock builds a plain text block.
func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: BlockText, Text: text}
}

// ToolResultBlock builds the reply to one tool_use block. The payload is always
// a JSON document rendered to a string: handing the model one opaque string per
// result keeps retrieved application data clearly framed as data.
func ToolResultBlock(toolUseID, payload string, isErr bool) ContentBlock {
	return ContentBlock{Type: BlockToolResult, ToolUseID: toolUseID, Content: payload, IsError: isErr}
}

// Message is one conversation turn.
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// Tool describes one callable tool in the provider's schema format.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// Request is one completion call.
type Request struct {
	System    string
	Messages  []Message
	Tools     []Tool
	MaxTokens int
}

// Usage reports token consumption for observability. Never used for logic.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Response is the provider's reply.
type Response struct {
	Content    []ContentBlock
	StopReason string
	Usage      Usage
}

// Error classification. The orchestrator maps these to user-facing behaviour:
// unavailability degrades gracefully, a bad request is our bug and is logged as
// such, and misconfiguration is surfaced to the operator.
var (
	// ErrUnavailable: the provider timed out, is overloaded, or kept failing
	// after retries. Transient from the caller's point of view.
	ErrUnavailable = errors.New("model provider unavailable")

	// ErrBadRequest: the provider rejected the request as malformed. This is a
	// defect in our request construction, not a transient condition.
	ErrBadRequest = errors.New("model request rejected")

	// ErrAuth: the provider rejected our credentials. Operator misconfiguration.
	ErrAuth = errors.New("model provider authentication failed")
)

// Client is the single seam between the assistant and any model provider.
// Tests substitute a scripted implementation.
type Client interface {
	Complete(ctx context.Context, req Request) (*Response, error)
}
