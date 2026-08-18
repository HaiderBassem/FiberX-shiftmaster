package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests for the local provider run against a stub llama.cpp server. What is
// worth testing here is everything BETWEEN the orchestrator and the runtime:
// how a transcript containing tool calls is rendered, how a reply becomes
// content blocks, what happens when the runtime misbehaves, and the two
// behaviours that exist specifically because the model is small and local —
// the one-shot grounding retry and the reasoning-leak guard.

type stubServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []oaiRequest
	replies  []oaiResponse
	status   int
	body     string
}

func newStub(t *testing.T, replies ...oaiResponse) *stubServer {
	t.Helper()
	s := &stubServer{replies: replies, status: http.StatusOK}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		var req oaiRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		s.mu.Lock()
		s.requests = append(s.requests, req)
		n := len(s.requests)
		status, body := s.status, s.body
		var reply oaiResponse
		if n <= len(s.replies) {
			reply = s.replies[n-1]
		} else if len(s.replies) > 0 {
			reply = s.replies[len(s.replies)-1]
		}
		s.mu.Unlock()

		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reply)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *stubServer) request(i int) oaiRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[i]
}

func (s *stubServer) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func textReply(text string) oaiResponse {
	return oaiResponse{Choices: []oaiChoice{{
		Message:      oaiMessage{Role: "assistant", Content: text},
		FinishReason: "stop",
	}}}
}

func toolReply(name, args string) oaiResponse {
	return oaiResponse{Choices: []oaiChoice{{
		Message: oaiMessage{Role: "assistant", ToolCalls: []oaiToolCall{{
			ID: "call_1", Type: "function", Function: oaiFunctionCall{Name: name, Arguments: args},
		}}},
		FinishReason: "tool_calls",
	}}}
}

func localClient(base string, groundingRetry bool) *Local {
	return NewLocal(LocalConfig{
		BaseURL: base, Model: "test", MaxTokens: 128,
		Timeout: 5 * time.Second, GroundingRetry: groundingRetry,
	}, nil)
}

func sampleTools() []Tool {
	return []Tool{{
		Name:        "get_current_shift",
		Description: "resolve the current shift",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}}
}

