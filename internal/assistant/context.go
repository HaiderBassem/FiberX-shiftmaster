package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"shiftmaster-backend/internal/assistant/llm"
	"shiftmaster-backend/internal/models"
)

// Context management for a model with a fixed, local context window.
//
// A hosted model with a 200k window lets you replay the whole dialogue and
// think no further. A locally hosted one does not: the window is a few
// thousand tokens shared between the system prompt, the tool catalogue, the
// transcript and the answer. Overflow there is not a cost problem, it is a
// correctness problem — llama.cpp discards the head of the prompt, which is
// exactly where the identity, authority and time facts live.
//
// So the transcript is bounded three ways, in order:
//
//  1. by message count, in SQL (MaxContextMsgs);
//  2. by estimated tokens, here, after the fixed cost of the system prompt and
//     tool catalogue is known;
//  3. by aging tool results — old retrieved payloads shrink to a marker while
//     the conversation around them stays intact, because what a follow-up turn
//     needs from an old tool call is that it happened, not its every field.
//
// What scrolls out is not simply forgotten: the service writes a short recap
// into the conversation row afterwards, and the prompt builder injects it. The
// recap is memory, never evidence — facts are re-fetched through tools every
// time.

// estimateTokens approximates a tokenizer without calling one. The ratio is
// deliberately pessimistic (≈3 bytes per token): Arabic text costs about two
// bytes per character and JSON is punctuation-heavy, and over-reserving costs
// a little history while under-reserving silently truncates the system prompt.
func estimateTokens(s string) int {
	return len(s)/3 + 4
}

func estimateBlocks(blocks []llm.ContentBlock) int {
	n := 0
	for _, b := range blocks {
		n += estimateTokens(b.Text) + estimateTokens(b.Content) + estimateTokens(string(b.Input)) + estimateTokens(b.Name) + 8
	}
	return n
}

func estimateMessages(msgs []llm.Message) int {
	n := 0
	for _, m := range msgs {
		n += estimateBlocks(m.Content) + 4
	}
	return n
}

// toolCatalogueTokens is the fixed cost of shipping the tool definitions,
// which the chat template renders into the prompt on every call.
func toolCatalogueTokens(tools []llm.Tool) int {
	n := 0
	for _, t := range tools {
		n += estimateTokens(t.Name) + estimateTokens(t.Description) + estimateTokens(string(t.InputSchema)) + 8
	}
	return n
}

// agedResultLimit is how much of an older tool result survives. Enough to keep
// the gist ("there were 4 tasks", "the shift ended 00:30") without carrying a
// full payload forward for the rest of the dialogue.
const agedResultLimit = 220

// keepFreshMessages is how many trailing messages keep their tool results
// verbatim. Two full model↔tool rounds: the current turn's evidence is never
// aged out from under the model mid-reasoning.
const keepFreshMessages = 6

// fitWindow trims a transcript to a token budget and returns how many leading
// messages it had to drop.
//
// Three invariants hold on the result:
//   - it starts at a clean boundary (a user text message), because a tool
//     result whose tool_use was trimmed away is a malformed conversation;
//   - the newest messages are always kept — the current turn is never the part
//     that gets dropped;
//   - aging is tried before dropping, so history is lost only when shrinking
//     was not enough.
func fitWindow(msgs []llm.Message, budget int) (kept []llm.Message, dropped int) {
	if budget <= 0 || len(msgs) == 0 {
		return msgs, 0
	}
	work := msgs
	if estimateMessages(work) > budget {
		work = ageToolResults(work)
	}

	// Drop from the front until it fits, then walk forward to a clean start.
	start := 0
	for start < len(work) && estimateMessages(work[start:]) > budget {
		start++
	}
	for start < len(work) && !isCleanStart(work[start]) {
		start++
	}
	if start >= len(work) {
		// Everything is over budget on its own: keep the final user message so
		// the model at least sees the question, aged as far as it goes.
		last := work[len(work)-1:]
		return last, len(work) - 1
	}
	return work[start:], start
}

// isCleanStart reports whether a message may begin a replayed window.
func isCleanStart(m llm.Message) bool {
	if m.Role != llm.RoleUser || len(m.Content) == 0 {
		return false
	}
	return m.Content[0].Type != llm.BlockToolResult
}

