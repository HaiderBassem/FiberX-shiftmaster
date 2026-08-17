package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"shiftmaster-backend/internal/assistant/llm"
)

// Tool is one capability the model may invoke. The contract every tool obeys:
//
//   - The actor comes exclusively from the server (LoadActor); no tool input
//     carries an employee ID, role, or permission — the schemas simply have no
//     such fields, so "pretend I am X" has nothing to attach to.
//   - Authorization happens inside Run against live database state, on every
//     call, regardless of conversation history.
//   - Results are bounded (LIMITs in SQL or explicit caps) — a tool result is
//     model context, not a data export.
//   - Reads answer directly. Writes only ever stage a pending action; nothing
//     in the tool layer mutates domain state.
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
	// Roles restricts catalogue visibility; nil means every authenticated
	// role. This is presentation-level pruning — Run re-checks authority
	// itself, so a tool call smuggled past the catalogue still fails closed.
	Roles []string

	Run func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error)
}

// userError is a message safe and useful to show to the model and the user:
// a business rule, a validation failure, an authorization denial. Anything
// else that goes wrong inside a tool is reported generically and logged.
type userError struct{ msg string }

func (e userError) Error() string { return e.msg }

// Errf builds a user-visible tool error.
func Errf(format string, args ...any) error {
	return userError{msg: fmt.Sprintf(format, args...)}
}

// asUserMessage converts any tool error into what the model may see.
func asUserMessage(toolName string, err error) string {
	var ue userError
	if errors.As(err, &ue) {
		return ue.msg
	}
	// Not written for users: log the detail server-side, return a generic
	// marker. Internal errors must not leak schema or infrastructure details
	// into model context.
	log.Printf("assistant: tool %s internal error: %v", toolName, err)
	return "internal error while running this tool; the result is unavailable"
}

// visibleTo reports catalogue visibility for a role.
func (t Tool) visibleTo(role string) bool {
	if len(t.Roles) == 0 {
		return true
	}
	for _, r := range t.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Registry is the ordered tool catalogue.
type Registry struct {
	tools []Tool
	index map[string]int
}

func NewRegistry(tools []Tool) *Registry {
	r := &Registry{tools: tools, index: make(map[string]int, len(tools))}
	for i, t := range tools {
		if _, dup := r.index[t.Name]; dup {
			panic("assistant: duplicate tool name " + t.Name)
		}
		r.index[t.Name] = i
	}
	return r
}

// ForRole renders the provider-format tool list for one role.
func (r *Registry) ForRole(role string) []llm.Tool {
	var out []llm.Tool
	for _, t := range r.tools {
		if t.visibleTo(role) {
			out = append(out, llm.Tool{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
		}
	}
	return out
}

// Lookup returns a tool if it exists AND is visible to the role. An invisible
// tool is indistinguishable from an unknown one, so the model cannot probe
// for capabilities above the caller's role.
func (r *Registry) Lookup(name, role string) (Tool, bool) {
	i, ok := r.index[name]
	if !ok {
		return Tool{}, false
	}
	t := r.tools[i]
	if !t.visibleTo(role) {
		return Tool{}, false
	}
	return t, true
}

// schema is a convenience for inline JSON Schema literals; it panics at init
// on malformed JSON so a typo cannot ship.
func schema(s string) json.RawMessage {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic("assistant: invalid tool schema: " + err.Error())
	}
	return json.RawMessage(s)
}

// decode parses tool input strictly: unknown fields are rejected so the model
// cannot smuggle extra parameters past a schema.
func decode(input json.RawMessage, into any) error {
	if len(input) == 0 {
		input = json.RawMessage("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return Errf("invalid tool input: %v", err)
	}
	return nil
}