func TestToolCallBecomesContentBlock(t *testing.T) {
	stub := newStub(t, toolReply("get_current_shift", `{"a":1}`))
	resp, err := localClient(stub.URL, false).Complete(context.Background(), Request{
		System:   "sys",
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
		Tools:    sampleTools(),
		Round:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StopReason != StopToolUse || len(resp.Content) != 1 {
		t.Fatalf("got %+v", resp)
	}
	block := resp.Content[0]
	if block.Type != BlockToolUse || block.Name != "get_current_shift" || string(block.Input) != `{"a":1}` {
		t.Fatalf("block = %+v", block)
	}
}

func TestNarrationIsDroppedWhenToolsAreCalled(t *testing.T) {
	// A model that both narrates and calls a tool has not answered yet; showing
	// the narration would put a half-formed thought on screen ahead of the
	// facts.
	reply := toolReply("get_current_shift", `{}`)
	reply.Choices[0].Message.Content = "Let me check that for you"
	stub := newStub(t, reply)

	resp, err := localClient(stub.URL, false).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
		Tools:    sampleTools(),
		Round:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range resp.Content {
		if b.Type == BlockText {
			t.Fatalf("narration leaked alongside a tool call: %q", b.Text)
		}
	}
}

func TestTranscriptRendersToolCallsNatively(t *testing.T) {
	// Past tool calls and their results must reach the model in the shape its
	// chat template was trained on, not as prose about what happened.
	stub := newStub(t, textReply("done"))
	history := []Message{
		{Role: RoleUser, Content: []ContentBlock{TextBlock("شنو شفتتي؟")}},
		{Role: RoleAssistant, Content: []ContentBlock{{
			Type: BlockToolUse, ID: "t1", Name: "get_current_shift", Input: json.RawMessage(`{}`),
		}}},
		{Role: RoleUser, Content: []ContentBlock{ToolResultBlock("t1", `{"end":"00:30"}`, false)}},
	}
	if _, err := localClient(stub.URL, false).Complete(context.Background(), Request{
		System: "sys", Messages: history, Round: 1,
	}); err != nil {
		t.Fatal(err)
	}

	msgs := stub.request(0).Messages
	if msgs[0].Role != "system" || msgs[1].Role != "user" {
		t.Fatalf("unexpected prefix: %+v", msgs[:2])
	}
	if msgs[2].Role != "assistant" || len(msgs[2].ToolCalls) != 1 || msgs[2].ToolCalls[0].Function.Name != "get_current_shift" {
		t.Fatalf("tool call not rendered natively: %+v", msgs[2])
	}
	if msgs[3].Role != "tool" || msgs[3].ToolCallID != "t1" || !strings.Contains(msgs[3].Content, "00:30") {
		t.Fatalf("tool result not rendered natively: %+v", msgs[3])
	}
}

func TestToolChoiceIsAlwaysAuto(t *testing.T) {
	// tool_choice "required" is never sent. Measured against llama.cpp it does
	// not force a tool call, and it suppresses the stop token, so a turn that
	// wanted a one-line answer runs to the token limit repeating itself. The
	// escape hatch must still be on offer so "no data needed" stays an explicit,
	// recorded choice rather than a silent one.
	stub := newStub(t, toolReply("get_current_shift", `{}`))
	client := localClient(stub.URL, true)

	if _, err := client.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
		Tools:    sampleTools(),
		Round:    0,
	}); err != nil {
		t.Fatal(err)
	}
	req := stub.request(0)
	if req.ToolChoice != "auto" {
		t.Fatalf("tool_choice = %q, want auto", req.ToolChoice)
	}
	var sawEscape bool
	for _, tool := range req.Tools {
		if tool.Function.Name == NoToolNeeded {
			sawEscape = true
		}
	}
	if !sawEscape {
		t.Fatalf("catalogue did not offer %s", NoToolNeeded)
	}
}

func TestUngroundedOpeningIsReconsideredOnce(t *testing.T) {
	// A turn that opens with prose while tools were available gets exactly one
	// nudge. If the model then finds the lookup it should have made, that wins.
	stub := newStub(t,
		textReply("Your shift is 8 to 4."),
		toolReply("get_current_shift", `{}`),
	)
	resp, err := localClient(stub.URL, true).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("what's my shift?")}}},
		Tools:    sampleTools(),
		Round:    0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StopReason != StopToolUse {
		t.Fatalf("reconsidered reply = %+v", resp)
	}
	if stub.count() != 2 {
		t.Fatalf("made %d calls, want exactly 2", stub.count())
	}
	if !strings.Contains(stub.request(1).Messages[len(stub.request(1).Messages)-1].Content, "without looking anything up") {
		t.Fatalf("nudge missing from the retry")
	}
}

func TestModelMayStandByAnUngroundedAnswer(t *testing.T) {
	// If it reconsiders and still answers directly, the ORIGINAL wording is
	// kept — a reply written under the nudge reads like a machine explaining
	// itself — and the turn is marked tool-free so evaluation can see it.
	stub := newStub(t,
		textReply("هلا بيك! شلون أگدر أساعدك؟"),
		textReply("As I said, hello."),
	)
	resp, err := localClient(stub.URL, true).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("هلا")}}},
		Tools:    sampleTools(),
		Round:    0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ToolFree {
		t.Fatalf("tool-free turn was not marked")
	}
	if got := resp.Content[0].Text; !strings.Contains(got, "هلا بيك") {
		t.Fatalf("kept the nudged reply instead of the original: %q", got)
	}
}

func TestLaterRoundsAreNotReconsidered(t *testing.T) {
	// The nudge belongs to the opening move only; a later round answering from
	// tool results it already has is exactly what should happen.
	stub := newStub(t, textReply("دوامك ينتهي 00:30"))
	if _, err := localClient(stub.URL, true).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
		Tools:    sampleTools(),
		Round:    2,
	}); err != nil {
		t.Fatal(err)
	}
	if stub.count() != 1 {
		t.Fatalf("later round made %d calls, want 1", stub.count())
	}
}

