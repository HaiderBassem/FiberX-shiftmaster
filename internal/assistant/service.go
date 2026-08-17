package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/assistant/llm"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/temporal"
)

// Service is the assistant's entry point for the HTTP layer.
type Service struct {
	deps     *Deps
	registry *Registry
	limiter  *rateLimiter
}

func NewService(deps *Deps) *Service {
	if deps.Clock == nil {
		deps.Clock = temporal.SystemClock{}
	}
	return &Service{
		deps:     deps,
		registry: NewRegistry(AllTools()),
		limiter:  newRateLimiter(deps.Cfg.RequestsPerMinute),
	}
}

// Enabled reports whether a model is configured. Everything else in the
// application works without it.
func (s *Service) Enabled() bool { return s.deps.Cfg.Enabled() }

// Typed failures the handler maps to HTTP statuses.
var (
	ErrDisabled     = errors.New("assistant is not configured")
	ErrRateLimited  = errors.New("too many assistant requests; slow down")
	ErrBusy         = errors.New("a previous assistant request is still running")
	ErrInputTooLong = errors.New("message too long")
	ErrConversation = errors.New("conversation unavailable")
)

// ─── rate limiting ──────────────────────────────────────────────────────────

// rateLimiter is a per-employee token bucket plus a single-flight guard: one
// in-flight chat turn per employee, N turns per minute. State is per-replica
// by design — it protects cost and the provider, not correctness; everything
// correctness-critical (conversations, pending actions) lives in Postgres.
type rateLimiter struct {
	mu        sync.Mutex
	perMinute int
	buckets   map[uuid.UUID]*bucket
	inFlight  map[uuid.UUID]bool
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newRateLimiter(perMinute int) *rateLimiter {
	return &rateLimiter{
		perMinute: perMinute,
		buckets:   map[uuid.UUID]*bucket{},
		inFlight:  map[uuid.UUID]bool{},
	}
}

// acquire reserves one turn; the returned release must be called when done.
func (r *rateLimiter) acquire(id uuid.UUID) (func(), error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.inFlight[id] {
		return nil, ErrBusy
	}
	b, ok := r.buckets[id]
	now := time.Now()
	if !ok {
		b = &bucket{tokens: float64(r.perMinute), last: now}
		r.buckets[id] = b
	}
	b.tokens += now.Sub(b.last).Minutes() * float64(r.perMinute)
	if max := float64(r.perMinute); b.tokens > max {
		b.tokens = max
	}
	b.last = now
	if b.tokens < 1 {
		return nil, ErrRateLimited
	}
	b.tokens--
	r.inFlight[id] = true

	// Opportunistic cleanup so the map does not grow with the workforce.
	if len(r.buckets) > 4096 {
		for k, v := range r.buckets {
			if now.Sub(v.last) > time.Hour {
				delete(r.buckets, k)
			}
		}
	}

	return func() {
		r.mu.Lock()
		delete(r.inFlight, id)
		r.mu.Unlock()
	}, nil
}

// ─── chat ───────────────────────────────────────────────────────────────────

// ChatResult reports how a turn ended, for the handler's final event.
type ChatResult struct {
	ConversationID uuid.UUID
}

// Begin validates a chat request and reserves the caller's rate-limit slot
// BEFORE any streaming starts, so quota and validation failures can be
// ordinary HTTP statuses instead of mid-stream error events. The returned
// release must be deferred by the caller.
func (s *Service) Begin(employeeID uuid.UUID, text string) (func(), error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: empty message", ErrInputTooLong)
	}
	if len(trimmed) > s.deps.Cfg.MaxInputChars {
		return nil, ErrInputTooLong
	}
	return s.limiter.acquire(employeeID)
}

