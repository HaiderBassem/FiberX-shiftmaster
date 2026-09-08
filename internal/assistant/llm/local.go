package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Local is the assistant's model provider: a llama.cpp server running on this
// host, reached over loopback or a Unix socket. Nothing it sends leaves the
// machine.
//
// Why the OpenAI-shaped endpoint rather than raw /completion: llama.cpp
// applies the model's OWN chat template there (--jinja), and when tools are
// supplied it compiles each tool's JSON Schema into a GBNF grammar and
// constrains sampling with it. That is the single most important property for
// driving a small local model reliably — an unknown tool name or an argument
// of the wrong type is not "usually avoided", it is unrepresentable in the
// token stream. What remains for us to police is semantics, which the tool
// layer does anyway.
//
// The provider therefore presents the same Client contract as any hosted
// tool-calling API, and the orchestrator above it does not know or care which
// one it is talking to.
type Local struct {
	cfg  LocalConfig
	http *http.Client
	gate Gate
}

// Gate is the concurrency/health guard a runtime supplies. A single loaded
// model is a fixed, expensive resource: every call must pass through here so
// that queueing, backpressure and cancellation are enforced in one place.
// Runtime implements it; tests pass nil.
type Gate interface {
	// Enter blocks until a slot is free or ctx expires. The returned release
	// must be called exactly once.
	Enter(ctx context.Context) (release func(), err error)
}

// LocalConfig is everything the provider needs. All of it comes from
// application configuration; no model path or runtime detail is ever surfaced
// to an end user.
type LocalConfig struct {
	// BaseURL is http://127.0.0.1:PORT or unix:///path/to/llm.sock.
	BaseURL string
	// Model is a human label used in logs and request records only.
	Model string
	// APIKey is an optional shared secret for the loopback listener. It is a
	// defence-in-depth measure against other local processes, not an external
	// credential.
	APIKey string

	Temperature float64
	TopP        float64
	MaxTokens   int

	// Timeout bounds one HTTP call to the runtime.
	Timeout    time.Duration
	MaxRetries int

	// RepeatPenalty and PresencePenalty bound repetition loops. Sensible
	// defaults are applied when they are left at zero.
	RepeatPenalty   float64
	PresencePenalty float64

	// GroundingRetry asks the model to reconsider once, under a grammar, when
	// its first reply of a turn answers from nothing while tools were
	// available. See the reconsideration block in Complete for what that costs
	// and why it is a second call rather than the first.
	//
	// The measurement that produced this design: asking in writing ("you
	// answered without looking anything up, call a tool") moved the local model
	// some of the time and not reliably — the same Arabic question about prices
	// was grounded on one run and invented on the next. Constraining the choice
	// instead of requesting it removed the variance, because a grammar is not
	// something the sampler can decline.
	//
	// A turn that calls a tool immediately — the common case — never pays for
	// any of this.
	GroundingRetry bool
}

// groundingRetryTokens caps the reconsideration.
//
// Under the tool grammar the model cannot emit a stop token by talking, so a
// turn where no tool genuinely fits will generate until something stops it.
// A tool call is short — a name and a small JSON object — so this is generous
// for the case that succeeds and cheap for the case that does not.
const groundingRetryTokens = 96

// NoToolNeeded is the pseudo-tool the model picks when a turn genuinely needs
// no system data. It never reaches the tool registry — the provider handles it
// and converts the turn into a plain answer.
const NoToolNeeded = "answer_without_tools"

var noToolNeededSchema = json.RawMessage(`{
  "type":"object",
  "properties":{
    "reason":{
      "type":"string",
      "enum":["greeting_or_small_talk","clarifying_question","explaining_how_something_works","facts_already_retrieved","out_of_scope"],
      "description":"why no system data is needed for this turn. 'clarifying_question' is only for an ACTION whose details are genuinely missing — never for a lookup: asking whether you should search is not a clarifying question, it is a search you did not run."
    }
  },
  "required":["reason"],
  "additionalProperties":false
}`)

func NewLocal(cfg LocalConfig, gate Gate) *Local {
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 768
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 90 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RepeatPenalty <= 0 {
		cfg.RepeatPenalty = 1.08
	}
	if cfg.PresencePenalty <= 0 {
		cfg.PresencePenalty = 0.4
	}
	return &Local{cfg: cfg, http: newLocalHTTPClient(cfg.BaseURL), gate: gate}
}