func TestLaterRoundsOfferTheSameToolList(t *testing.T) {
	// The chat template renders the tool list into the prompt prefix, so a list
	// that changes between rounds throws away the runtime's KV cache and makes
	// every later round re-process the whole conversation.
	stub := newStub(t, toolReply("get_current_shift", `{}`))
	client := localClient(stub.URL, false)
	for _, round := range []int{0, 1} {
		if _, err := client.Complete(context.Background(), Request{
			Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
			Tools:    sampleTools(),
			Round:    round,
		}); err != nil {
			t.Fatal(err)
		}
	}
	first, second := stub.request(0), stub.request(1)
	if len(first.Tools) != len(second.Tools) {
		t.Fatalf("tool list changed between rounds: %d then %d", len(first.Tools), len(second.Tools))
	}
	if first.ToolChoice != "auto" || second.ToolChoice != "auto" {
		t.Fatalf("tool_choice = %q then %q, want auto both", first.ToolChoice, second.ToolChoice)
	}
}

func TestEscapeHatchProducesAPlainAnswer(t *testing.T) {
	// Choosing "no data needed" must not surface as a tool call to the
	// orchestrator; it becomes an ordinary answer, marked tool-free so the
	// evaluation suite can see it happened.
	stub := newStub(t,
		toolReply(NoToolNeeded, `{"reason":"greeting_or_small_talk"}`),
		textReply("هلا بيك! شلون أگدر أساعدك؟"),
	)
	resp, err := localClient(stub.URL, true).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("هلا")}}},
		Tools:    sampleTools(),
		Round:    0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StopReason != StopEndTurn {
		t.Fatalf("stop reason = %s", resp.StopReason)
	}
	if !resp.ToolFree {
		t.Fatalf("tool-free answer not marked as such")
	}
	if len(resp.Content) != 1 || resp.Content[0].Type != BlockText {
		t.Fatalf("content = %+v", resp.Content)
	}
	if stub.request(1).ToolChoice != "none" {
		t.Fatalf("answer call still offered tools: %q", stub.request(1).ToolChoice)
	}
}

func TestReasoningIsNeverSurfaced(t *testing.T) {
	// The runtime is launched with reasoning disabled; this is the second
	// defence, because private reasoning reaching a user is not recoverable.
	stub := newStub(t, textReply("<think>the user wants their shift; I should check</think>دوامك ينتهي 12:30"))
	resp, err := localClient(stub.URL, false).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
		Round:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := resp.Content[0].Text
	if strings.Contains(text, "<think>") || strings.Contains(text, "I should check") {
		t.Fatalf("reasoning leaked: %q", text)
	}
	if !strings.Contains(text, "12:30") {
		t.Fatalf("answer lost with the reasoning: %q", text)
	}
}

func TestUnterminatedReasoningIsDroppedEntirely(t *testing.T) {
	stub := newStub(t,
		textReply("<think>reasoning that never closes"),
		textReply("دوامك ينتهي 12:30"),
	)
	resp, err := localClient(stub.URL, false).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
		Round:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// The first reply is empty once the reasoning is stripped, so the provider
	// retries once rather than returning nothing.
	if len(resp.Content) == 0 || !strings.Contains(resp.Content[0].Text, "12:30") {
		t.Fatalf("empty reply was not recovered: %+v", resp.Content)
	}
	if stub.count() != 2 {
		t.Fatalf("expected exactly one retry, got %d calls", stub.count())
	}
}

func TestMalformedArgumentsReachTheToolLayerAsData(t *testing.T) {
	// The runtime's grammar makes this nearly impossible, but a corrupt reply
	// must degrade into a tool error the model can read, never a crash.
	stub := newStub(t, toolReply("get_current_shift", `{not json`))
	resp, err := localClient(stub.URL, false).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
		Tools:    sampleTools(),
		Round:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StopReason != StopToolUse {
		t.Fatalf("stop reason = %s", resp.StopReason)
	}
	if !json.Valid(resp.Content[0].Input) {
		t.Fatalf("invalid JSON was passed through: %s", resp.Content[0].Input)
	}
}

