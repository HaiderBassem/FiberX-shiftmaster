package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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
	Rounds       int
	// ToolFree records that the model explicitly declared the turn needed no
	// system data. The evaluation harness asserts on this to catch answers
	// invented from the model's priors.
	ToolFree bool
	// Dropped counts transcript messages the token budget forced out.
	Dropped int
	// FirstTokenMs is how long the first model round took, which is what the
	// person actually experiences as "did it hear me".
	FirstTokenMs int
	// TokensPerSec is the generation rate the runtime last reported.
	TokensPerSec float64
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
//   - The prompt is fitted to the model's real context window before every
//     round, after the fixed cost of the system prompt and tool catalogue is
//     known.
func (d *Deps) runTurn(
	ctx context.Context,
	actor *Actor,
	registry *Registry,
	history []llm.Message,
	emit Emit,
) ([]llm.Message, *turnStats, error) {
	stats := &turnStats{}
	system := buildSystemPrompt(ctx, d, actor, latestUserText(history))
	tools := registry.ForRole(actor.Role())

	// What is left of the context window for conversation, once the standing
	// instructions, the catalogue and the reply have taken their share.
	transcriptBudget := d.Cfg.ContextSize -
		estimateTokens(system) -
		toolCatalogueTokens(tools) -
		d.Cfg.MaxTokens -
		256 // template scaffolding and rounding

	messages := history
	var appended []llm.Message
	// repeats detects a model looping on one call; see below.
	repeats := map[string]int{}

	for round := 0; ; round++ {
		stats.Rounds = round + 1
		if round >= d.Cfg.MaxToolRounds {
			// Budget exhausted: make the model produce a final answer with the
			// tools it has already run, rather than looping forever.
			note := llm.Message{Role: llm.RoleUser, Content: []llm.ContentBlock{
				llm.TextBlock("[system: no more lookups are possible this turn — answer now with what you already have, and say plainly if something is still missing]"),
			}}
			messages = append(messages, note)
			appended = append(appended, note)
			tools = nil
		}

		fitted, dropped := fitWindow(messages, transcriptBudget)
		stats.Dropped += dropped

		emit(Event{Type: "status", Text: "thinking"})
		modelStart := time.Now()
		resp, err := d.LLM.Complete(ctx, llm.Request{
			System:   system,
			Messages: fitted,
			Tools:    tools,
			Round:    round,
		})
		elapsed := int(time.Since(modelStart).Milliseconds())
		stats.ModelMs += elapsed
		if round == 0 {
			stats.FirstTokenMs = elapsed
		}
		if err != nil {
			return appended, stats, err
		}
		stats.InputTokens += resp.Usage.InputTokens
		stats.OutputTokens += resp.Usage.OutputTokens
		if resp.Timings.TokensPerSec > 0 {
			stats.TokensPerSec = resp.Timings.TokensPerSec
		}
		if resp.ToolFree && round == 0 {
			stats.ToolFree = true
		}

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

			// A model that asks the same question twice will ask it forever;
			// answering the repeat with a nudge instead of the same payload
			// costs one round and breaks the loop. This is a deterministic
			// safety rail around the model, not a substitute for its judgement.
			key := callKey(block)
			repeats[key]++
			if repeats[key] > 1 {
				const msg = "you already called this tool with these exact arguments in this turn; the earlier result above is still current — use it, or call a different tool"
				body, _ := json.Marshal(map[string]string{"error": msg})
				// Reported like any other tool failure so a model stuck in a
				// loop is visible in the request log and to the evaluation
				// suite, rather than looking like a turn that never called
				// anything.
				emit(Event{Type: "tool", Tool: block.Name, OK: false, Message: msg})
				results = append(results, llm.ToolResultBlock(block.ID, string(body), true))
				continue
			}

			payload, ok := d.execTool(ctx, actor, registry, block, emit, stats)
			results = append(results, llm.ToolResultBlock(block.ID, payload, !ok))
		}
		if len(results) == 0 {
			// The provider said tool_use but no tool blocks arrived: treat as
			// a provider inconsistency and end the turn safely.
			return appended, stats, fmt.Errorf("%w: tool_use stop without tool blocks", llm.ErrUnavailable)
		}
		resultMsg := llm.Message{Role: llm.RoleUser, Content: results}
		messages = append(messages, resultMsg)
		appended = append(appended, resultMsg)
	}
}

// latestUserText is the message this turn is answering, used only to decide
// which language the reply must be written in.
func latestUserText(history []llm.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role != llm.RoleUser {
			continue
		}
		for _, block := range history[i].Content {
			if block.Type == llm.BlockText && block.Text != "" {
				return block.Text
			}
		}
	}
	return ""
}

// callKey identifies one tool call by name and arguments.
func callKey(block llm.ContentBlock) string {
	sum := sha256.Sum256(append([]byte(block.Name+"\x00"), block.Input...))
	return hex.EncodeToString(sum[:8])
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
		// Either a name the grammar should have made impossible, or a tool the
		// caller's role cannot see. Both answer identically, so the model
		// cannot probe for capabilities above the caller's authority.
		log.Printf("assistant: %s called unavailable tool %q", actor.Role(), call.Name)
		return fail("that tool does not exist for this user")
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
