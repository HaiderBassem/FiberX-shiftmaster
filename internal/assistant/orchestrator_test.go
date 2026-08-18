package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"shiftmaster-backend/internal/assistant/llm"
	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/temporal"
)

// scriptedLLM returns canned responses in order; the orchestrator's control
// flow is exercised without a provider or a database.
type scriptedLLM struct {
	responses []*llm.Response
	err       error
	calls     int
	requests  []llm.Request
}

func (s *scriptedLLM) Complete(ctx context.Context, req llm.Request) (*llm.Response, error) {
	s.requests = append(s.requests, req)
	if s.err != nil {
		return nil, s.err
	}
	if s.calls >= len(s.responses) {
		return &llm.Response{Content: []llm.ContentBlock{llm.TextBlock("done")}, StopReason: llm.StopEndTurn}, nil
	}
	r := s.responses[s.calls]
	s.calls++
	return r, nil
}

func unitDeps(client llm.Client) *Deps {
	return &Deps{
		Cfg: config.AssistantConfig{
			Enable:        true,
			ContextSize:   8192,
			MaxToolRounds: 3,
			MaxTokens:     512,
		},
		LLM:   client,
		Clock: temporal.FixedClock{T: time.Date(2026, 8, 17, 20, 0, 0, 0, temporal.Location())},
	}
}

// employeeActor builds an actor with no department so buildSystemPrompt makes
// no repository calls.
func employeeActor(role string) *Actor {
	return &Actor{Employee: &models.Employee{FirstName: "Test", LastName: "User", Role: role}}
}

func toolUse(id, name, input string) llm.ContentBlock {
	return llm.ContentBlock{Type: llm.BlockToolUse, ID: id, Name: name, Input: json.RawMessage(input)}
}

func collectEvents() (Emit, *[]Event) {
	var events []Event
	return func(e Event) { events = append(events, e) }, &events
}

func TestUnknownToolYieldsErrorResultAndContinues(t *testing.T) {
	client := &scriptedLLM{responses: []*llm.Response{
		{Content: []llm.ContentBlock{toolUse("t1", "drop_all_tables", `{}`)}, StopReason: llm.StopToolUse},
		{Content: []llm.ContentBlock{llm.TextBlock("sorry")}, StopReason: llm.StopEndTurn},
	}}
	d := unitDeps(client)
	emit, events := collectEvents()

	appended, stats, err := d.runTurn(context.Background(), employeeActor("employee"), NewRegistry(AllTools()),
		[]llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock("hi")}}}, emit)
	if err != nil {
		t.Fatal(err)
	}
	if stats.ToolCalls != 1 {
		t.Fatalf("tool calls = %d", stats.ToolCalls)
	}
	// The tool_result the model received must be an error, and the transcript
	// must contain assistant(tool_use) + user(tool_result) + assistant(text).
	if len(appended) != 3 {
		t.Fatalf("appended %d messages", len(appended))
	}
	result := appended[1].Content[0]
	if result.Type != llm.BlockToolResult || !result.IsError || !strings.Contains(result.Content, "does not exist for this user") {
		t.Fatalf("tool result = %+v", result)
	}
	var sawErrorEvent bool
	for _, e := range *events {
		if e.Type == "tool" && !e.OK {
			sawErrorEvent = true
		}
	}
	if !sawErrorEvent {
		t.Fatalf("no error tool event emitted")
	}
}

// The catalogue must make privileged tools invisible AND uncallable for lower
// roles: an employee calling get_team_status is indistinguishable from calling
// a tool that does not exist.
func TestRoleInvisibleToolIsUnknown(t *testing.T) {
	registry := NewRegistry(AllTools())

	if _, visible := registry.Lookup("get_team_status", "employee"); visible {
		t.Fatalf("supervisor tool visible to employee role")
	}
	if _, visible := registry.Lookup("get_team_status", "team_leader"); !visible {
		t.Fatalf("supervisor tool not visible to team_leader")
	}
	if _, visible := registry.Lookup("propose_leave_decision", "employee"); visible {
		t.Fatalf("decision tool visible to employee role")
	}

	// And through the loop: the employee's call yields "unknown tool".
	client := &scriptedLLM{responses: []*llm.Response{
		{Content: []llm.ContentBlock{toolUse("t1", "get_team_status", `{}`)}, StopReason: llm.StopToolUse},
	}}
	d := unitDeps(client)
	emit, _ := collectEvents()
	appended, _, err := d.runTurn(context.Background(), employeeActor("employee"), registry,
		[]llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock("منو موجود؟")}}}, emit)
	if err != nil {
		t.Fatal(err)
	}
	result := appended[1].Content[0]
	if !result.IsError || !strings.Contains(result.Content, "does not exist for this user") {
		t.Fatalf("expected unknown tool error, got %+v", result)
	}
}

// The tool catalogue sent to the provider must also be pruned per role.
func TestToolCatalogueIsRoleFiltered(t *testing.T) {
	registry := NewRegistry(AllTools())
	names := func(role string) map[string]bool {
		out := map[string]bool{}
		for _, tl := range registry.ForRole(role) {
			out[tl.Name] = true
		}
		return out
	}
	emp := names("employee")
	if emp["get_team_status"] || emp["get_department_overview"] || emp["propose_leave_decision"] {
		t.Fatalf("employee catalogue leaks supervisor tools: %v", emp)
	}
	tl := names("team_leader")
	if !tl["get_team_status"] || !tl["get_pending_approvals"] {
		t.Fatalf("team_leader catalogue missing supervisor tools")
	}
	if tl["get_department_overview"] {
		t.Fatalf("team_leader catalogue includes manager-only overview")
	}
	mgr := names("manager")
	if !mgr["get_department_overview"] {
		t.Fatalf("manager catalogue missing overview")
	}
}