// newLocalHTTPClient dials TCP loopback or a Unix socket depending on the URL
// scheme. A Unix socket is the stronger deployment: it cannot be reached from
// another host even by misconfiguration, and file permissions decide who on
// the box may talk to the model.
func newLocalHTTPClient(baseURL string) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        4,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     120 * time.Second,
	}
	if socket, ok := unixSocketPath(baseURL); ok {
		transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}
	}
	// No client-level timeout: the per-call context carries the deadline, so a
	// caller with less time than cfg.Timeout still wins.
	return &http.Client{Transport: transport}
}

// unixSocketPath reports the socket path for unix:// URLs.
func unixSocketPath(baseURL string) (string, bool) {
	if strings.HasPrefix(baseURL, "unix://") {
		return strings.TrimPrefix(baseURL, "unix://"), true
	}
	return "", false
}

// endpoint builds an absolute URL. Unix-socket URLs get a synthetic host,
// which net/http requires but never resolves because DialContext is overridden.
func (l *Local) endpoint(path string) string {
	if _, ok := unixSocketPath(l.cfg.BaseURL); ok {
		return "http://localhost" + path
	}
	return strings.TrimRight(l.cfg.BaseURL, "/") + path
}

func (l *Local) Model() string { return l.cfg.Model }

var _ Client = (*Local)(nil)

// ─── wire format ────────────────────────────────────────────────────────────

type oaiFunctionCall struct {
	Name string `json:"name"`
	// Arguments is a JSON *string* in this protocol, not an object.
	Arguments string `json:"arguments"`
}

type oaiToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function oaiFunctionCall `json:"function"`
}

type oaiMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	ToolCalls  []oaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type oaiToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type oaiTool struct {
	Type     string          `json:"type"`
	Function oaiToolFunction `json:"function"`
}

type oaiRequest struct {
	Model       string       `json:"model,omitempty"`
	Messages    []oaiMessage `json:"messages"`
	Tools       []oaiTool    `json:"tools,omitempty"`
	ToolChoice  string       `json:"tool_choice,omitempty"`
	Temperature float64      `json:"temperature"`
	TopP        float64      `json:"top_p,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	// RepeatPenalty guards against a failure mode that is specific to small
	// models and very visible to users: locking into a loop and restating the
	// same sentence until the token budget runs out. llama.cpp leaves this at
	// 1.0 (off) by default. It is kept mild — Arabic repeats particles and
	// pronouns legitimately, and a heavy penalty degrades the prose.
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"`
	// PresencePenalty discourages re-opening a topic already covered, which is
	// the paragraph-level version of the same problem.
	PresencePenalty float64 `json:"presence_penalty,omitempty"`
	Stream          bool    `json:"stream"`
	// CachePrompt keeps the KV cache of the shared prefix between the rounds of
	// one turn; without it every round re-processes the whole transcript.
	CachePrompt bool `json:"cache_prompt"`
}

type oaiChoice struct {
	Message      oaiMessage `json:"message"`
	FinishReason string     `json:"finish_reason"`
}

type oaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type oaiTimings struct {
	PromptMs           float64 `json:"prompt_ms"`
	PredictedMs        float64 `json:"predicted_ms"`
	PredictedPerSecond float64 `json:"predicted_per_second"`
}