// HandleMessage runs one conversation turn after a successful Begin. Events
// stream through emit; the method returns after the turn is fully persisted.
func (s *Service) HandleMessage(ctx context.Context, employeeID uuid.UUID, conversationID *uuid.UUID, text string, emit Emit) (*ChatResult, error) {
	text = strings.TrimSpace(text)

	started := time.Now()
	actor, err := s.deps.LoadActor(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	conv, err := s.resolveConversation(ctx, actor, conversationID)
	if err != nil {
		return nil, err
	}
	ctx = withConversation(ctx, conv.ID)

	history, err := s.loadHistory(ctx, conv, actor)
	if err != nil {
		return nil, err
	}

	// Persist the user turn first: even if the model call fails, the
	// transcript reflects what the user said.
	userContent, _ := json.Marshal([]llm.ContentBlock{llm.TextBlock(text)})
	if err := s.deps.AssistantRepo.AppendMessages(ctx, conv.ID, actor.ID(),
		[]string{"user"}, []json.RawMessage{userContent}); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversation, err)
	}
	history = append(history, llm.Message{Role: llm.RoleUser, Content: []llm.ContentBlock{llm.TextBlock(text)}})

	appended, stats, turnErr := s.deps.runTurn(ctx, actor, s.registry, history, emit)

	// Persist whatever the turn produced, even a partial tool exchange —
	// the transcript must stay coherent for the next turn.
	if len(appended) > 0 {
		roles := make([]string, len(appended))
		contents := make([]json.RawMessage, len(appended))
		for i, m := range appended {
			roles[i] = m.Role
			raw, mErr := json.Marshal(m.Content)
			if mErr != nil {
				raw = json.RawMessage(`[]`)
			}
			contents[i] = raw
		}
		if err := s.deps.AssistantRepo.AppendMessages(ctx, conv.ID, actor.ID(), roles, contents); err != nil {
			log.Printf("assistant: persist turn: %v", err)
		}
	}

	status, errKind := "ok", ""
	if turnErr != nil {
		status = "error"
		switch {
		case errors.Is(turnErr, llm.ErrUnavailable):
			errKind = "model_unavailable"
		case errors.Is(turnErr, llm.ErrAuth):
			errKind = "model_auth"
		case errors.Is(turnErr, llm.ErrBadRequest):
			errKind = "model_bad_request"
		case errors.Is(turnErr, context.Canceled):
			errKind = "client_disconnected"
		default:
			errKind = "internal"
		}
	}
	s.logRequest(actor, conv, stats, len(text), int(time.Since(started).Milliseconds()), status, errKind)

	if turnErr != nil {
		return &ChatResult{ConversationID: conv.ID}, turnErr
	}
	return &ChatResult{ConversationID: conv.ID}, nil
}

// resolveConversation loads (with ownership enforced in SQL) or creates the
// conversation, refusing to continue one recorded under a different authority:
// stored tool results may describe data the employee could see then but not
// now, so a role or department change forces a fresh dialogue.
func (s *Service) resolveConversation(ctx context.Context, actor *Actor, conversationID *uuid.UUID) (*models.AssistantConversation, error) {
	if conversationID == nil {
		return s.deps.AssistantRepo.CreateConversation(ctx, actor.ID(), actor.Role(), actor.DeptID())
	}
	conv, err := s.deps.AssistantRepo.GetConversation(ctx, *conversationID, actor.ID())
	if err != nil {
		return nil, fmt.Errorf("%w: not found", ErrConversation)
	}
	sameDept := (conv.DepartmentID == nil && actor.DeptID() == nil) ||
		(conv.DepartmentID != nil && actor.DeptID() != nil && *conv.DepartmentID == *actor.DeptID())
	if conv.Role != actor.Role() || !sameDept {
		return nil, fmt.Errorf("%w: your role or department changed — start a new conversation", ErrConversation)
	}
	if conv.MessageCount >= s.deps.Cfg.MaxConversationMsgs {
		return nil, fmt.Errorf("%w: conversation is full — start a new one", ErrConversation)
	}
	return conv, nil
}