// ageToolResults shrinks the payloads of older tool results in place on a
// copy, leaving recent ones untouched.
func ageToolResults(msgs []llm.Message) []llm.Message {
	cutoff := len(msgs) - keepFreshMessages
	if cutoff <= 0 {
		return msgs
	}
	out := make([]llm.Message, len(msgs))
	copy(out, msgs)
	for i := 0; i < cutoff; i++ {
		var changed bool
		blocks := make([]llm.ContentBlock, len(out[i].Content))
		copy(blocks, out[i].Content)
		for j := range blocks {
			if blocks[j].Type != llm.BlockToolResult || len(blocks[j].Content) <= agedResultLimit {
				continue
			}
			blocks[j].Content = blocks[j].Content[:agedResultLimit] + ` …] (older result shortened — call the tool again if you need the details)`
			changed = true
		}
		if changed {
			out[i].Content = blocks
		}
	}
	return out
}

// ─── recap of what scrolled out ─────────────────────────────────────────────

// summaryPrompt asks for memory, not analysis. It is deliberately blunt about
// what must survive, because the failure that matters is a follow-up like
// "make it half an hour instead" losing which shift was being discussed.
const summaryPrompt = `Write a compact recap of this conversation so it can be continued after the older turns are no longer visible.

Keep, in at most 120 words:
- what the person asked about and what was found (dates, shift times, names, amounts)
- anything they decided, requested, or are waiting on
- the language and dialect they write in

Do not add advice, do not invent anything that is not above, and do not write a preamble. Write the recap only.`

// maybeSummarize refreshes a conversation's recap when turns have scrolled out
// of the replay window.
//
// It runs AFTER the reply has been streamed, on a detached context, so the
// person never waits for it, and a failure is logged and forgotten rather than
// failing a turn that already succeeded.
func (s *Service) maybeSummarize(actor *Actor, conv *models.AssistantConversation) {
	if s.deps.LLM == nil || conv == nil {
		return
	}
	// Everything up to this sequence number is outside the replayed window.
	outside := conv.MessageCount - s.deps.Cfg.MaxContextMsgs
	if outside <= conv.SummarizedSeq {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		stored, err := s.deps.AssistantRepo.RecentMessages(ctx, conv.ID, actor.ID(), s.deps.Cfg.MaxConversationMsgs)
		if err != nil {
			return
		}
		var transcript strings.Builder
		var through int
		for _, m := range stored {
			if m.Seq > outside {
				break
			}
			through = m.Seq
			transcript.WriteString(renderForRecap(m))
		}
		if through == 0 || transcript.Len() == 0 {
			return
		}
		if conv.Summary != "" {
			transcript.WriteString("\n[recap so far] " + conv.Summary + "\n")
		}

		resp, err := s.deps.LLM.Complete(ctx, llm.Request{
			System: summaryPrompt,
			Messages: []llm.Message{{
				Role:    llm.RoleUser,
				Content: []llm.ContentBlock{llm.TextBlock(truncateRunes(transcript.String(), 6000))},
			}},
			MaxTokens: 220,
		})
		if err != nil {
			log.Printf("assistant: summarise conversation: %v", err)
			return
		}
		var recap strings.Builder
		for _, b := range resp.Content {
			if b.Type == llm.BlockText {
				recap.WriteString(b.Text)
			}
		}
		text := strings.TrimSpace(recap.String())
		if text == "" {
			return
		}
		if err := s.deps.AssistantRepo.SaveSummary(ctx, conv.ID, actor.ID(), text, through); err != nil {
			log.Printf("assistant: save conversation summary: %v", err)
		}
	}()
}

// renderForRecap flattens one stored message to plain text. Tool results
// contribute only their shape, never their full payload: a recap must not
// become a second, unaudited copy of employee data.
func renderForRecap(m models.AssistantMessage) string {
	var blocks []llm.ContentBlock
	if err := json.Unmarshal(m.Content, &blocks); err != nil {
		return ""
	}
	var b strings.Builder
	for _, blk := range blocks {
		switch blk.Type {
		case llm.BlockText:
			if blk.Text != "" {
				fmt.Fprintf(&b, "%s: %s\n", m.Role, truncateRunes(blk.Text, 600))
			}
		case llm.BlockToolUse:
			fmt.Fprintf(&b, "assistant looked up: %s\n", blk.Name)
		case llm.BlockToolResult:
			fmt.Fprintf(&b, "result: %s\n", truncateRunes(blk.Content, 300))
		}
	}
	return b.String()
}

// truncateRunes cuts on a rune boundary so Arabic text never ends mid-codepoint.
func truncateRunes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	r := []rune(s)
	// Byte length is the cap that matters; step back until the prefix fits.
	for len(string(r)) > max && len(r) > 0 {
		r = r[:len(r)*max/len(string(r))+1]
	}
	return string(r) + "…"
}