func TestRuntimeErrorsAreClassified(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusServiceUnavailable, ErrUnavailable}, // still loading
		{http.StatusUnauthorized, ErrAuth},              // wrong shared secret
		{http.StatusBadRequest, ErrBadRequest},          // our bug
		{http.StatusInternalServerError, ErrUnavailable},
	}
	for _, tc := range cases {
		stub := newStub(t, textReply("x"))
		stub.status = tc.status
		stub.body = `{"error":{"message":"boom"}}`
		_, err := localClient(stub.URL, false).Complete(context.Background(), Request{
			Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
			Round:    1,
		})
		if !errors.Is(err, tc.want) {
			t.Errorf("HTTP %d → %v, want %v", tc.status, err, tc.want)
		}
	}
}

func TestWarmupDoesNotRetry(t *testing.T) {
	// A warmup exists only to populate the prompt cache; spending retries
	// coaxing text out of it would double the cost of every startup.
	stub := newStub(t, textReply(""))
	_, err := localClient(stub.URL, false).Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("ok")}}},
		Round:    1,
		Warmup:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stub.count() != 1 {
		t.Fatalf("warmup made %d calls, want 1", stub.count())
	}
}

func TestGateIsAlwaysReleased(t *testing.T) {
	// Every call passes through the runtime's concurrency gate; a path that
	// forgets to release it leaks a slot and eventually stalls the assistant
	// for everyone.
	stub := newStub(t, textReply("ok"))
	gate := &countingGate{}
	client := NewLocal(LocalConfig{BaseURL: stub.URL, Model: "t", Timeout: 5 * time.Second}, gate)
	for i := 0; i < 3; i++ {
		_, _ = client.Complete(context.Background(), Request{
			Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
			Round:    1,
		})
	}
	if gate.entered != 3 || gate.released != 3 {
		t.Fatalf("entered %d, released %d", gate.entered, gate.released)
	}

	// A gate that refuses must abort the call, not proceed unguarded.
	before := stub.count()
	blocked := NewLocal(LocalConfig{BaseURL: stub.URL, Model: "t", Timeout: time.Second}, &countingGate{err: ErrBusy})
	if _, err := blocked.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("hi")}}},
	}); !errors.Is(err, ErrBusy) {
		t.Fatalf("blocked call returned %v", err)
	}
	if stub.count() != before {
		t.Fatalf("a refused call still reached the runtime")
	}
}

type countingGate struct {
	entered, released int
	err               error
}

func (g *countingGate) Enter(context.Context) (func(), error) {
	if g.err != nil {
		return nil, g.err
	}
	g.entered++
	return func() { g.released++ }, nil
}

func TestUnixSocketURLsAreRecognised(t *testing.T) {
	// A Unix socket is the stronger deployment: unreachable from another host
	// by construction, with file permissions deciding who may talk to it.
	client := localClient("unix:///run/shiftmaster/llm.sock", false)
	if got := client.endpoint("/v1/chat/completions"); got != "http://localhost/v1/chat/completions" {
		t.Fatalf("unix endpoint = %q", got)
	}
	if path, ok := unixSocketPath("unix:///run/x.sock"); !ok || path != "/run/x.sock" {
		t.Fatalf("socket path = %q %v", path, ok)
	}
	if _, ok := unixSocketPath("http://127.0.0.1:8081"); ok {
		t.Fatalf("http URL mistaken for a socket")
	}
}

func TestSplitHostPort(t *testing.T) {
	for _, tc := range []struct{ in, host, port string }{
		{"http://127.0.0.1:8081", "127.0.0.1", "8081"},
		{"http://localhost:9000/", "localhost", "9000"},
		{"127.0.0.1:8081", "127.0.0.1", "8081"},
	} {
		host, port := splitHostPort(tc.in)
		if host != tc.host || port != tc.port {
			t.Errorf("%q → %q %q, want %q %q", tc.in, host, port, tc.host, tc.port)
		}
	}
}