// loadHistory converts the stored transcript window into provider messages.
// The window must start at a clean boundary: a tool_result message whose
// tool_use was trimmed away would be rejected by the provider, so leading
// orphaned tool responses are dropped.
func (s *Service) loadHistory(ctx context.Context, conv *models.AssistantConversation, actor *Actor) ([]llm.Message, error) {
	stored, err := s.deps.AssistantRepo.RecentMessages(ctx, conv.ID, actor.ID(), s.deps.Cfg.MaxContextMsgs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversation, err)
	}
	var out []llm.Message
	for _, m := range stored {
		var content []llm.ContentBlock
		if err := json.Unmarshal(m.Content, &content); err != nil || len(content) == 0 {
			continue
		}
		if len(out) == 0 {
			if m.Role != "user" {
				continue
			}
			if content[0].Type == llm.BlockToolResult {
				continue
			}
		}
		out = append(out, llm.Message{Role: m.Role, Content: content})
	}
	return out, nil
}

func (s *Service) logRequest(actor *Actor, conv *models.AssistantConversation, stats *turnStats, inputChars, totalMs int, status, errKind string) {
	entry := &models.AssistantRequestLog{
		ID:         uuid.New(),
		EmployeeID: actor.ID(),
		Model:      s.deps.Cfg.Model,
		InputChars: inputChars,
		TotalMs:    totalMs,
		Status:     status,
		ErrorKind:  errKind,
	}
	if conv != nil {
		id := conv.ID
		entry.ConversationID = &id
	}
	if stats != nil {
		entry.ToolCalls = stats.ToolCalls
		entry.ModelMs = stats.ModelMs
		entry.ToolsMs = stats.ToolsMs
		entry.InputTokens = stats.InputTokens
		entry.OutputTokens = stats.OutputTokens
	}
	// Best effort on a fresh context: observability must not fail the turn,
	// and the turn's context may already be cancelled by a disconnect.
	logCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.deps.AssistantRepo.LogRequest(logCtx, entry); err != nil {
		log.Printf("assistant: request log: %v", err)
	}
}

// ─── approval / rejection ───────────────────────────────────────────────────

// Decide resolves a pending action. Approval claims the row atomically
// (exactly one winner across replicas and repeated clicks), re-validates via
// the executor against live state, executes, and records the outcome; every
// non-claimable state — expired, already decided, superseded, someone else's —
// surfaces as ErrActionNotDecidable with the current status.
type DecideOutcome struct {
	Action *models.AssistantPendingAction
	// Result is present when execution succeeded.
	Result map[string]any
	// FailureReason is present when execution was attempted and refused.
	FailureReason string
}

var ErrActionNotDecidable = errors.New("action cannot be decided")

