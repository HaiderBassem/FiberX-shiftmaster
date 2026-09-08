// Package llm defines the narrow contract the assistant has with a language
// model, and the local implementation of it.
//
// The model runs on this machine — a llama.cpp server on loopback or a Unix
// socket — and nothing in this package can reach a hosted AI service. That is
// deliberate and load-bearing: employee schedules, leave records and internal
// documents pass through here, and the only way to be sure they do not leave
// the premises is for there to be no code that could send them. The Anthropic
// client this package used to contain was removed for exactly that reason,
// along with the API key that gated the whole feature.
//
// The interface is deliberately small: one blocking completion call. The model
// is treated as an untrusted planner — nothing in this package executes tools,
// touches the database, or knows about authorization. It converts conversation
// state to the runtime's wire format and back, and that is all.
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
// populated depending on Type. The JSON tags are the conversation's storage
// format: a transcript written by one release must stay readable by the next,
// so these names are a compatibility surface, not an implementation detail.
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
	// Round is the zero-based model round within the current conversation
	// turn. A provider driving a small local model uses it to force an
	// explicit tool decision on the first round; hosted providers ignore it.
	Round int
	// Warmup marks a call made only to populate the runtime's prompt cache.
	// Its output is discarded, so the provider must not spend retries trying
	// to coax a usable answer out of it.
	Warmup bool
}

// Usage reports token consumption for observability. Never used for logic.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Timings is what the local runtime measured for one call. Observability only:
// prompt processing, generation, and the achieved rate. Absent for providers
// that do not report it.
type Timings struct {
	PromptMs     float64 `json:"prompt_ms,omitempty"`
	GenerationMs float64 `json:"generation_ms,omitempty"`
	TokensPerSec float64 `json:"tokens_per_sec,omitempty"`
}

// Response is the provider's reply.
type Response struct {
	Content    []ContentBlock
	StopReason string
	Usage      Usage
	Timings    Timings
	// ToolFree records that the model explicitly declared this turn needs no
	// system data. The evaluation harness asserts on it: an answer about real
	// schedule or leave data that arrives tool-free is a grounding failure,
	// and this is how that becomes detectable rather than a matter of opinion.
	ToolFree bool
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

	// ErrBusy: every inference slot is occupied and the caller's wait budget
	// expired before one freed up. Backpressure, not a fault.
	ErrBusy = errors.New("model runtime is busy")
)

// Lifecycle states reported by a runtime. They are deliberately coarse: the
// frontend must be able to render an honest state without ever learning a
// model path, a port, or a stack trace.
const (
	// StateDisabled: the operator turned the assistant off.
	StateDisabled = "disabled"
	// StateStarting: the runtime is loading the model; requests will work
	// shortly.
	StateStarting = "starting"
	// StateReady: the model is loaded and answering.
	StateReady = "ready"
	// StateDegraded: reachable but recently failing, or restarting.
	StateDegraded = "degraded"
	// StateUnavailable: configured but not reachable.
	StateUnavailable = "unavailable"
)

// Health is the safe, public view of the runtime's condition.
type Health struct {
	State string `json:"state"`
	// Detail is a short, non-sensitive reason code for the UI ("loading_model",
	// "not_running"). Never a path, port, credential or traceback.
	Detail string `json:"detail,omitempty"`
}

// Prober is implemented by anything that can report runtime health to the
// status endpoint.
type Prober interface {
	Health() Health
}

// Client is the single seam between the assistant and any model provider.
// Tests substitute a scripted implementation.
type Client interface {
	Complete(ctx context.Context, req Request) (*Response, error)
}