type oaiResponse struct {
	Choices []oaiChoice `json:"choices"`
	Usage   oaiUsage    `json:"usage"`
	Timings oaiTimings  `json:"timings"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ─── conversation translation ───────────────────────────────────────────────

// toWire converts the assistant's transcript into the message list the chat
// template expects. Tool calls and tool results are rendered in the model's
// NATIVE format (tool_calls / role:"tool"), which is what it was trained on —
// far more reliable than describing past calls in prose.
func toWire(system string, msgs []Message) []oaiMessage {
	out := make([]oaiMessage, 0, len(msgs)+1)
	if system != "" {
		out = append(out, oaiMessage{Role: "system", Content: system})
	}
	for _, m := range msgs {
		var text strings.Builder
		var calls []oaiToolCall
		var results []oaiMessage

		for _, block := range m.Content {
			switch block.Type {
			case BlockText:
				if block.Text != "" {
					if text.Len() > 0 {
						text.WriteString("\n")
					}
					text.WriteString(block.Text)
				}
			case BlockToolUse:
				args := string(block.Input)
				if args == "" {
					args = "{}"
				}
				calls = append(calls, oaiToolCall{
					ID:       block.ID,
					Type:     "function",
					Function: oaiFunctionCall{Name: block.Name, Arguments: args},
				})
			case BlockToolResult:
				results = append(results, oaiMessage{
					Role:       "tool",
					ToolCallID: block.ToolUseID,
					Content:    block.Content,
				})
			}
		}

		if m.Role == RoleAssistant {
			if text.Len() > 0 || len(calls) > 0 {
				out = append(out, oaiMessage{Role: "assistant", Content: text.String(), ToolCalls: calls})
			}
			continue
		}
		// User turns carry either the person's text or the tool results we
		// produced for the previous assistant message.
		if text.Len() > 0 {
			out = append(out, oaiMessage{Role: "user", Content: text.String()})
		}
		out = append(out, results...)
	}
	return out
}

// toWireTools renders the catalogue, appending the no-tool escape hatch so a
// forced choice always has a truthful option.
func toWireTools(tools []Tool, withEscape bool) []oaiTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]oaiTool, 0, len(tools)+1)
	for _, t := range tools {
		params := t.InputSchema
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, oaiTool{Type: "function", Function: oaiToolFunction{
			Name: t.Name, Description: t.Description, Parameters: params,
		}})
	}
	if withEscape {
		out = append(out, oaiTool{Type: "function", Function: oaiToolFunction{
			Name: NoToolNeeded,
			Description: "Choose this ONLY when answering needs no data from ShiftMaster: greetings and small talk, " +
				"asking the user a clarifying question, explaining how something works in general, or when the facts you need " +
				"are already in this conversation from earlier tool results. Never choose it for a question about a real " +
				"schedule, shift, task, leave balance, person, price or document — those always need a tool. " +
				// The Arabic is not decoration. Measured on the local model, an
				// Arabic question about prices or policy was far likelier to be
				// answered from nothing than the same question in English, and the
				// gap closed when the tool contracts named the Arabic words people
				// actually type. This one is the last line of defence, so it says
				// them too.
				"لا تختر هذا أبداً لسؤال عن سعر أو باقة أو خدمة أو سياسة أو مستند أو دوام أو إجازة أو مهمة — كل هذه تحتاج أداة.",
			Parameters: noToolNeededSchema,
		}})
	}
	return out
}

// ─── completion ─────────────────────────────────────────────────────────────

// Complete runs one model round. With tools present it may make two calls to
// the runtime: the forced choice, and — when the model declares no tool is
// needed — the plain-language answer. Both share a KV-cached prefix, so the
// second is generation-only.
func (l *Local) Complete(ctx context.Context, req Request) (*Response, error) {
	if l.gate != nil {
		release, err := l.gate.Enter(ctx)
		if err != nil {
			return nil, err
		}
		defer release()
	}

	messages := toWire(req.System, req.Messages)
	// The escape hatch is offered on every round: the tool list is rendered
	// into the prompt by the chat template, so changing it between rounds would
	// invalidate the KV cache of the shared prefix and make every round after
	// the first re-process the whole transcript.
	tools := toWireTools(req.Tools, len(req.Tools) > 0)

	choice := ""
	if len(tools) > 0 {
		choice = "auto"
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = l.cfg.MaxTokens
	}

	base := oaiRequest{
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  choice,
		Temperature: l.cfg.Temperature,
		TopP:        l.cfg.TopP,
		MaxTokens:   maxTokens,
		CachePrompt: true,
	}

	resp, err := l.chat(ctx, base)
	if err != nil {
		return nil, err
	}

	out := l.decode(resp)

	// One chance to reconsider an ungrounded opening answer.
	//
	// The same messages are sent again with the choice constrained to a tool.
	// Constrained is the operative word: under tool_choice "required" the
	// runtime compiles the tool schemas into a grammar and the sampler cannot
	// leave it, so this is not a request the model may talk its way out of the
	// way a written instruction is. Measured on the local model, the Arabic
	// questions that were answered from nothing — "شكد أرخص باقة عندكم؟",
	// "شنو سياسة التأخير عندنا؟" — came back as the right call with the right
	// Arabic argument every time under the constraint, and as prose without it.
	//
	// It is not used on the first call because the constraint also suppresses
	// the stop token: a turn that genuinely needs no lookup ("مرحبا شلونك")
	// cannot satisfy the grammar and runs to the token limit instead of
	// stopping. That is why this is a second call, why it is capped short, and
	// why prose coming back from it is read as "no tool fits" rather than as an
	// answer — the wasted tokens are discarded and the user never sees them.
	//
	// The messages and the tool list are byte-identical to the first call, so
	// the whole prefix is still in the KV cache and this costs generation only.
	if l.cfg.GroundingRetry && req.Round == 0 && len(tools) > 0 && !req.Warmup &&
		out.StopReason != StopToolUse {
		second := base
		second.ToolChoice = "required"
		second.MaxTokens = groundingRetryTokens
		if retry, retryErr := l.chat(ctx, second); retryErr == nil {
			reconsidered := l.decode(retry)
			out.Usage.InputTokens += reconsidered.Usage.InputTokens
			out.Usage.OutputTokens += reconsidered.Usage.OutputTokens
			if reconsidered.StopReason == StopToolUse && !declinedTools(reconsidered) {
				// It found a lookup it should have made; take it.
				reconsidered.Usage = out.Usage
				out = reconsidered
			} else {
				// Either it chose the escape hatch, or the grammar produced
				// nothing usable. Keep the ORIGINAL reply: it was written
				// without a constraint fighting the stop token, and it is the
				// one the person should read.
				out.ToolFree = true
			}
		}
	}

	// The model declared it needs nothing from the system: ask it for the
	// answer itself, with tools switched off so it cannot change its mind
	// mid-turn.
	if declinedTools(out) && !req.Warmup {
		answer, err := l.chat(ctx, oaiRequest{
			Messages:    messages,
			Tools:       tools,
			ToolChoice:  "none",
			Temperature: l.cfg.Temperature,
			TopP:        l.cfg.TopP,
			MaxTokens:   maxTokens,
			CachePrompt: true,
		})
		if err != nil {
			return nil, err
		}
		plain := l.decode(answer)
		plain.Usage.InputTokens += out.Usage.InputTokens
		plain.Usage.OutputTokens += out.Usage.OutputTokens
		plain.ToolFree = true
		out = plain
	}

	// An empty reply is a real failure mode for small models under a grammar.
	// One bounded retry with a nudge, then give up honestly.
	if len(out.Content) == 0 && !req.Warmup {
		nudged := append(append([]oaiMessage{}, messages...), oaiMessage{
			Role:    "user",
			Content: "[system] Your previous reply was empty. Answer the user's last message now, in their language, in one or two sentences.",
		})
		answer, err := l.chat(ctx, oaiRequest{
			Messages:    nudged,
			Temperature: l.cfg.Temperature,
			MaxTokens:   maxTokens,
			CachePrompt: true,
		})
		if err != nil {
			return nil, err
		}
		retry := l.decode(answer)
		retry.Usage.InputTokens += out.Usage.InputTokens
		retry.Usage.OutputTokens += out.Usage.OutputTokens
		out = retry
		if len(out.Content) == 0 {
			return nil, fmt.Errorf("%w: model returned no content", ErrUnavailable)
		}
	}

	return out, nil
}

// declinedTools reports the escape-hatch choice.
func declinedTools(r *Response) bool {
	for _, b := range r.Content {
		if b.Type == BlockToolUse && b.Name == NoToolNeeded {
			return true
		}
	}
	return false
}

// decode turns one runtime reply into the transcript's content blocks. Tool
// calls always win: a model that both narrates and calls is still calling.
func (l *Local) decode(resp *oaiResponse) *Response {
	out := &Response{
		StopReason: StopEndTurn,
		Usage:      Usage{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens},
		Timings: Timings{
			PromptMs:     resp.Timings.PromptMs,
			GenerationMs: resp.Timings.PredictedMs,
			TokensPerSec: resp.Timings.PredictedPerSecond,
		},
	}
	if len(resp.Choices) == 0 {
		return out
	}
	choice := resp.Choices[0]

	// Sanitise BEFORE the emptiness test: a reply consisting only of an
	// unterminated reasoning block is empty once the reasoning is removed, and
	// appending it anyway would put a blank bubble on screen and skip the
	// recovery path that would have produced a real answer.
	if text := sanitiseModelText(choice.Message.Content); text != "" {
		out.Content = append(out.Content, TextBlock(text))
	}

	for i, call := range choice.Message.ToolCalls {
		if call.Function.Name == "" {
			continue
		}
		id := call.ID
		if id == "" {
			id = fmt.Sprintf("call_%d", i)
		}
		args := strings.TrimSpace(call.Function.Arguments)
		if args == "" {
			args = "{}"
		}
		if !json.Valid([]byte(args)) {
			// The grammar makes this close to impossible, but a corrupt reply
			// must not crash the turn: hand the raw text to the tool layer,
			// which rejects it with a schema message the model can act on.
			log.Printf("assistant: model produced non-JSON arguments for %s", call.Function.Name)
			args = `{"__malformed__":true}`
		}
		out.Content = append(out.Content, ContentBlock{
			Type:  BlockToolUse,
			ID:    id,
			Name:  call.Function.Name,
			Input: json.RawMessage(args),
		})
		out.StopReason = StopToolUse
	}

	if out.StopReason != StopToolUse && choice.FinishReason == "length" {
		out.StopReason = StopMaxTokens
	}
	// A model that called a tool AND narrated is not answering yet; drop the
	// narration so the UI does not show a half-thought before the facts.
	if out.StopReason == StopToolUse {
		kept := out.Content[:0]
		for _, b := range out.Content {
			if b.Type != BlockText {
				kept = append(kept, b)
			}
		}
		out.Content = kept
	}
	return out
}

// sanitiseModelText strips reasoning scaffolding that a hybrid model can leak
// when its template is configured for visible thinking. Private reasoning must
// never reach the user; the runtime is also launched with reasoning disabled,
// so this is the second of two defences.
func sanitiseModelText(s string) string {
	for _, pair := range [][2]string{{"<think>", "</think>"}, {"<thinking>", "</thinking>"}, {"<reasoning>", "</reasoning>"}} {
		for {
			start := strings.Index(s, pair[0])
			if start < 0 {
				break
			}
			end := strings.Index(s[start:], pair[1])
			if end < 0 {
				// Unterminated: everything from the marker on is reasoning.
				s = s[:start]
				break
			}
			s = s[:start] + s[start+end+len(pair[1]):]
		}
	}
	return strings.TrimSpace(s)
}

// chat performs one HTTP call with bounded retries. A completion has no side
// effects — every state change in this system happens in the tool and approval
// layers — so retrying is safe by construction.
func (l *Local) chat(ctx context.Context, body oaiRequest) (*oaiResponse, error) {
	body.Model = l.cfg.Model
	body.Stream = false
	if body.RepeatPenalty == 0 {
		body.RepeatPenalty = l.cfg.RepeatPenalty
	}
	if body.PresencePenalty == 0 {
		body.PresencePenalty = l.cfg.PresencePenalty
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal: %v", ErrBadRequest, err)
	}

	var lastErr error
	for attempt := 0; attempt <= l.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
			case <-time.After(time.Duration(attempt) * 400 * time.Millisecond):
			}
		}
		resp, retryable, err := l.attempt(ctx, payload)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !retryable {
			return nil, err
		}
	}
	return nil, lastErr
}

func (l *Local) attempt(ctx context.Context, payload []byte) (*oaiResponse, bool, error) {
	callCtx, cancel := context.WithTimeout(ctx, l.cfg.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(callCtx, http.MethodPost,
		l.endpoint("/v1/chat/completions"), bytes.NewReader(payload))
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if l.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+l.cfg.APIKey)
	}

	httpResp, err := l.http.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, fmt.Errorf("%w: %v", ErrUnavailable, ctx.Err())
		}
		return nil, true, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 8<<20))
	if err != nil {
		return nil, true, fmt.Errorf("%w: read response: %v", ErrUnavailable, err)
	}

	if httpResp.StatusCode != http.StatusOK {
		retryable, cErr := l.classify(httpResp.StatusCode, respBody)
		return nil, retryable, cErr
	}

	var parsed oaiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, true, fmt.Errorf("%w: decode response: %v", ErrUnavailable, err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, false, fmt.Errorf("%w: %s", ErrBadRequest, truncate(parsed.Error.Message, 200))
	}
	return &parsed, false, nil
}

// classify maps a runtime failure onto the package taxonomy. Runtime detail is
// logged here and never propagated to a user-facing surface.
func (l *Local) classify(status int, body []byte) (bool, error) {
	var apiErr struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &apiErr)
	detail := truncate(apiErr.Error.Message, 300)
	if detail == "" {
		detail = http.StatusText(status)
	}

	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		// The loopback listener rejected our shared secret: an operator
		// configuration error, not a user-visible condition.
		return false, fmt.Errorf("%w: %s", ErrAuth, detail)
	case status == http.StatusServiceUnavailable || status == http.StatusTooManyRequests || status >= 500:
		// The model is still loading, or every slot is busy. Both clear up.
		return true, fmt.Errorf("%w: HTTP %d: %s", ErrUnavailable, status, detail)
	case status >= 400:
		log.Printf("assistant: local runtime rejected request (HTTP %d, type=%s): %s", status, apiErr.Error.Type, detail)
		return false, fmt.Errorf("%w: HTTP %d: %s", ErrBadRequest, status, detail)
	default:
		return true, fmt.Errorf("%w: HTTP %d: %s", ErrUnavailable, status, detail)
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// newReader avoids importing bytes in runtime.go for a single call.
func newReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