func (s *Service) Decide(ctx context.Context, employeeID, actionID uuid.UUID, approve bool) (*DecideOutcome, error) {
	// Fresh authorization: logout cannot be checked server-side (JWTs are
	// stateless) but deactivation, lockout and role changes are — and the
	// executors re-check scope themselves.
	actor, err := s.deps.LoadActor(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	// Lazily expire overdue cards so their status reads truthfully.
	_ = s.deps.AssistantRepo.ExpireStale(ctx, actor.ID())

	if !approve {
		if err := s.deps.AssistantRepo.Reject(ctx, actionID, actor.ID()); err != nil {
			return s.notDecidable(ctx, actor, actionID, err)
		}
		action, _ := s.deps.AssistantRepo.GetPendingAction(ctx, actionID, actor.ID())
		s.appendDecisionNote(ctx, actor, action, "رفضتَ هذا الإجراء ولم يُنفَّذ. / You rejected this action; nothing was submitted.")
		return &DecideOutcome{Action: action}, nil
	}

	claimed, err := s.deps.AssistantRepo.ClaimForExecution(ctx, actionID, actor.ID())
	if err != nil {
		return s.notDecidable(ctx, actor, actionID, err)
	}

	def, ok := actionDefs()[claimed.ActionType]
	if !ok {
		_ = s.deps.AssistantRepo.FinishExecution(ctx, claimed.ID, models.ActionFailed, nil, "unknown action type")
		return nil, fmt.Errorf("unknown action type %s", claimed.ActionType)
	}

	result, execErr := def.execute(ctx, s.deps, actor, claimed.Params)
	if execErr != nil {
		reason := asUserMessage(claimed.ActionType, execErr)
		_ = s.deps.AssistantRepo.FinishExecution(ctx, claimed.ID, models.ActionFailed, nil, reason)
		action, _ := s.deps.AssistantRepo.GetPendingAction(ctx, actionID, actor.ID())
		s.appendDecisionNote(ctx, actor, action, "تعذّر تنفيذ الإجراء بعد الموافقة: "+reason)
		s.audit(ctx, actor, claimed, "failed", reason)
		return &DecideOutcome{Action: action, FailureReason: reason}, nil
	}

	resultJSON, _ := json.Marshal(result)
	if err := s.deps.AssistantRepo.FinishExecution(ctx, claimed.ID, models.ActionExecuted, resultJSON, ""); err != nil {
		log.Printf("assistant: record execution result: %v", err)
	}
	action, _ := s.deps.AssistantRepo.GetPendingAction(ctx, actionID, actor.ID())
	s.appendDecisionNote(ctx, actor, action, "✅ وافقتَ على الإجراء وتم تنفيذه. / You approved this action and it was executed.")
	s.audit(ctx, actor, claimed, "executed", "")
	return &DecideOutcome{Action: action, Result: result}, nil
}

// notDecidable converts a failed transition into an outcome the UI can
// explain: it re-reads the row (ownership enforced) to report its real state.
func (s *Service) notDecidable(ctx context.Context, actor *Actor, actionID uuid.UUID, cause error) (*DecideOutcome, error) {
	action, err := s.deps.AssistantRepo.GetPendingAction(ctx, actionID, actor.ID())
	if err != nil {
		// Not this employee's action (or nonexistent): reveal nothing more.
		return nil, ErrActionNotDecidable
	}
	_ = cause
	return &DecideOutcome{Action: action}, ErrActionNotDecidable
}

// appendDecisionNote writes a server-authored line into the transcript so the
// next model turn knows what actually happened — the model itself never
// observes the approval click directly.
func (s *Service) appendDecisionNote(ctx context.Context, actor *Actor, action *models.AssistantPendingAction, note string) {
	if action == nil || action.ConversationID == nil {
		return
	}
	var title struct {
		TitleAr string `json:"title_ar"`
		TitleEn string `json:"title_en"`
	}
	_ = json.Unmarshal(action.Summary, &title)
	line := fmt.Sprintf("[%s / %s] %s", title.TitleAr, title.TitleEn, note)
	content, _ := json.Marshal([]llm.ContentBlock{llm.TextBlock(line)})
	if err := s.deps.AssistantRepo.AppendMessages(ctx, *action.ConversationID, actor.ID(),
		[]string{"assistant"}, []json.RawMessage{content}); err != nil {
		log.Printf("assistant: append decision note: %v", err)
	}
}

// audit records executed/failed assistant actions in the application's
// existing audit trail, so "what did the assistant change" is answerable in
// the same place as every other change.
func (s *Service) audit(ctx context.Context, actor *Actor, action *models.AssistantPendingAction, outcome, detail string) {
	if s.deps.AuditLogRepo == nil {
		return
	}
	newData := fmt.Sprintf(`{"assistant_action":%q,"outcome":%q,"detail":%q}`, action.ActionType, outcome, detail)
	empID := actor.ID()
	recordID := action.ID
	entry := &models.AuditLog{
		EmployeeID: &empID,
		Action:     "assistant." + action.ActionType,
		TableName:  "assistant_pending_actions",
		RecordID:   &recordID,
		NewData:    &newData,
	}
	if err := s.deps.AuditLogRepo.Create(ctx, entry); err != nil {
		log.Printf("assistant: audit: %v", err)
	}
}

// PendingActionsFor lists the caller's currently pending cards (for UI
// reload), expiring stale ones first.
func (s *Service) PendingAction(ctx context.Context, employeeID, actionID uuid.UUID) (*models.AssistantPendingAction, error) {
	_ = s.deps.AssistantRepo.ExpireStale(ctx, employeeID)
	return s.deps.AssistantRepo.GetPendingAction(ctx, actionID, employeeID)
}
