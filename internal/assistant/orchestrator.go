package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"shiftmaster-backend/internal/assistant/llm"
)

// turnStats is what one conversation turn cost, for the request log.
type turnStats struct {
	ToolCalls    int
	ModelMs      int
	ToolsMs      int
	InputTokens  int
	OutputTokens int
}

// runTurn drives one user message to completion: model → tools → model …
// until the model stops asking for tools or the round budget runs out. It
// returns every message that must be appended to the stored transcript (the
// user turn is appended by the caller).
//
// Invariants:
//   - Tool calls execute strictly through the registry with the actor's role;
//     an unknown or role-invisible tool yields an error result, never a crash
//     and never a lookup outside the role's catalogue.
//   - Tool errors become is_error tool_results; the turn continues so the
//     model can explain the failure. Model transport errors abort the turn.
//   - Every round appends BOTH the assistant tool_use message and the paired
//     tool_result message, so the stored transcript is always well-formed for
//     the next call.
func (d *Deps) runTurn(
	ctx context.Context,
	actor *Actor,
	registry *Registry,
	history []llm.Message,
	emit Emit,
) ([]llm.Message, *turnStats, error) {
	stats := &turnStats{}
	system := buildSystemPrompt(ctx, d, actor)
	tools := registry.ForRole(actor.Role())

	messages := history
	var appended []llm.Message

	for round := 0; ; round++ {
		if round >= d.Cfg.MaxToolRounds {
			// Budget exhausted: make the model produce a final answer with the
			// tools it has already run, rather than looping forever.
			note := llm.Message{Role: llm.RoleUser, Content: []llm.ContentBlock{
				llm.TextBlock("[system: tool budget for this turn is exhausted — answer now with what you have]"),
			}}
			messages = append(messages, note)
			appended = append(appended, note)
			tools = nil
		}

		emit(Event{Type: "status", Text: "thinking"})
		modelStart := time.Now()
		resp, err := d.LLM.Complete(ctx, llm.Request{
			System:   system,
			Messages: messages,
			Tools:    tools,
		})
		stats.ModelMs += int(time.Since(modelStart).Milliseconds())
		if err != nil {
			return appended, stats, err
		}
		stats.InputTokens += resp.Usage.InputTokens
		stats.OutputTokens += resp.Usage.OutputTokens

		assistantMsg := llm.Message{Role: llm.RoleAssistant, Content: resp.Content}
		messages = append(messages, assistantMsg)
		appended = append(appended, assistantMsg)

		// Surface text blocks as they come; the model may narrate briefly
		// before calling tools.
		for _, block := range resp.Content {
			if block.Type == llm.BlockText && block.Text != "" {
				emit(Event{Type: "text", Text: block.Text})
			}
		}

		if resp.StopReason != llm.StopToolUse {
			if resp.StopReason == llm.StopMaxTokens {
				emit(Event{Type: "text", Text: "…"})
			}
			return appended, stats, nil
		}

		// Execute the round's tool calls in order.
		var results []llm.ContentBlock
		for _, block := range resp.Content {
			if block.Type != llm.BlockToolUse {
				continue
			}
			stats.ToolCalls++
			payload, ok := d.execTool(ctx, actor, registry, block, emit, stats)
			results = append(results, llm.ToolResultBlock(block.ID, payload, !ok))
		}
		if len(results) == 0 {
			// stop_reason said tool_use but no tool blocks arrived: treat as a
			// provider inconsistency and end the turn safely.
			return appended, stats, fmt.Errorf("%w: tool_use stop without tool blocks", llm.ErrUnavailable)
		}
		resultMsg := llm.Message{Role: llm.RoleUser, Content: results}
		messages = append(messages, resultMsg)
		appended = append(appended, resultMsg)
	}
}

// execTool runs one tool call defensively: role-filtered lookup, strict
// decoding inside the tool, bounded execution time, panic containment, and a
// result the model can always parse. Returns the JSON payload and ok=false
// when the payload describes an error.
func (d *Deps) execTool(
	ctx context.Context,
	actor *Actor,
	registry *Registry,
	call llm.ContentBlock,
	emit Emit,
	stats *turnStats,
) (payload string, ok bool) {
	fail := func(msg string) (string, bool) {
		emit(Event{Type: "tool", Tool: call.Name, OK: false, Message: msg})
		body, _ := json.Marshal(map[string]string{"error": msg})
		return string(body), false
	}

	tool, found := registry.Lookup(call.Name, actor.Role())
	if !found {
		return fail("unknown tool")
	}

	emit(Event{Type: "status", Text: "tool:" + tool.Name})
	toolStart := time.Now()

	toolCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	result, err := func() (result any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("tool panic: %v", r)
			}
		}()
		return tool.Run(toolCtx, d, actor, call.Input)
	}()
	stats.ToolsMs += int(time.Since(toolStart).Milliseconds())

	if err != nil {
		return fail(asUserMessage(tool.Name, err))
	}

	body, err := json.Marshal(result)
	if err != nil {
		return fail(asUserMessage(tool.Name, err))
	}
	// Cap what one tool result may occupy in model context. Tools bound their
	// own output, so hitting this means a tool bug — degrade, don't explode.
	if len(body) > 24*1024 {
		return fail("tool result too large")
	}

	// Approval cards get their own event so the UI renders them prominently.
	if pending, isProposal := extractPendingAction(body); isProposal {
		emit(Event{Type: "approval", Tool: tool.Name, OK: true, Action: pending})
	} else {
		emit(Event{Type: "tool", Tool: tool.Name, OK: true, Data: json.RawMessage(body)})
	}
	return string(body), true
}

// extractPendingAction detects a propose_* result and lifts the card payload.
func extractPendingAction(body []byte) (json.RawMessage, bool) {
	var probe struct {
		PendingAction json.RawMessage `json:"pending_action"`
	}
	if err := json.Unmarshal(body, &probe); err != nil || len(probe.PendingAction) == 0 {
		return nil, false
	}
	return probe.PendingAction, true
}