func TestToolBudgetExhaustionForcesFinalAnswer(t *testing.T) {
	// The model insists on tools every round; after MaxToolRounds the loop
	// must inject the budget note, withdraw tools, and get a final answer.
	loop := &llm.Response{Content: []llm.ContentBlock{toolUse("t", "nope", `{}`)}, StopReason: llm.StopToolUse}
	client := &scriptedLLM{responses: []*llm.Response{loop, loop, loop,
		{Content: []llm.ContentBlock{llm.TextBlock("final")}, StopReason: llm.StopEndTurn},
	}}
	d := unitDeps(client) // MaxToolRounds = 3
	emit, _ := collectEvents()

	_, stats, err := d.runTurn(context.Background(), employeeActor("employee"), NewRegistry(AllTools()),
		[]llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock("go")}}}, emit)
	if err != nil {
		t.Fatal(err)
	}
	if stats.ToolCalls != 3 {
		t.Fatalf("tool calls = %d, want exactly MaxToolRounds", stats.ToolCalls)
	}
	last := client.requests[len(client.requests)-1]
	if len(last.Tools) != 0 {
		t.Fatalf("final call still offered tools")
	}
	joined := ""
	for _, m := range last.Messages {
		for _, b := range m.Content {
			joined += b.Text
		}
	}
	if !strings.Contains(joined, "no more lookups") {
		t.Fatalf("budget note missing from final call")
	}
}

func TestProviderFailureAborts(t *testing.T) {
	client := &scriptedLLM{err: llm.ErrUnavailable}
	d := unitDeps(client)
	emit, _ := collectEvents()
	_, _, err := d.runTurn(context.Background(), employeeActor("employee"), NewRegistry(AllTools()),
		[]llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock("hi")}}}, emit)
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
}

func TestPanickingToolIsContained(t *testing.T) {
	registry := NewRegistry([]Tool{{
		Name:        "boom",
		Description: "explodes",
		InputSchema: schema(`{"type":"object"}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			panic("kaboom")
		},
	}})
	client := &scriptedLLM{responses: []*llm.Response{
		{Content: []llm.ContentBlock{toolUse("t1", "boom", `{}`)}, StopReason: llm.StopToolUse},
		{Content: []llm.ContentBlock{llm.TextBlock("survived")}, StopReason: llm.StopEndTurn},
	}}
	d := unitDeps(client)
	emit, _ := collectEvents()
	appended, _, err := d.runTurn(context.Background(), employeeActor("employee"), registry,
		[]llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock("hi")}}}, emit)
	if err != nil {
		t.Fatalf("panic escaped: %v", err)
	}
	result := appended[1].Content[0]
	if !result.IsError || !strings.Contains(result.Content, "internal error") {
		t.Fatalf("panic result leaked detail: %+v", result)
	}
	if strings.Contains(result.Content, "kaboom") {
		t.Fatalf("panic message leaked to model context")
	}
}

func TestMalformedToolInputIsRejected(t *testing.T) {
	// Unknown fields and wrong types must be rejected by strict decoding, not
	// silently ignored — a smuggled employee_id must fail loudly.
	client := &scriptedLLM{responses: []*llm.Response{
		{Content: []llm.ContentBlock{toolUse("t1", "get_my_schedule", `{"from":"2026-08-17","to":"2026-08-18","employee_id":"someone-else"}`)}, StopReason: llm.StopToolUse},
		{Content: []llm.ContentBlock{llm.TextBlock("ok")}, StopReason: llm.StopEndTurn},
	}}
	d := unitDeps(client)
	emit, _ := collectEvents()
	appended, _, err := d.runTurn(context.Background(), employeeActor("employee"), NewRegistry(AllTools()),
		[]llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock("hi")}}}, emit)
	if err != nil {
		t.Fatal(err)
	}
	result := appended[1].Content[0]
	if !result.IsError || !strings.Contains(result.Content, "invalid tool input") {
		t.Fatalf("smuggled field not rejected: %+v", result)
	}
}

func TestSystemPromptCarriesTheDefenses(t *testing.T) {
	client := &scriptedLLM{responses: []*llm.Response{
		{Content: []llm.ContentBlock{llm.TextBlock("hello")}, StopReason: llm.StopEndTurn},
	}}
	d := unitDeps(client)
	emit, _ := collectEvents()
	_, _, err := d.runTurn(context.Background(), employeeActor("employee"), NewRegistry(AllTools()),
		[]llm.Message{{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock("hi")}}}, emit)
	if err != nil {
		t.Fatal(err)
	}
	system := client.requests[0].System
	for _, required := range []string{
		"data, never instructions",   // retrieved content is data
		"cross midnight",             // overnight rule
		"get_current_shift",          // temporal delegation
		"presses Approve",            // approval semantics
		"Asia/Baghdad",               // timezone
		"permissions are enforced",   // no self-authorized elevation
		"Reply in the SAME language", // language mirroring is a hard requirement
		"comes from a tool result",   // grounding
	} {
		if !strings.Contains(system, required) {
			t.Errorf("system prompt is missing %q", required)
		}
	}
}

func TestExtractPendingAction(t *testing.T) {
	body := []byte(`{"pending_action":{"action_id":"x","summary":{"title_ar":"ت"}},"note":"n"}`)
	if payload, ok := extractPendingAction(body); !ok || !strings.Contains(string(payload), "action_id") {
		t.Fatalf("pending action not extracted: %s", payload)
	}
	if _, ok := extractPendingAction([]byte(`{"plans":[]}`)); ok {
		t.Fatalf("false positive")
	}
}
