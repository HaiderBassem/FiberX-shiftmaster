package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// ErrActionNotPending reports a decision attempted on an action that is not in
// a decidable state — already decided, expired, superseded, or owned by
// someone else (ownership failures are indistinguishable by design).
var ErrActionNotPending = errors.New("action is not pending")

// AssistantRepository persists conversations, pending actions and the request
// log for the AI assistant. Every query that touches a conversation or action
// carries the employee ID in its WHERE clause: ownership is enforced in SQL,
// not by the politeness of callers.
type AssistantRepository interface {
	CreateConversation(ctx context.Context, employeeID uuid.UUID, role string, departmentID *uuid.UUID) (*models.AssistantConversation, error)
	GetConversation(ctx context.Context, id, employeeID uuid.UUID) (*models.AssistantConversation, error)
	// AppendMessages reserves sequence numbers and stores the turns atomically.
	AppendMessages(ctx context.Context, conversationID, employeeID uuid.UUID, roles []string, contents []json.RawMessage) error
	// RecentMessages returns the last `limit` messages in ascending order.
	RecentMessages(ctx context.Context, conversationID, employeeID uuid.UUID, limit int) ([]models.AssistantMessage, error)

	CreatePendingAction(ctx context.Context, a *models.AssistantPendingAction) error
	// SupersedePending marks every other pending action in the conversation as
	// superseded, so at most one approval card is live per dialogue.
	SupersedePending(ctx context.Context, conversationID, employeeID, exceptID uuid.UUID) error
	GetPendingAction(ctx context.Context, id, employeeID uuid.UUID) (*models.AssistantPendingAction, error)
	// ClaimForExecution atomically moves pending → approved. Exactly one caller
	// can win regardless of replicas or double clicks; losers get
	// ErrActionNotPending.
	ClaimForExecution(ctx context.Context, id, employeeID uuid.UUID) (*models.AssistantPendingAction, error)
	// FinishExecution records the outcome of a claimed action.
	FinishExecution(ctx context.Context, id uuid.UUID, status string, result json.RawMessage, execErr string) error
	// Reject atomically moves pending → rejected.
	Reject(ctx context.Context, id, employeeID uuid.UUID) error
	// ExpireStale lazily flips pending actions past their deadline to expired.
	ExpireStale(ctx context.Context, employeeID uuid.UUID) error

	LogRequest(ctx context.Context, r *models.AssistantRequestLog) error
}

type assistantRepo struct {
	db *database.DB
}

func NewAssistantRepository(db *database.DB) AssistantRepository {
	return &assistantRepo{db: db}
}

func (r *assistantRepo) CreateConversation(ctx context.Context, employeeID uuid.UUID, role string, departmentID *uuid.UUID) (*models.AssistantConversation, error) {
	c := &models.AssistantConversation{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO assistant_conversations (employee_id, role, department_id)
		 VALUES ($1, $2, $3)
		 RETURNING id, employee_id, role, department_id, message_count, created_at, updated_at`,
		employeeID, role, departmentID,
	).Scan(&c.ID, &c.EmployeeID, &c.Role, &c.DepartmentID, &c.MessageCount, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return c, nil
}

func (r *assistantRepo) GetConversation(ctx context.Context, id, employeeID uuid.UUID) (*models.AssistantConversation, error) {
	c := &models.AssistantConversation{}
	err := r.db.QueryRow(ctx,
		`SELECT id, employee_id, role, department_id, message_count, created_at, updated_at
		 FROM assistant_conversations
		 WHERE id = $1 AND employee_id = $2`,
		id, employeeID,
	).Scan(&c.ID, &c.EmployeeID, &c.Role, &c.DepartmentID, &c.MessageCount, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return c, nil
}

func (r *assistantRepo) AppendMessages(ctx context.Context, conversationID, employeeID uuid.UUID, roles []string, contents []json.RawMessage) error {
	if len(roles) != len(contents) {
		return fmt.Errorf("append messages: %d roles for %d contents", len(roles), len(contents))
	}
	if len(roles) == 0 {
		return nil
	}
	return r.db.ExecTx(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		// Reserve a contiguous sequence range under the row lock the UPDATE
		// takes, so concurrent turns on the same conversation cannot collide.
		// The employee predicate makes appending to someone else's
		// conversation impossible even if a caller skipped GetConversation.
		var base int
		err := tx.QueryRow(txCtx,
			`UPDATE assistant_conversations
			 SET message_count = message_count + $3, updated_at = now()
			 WHERE id = $1 AND employee_id = $2
			 RETURNING message_count - $3`,
			conversationID, employeeID, len(roles),
		).Scan(&base)
		if err != nil {
			return fmt.Errorf("reserve message sequence: %w", err)
		}
		for i := range roles {
			if _, err := tx.Exec(txCtx,
				`INSERT INTO assistant_messages (conversation_id, seq, role, content)
				 VALUES ($1, $2, $3, $4)`,
				conversationID, base+i+1, roles[i], contents[i],
			); err != nil {
				return fmt.Errorf("insert message: %w", err)
			}
		}
		return nil
	})
}

func (r *assistantRepo) RecentMessages(ctx context.Context, conversationID, employeeID uuid.UUID, limit int) ([]models.AssistantMessage, error) {
	rows, err := r.db.Query(ctx,
		`SELECT m.id, m.conversation_id, m.seq, m.role, m.content, m.created_at
		 FROM assistant_messages m
		 JOIN assistant_conversations c ON c.id = m.conversation_id
		 WHERE m.conversation_id = $1 AND c.employee_id = $2
		 ORDER BY m.seq DESC
		 LIMIT $3`,
		conversationID, employeeID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent messages: %w", err)
	}
	defer rows.Close()

	var out []models.AssistantMessage
	for rows.Next() {
		var m models.AssistantMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Seq, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Reverse into ascending order.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (r *assistantRepo) CreatePendingAction(ctx context.Context, a *models.AssistantPendingAction) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO assistant_pending_actions
		     (employee_id, conversation_id, action_type, params, summary, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, status, created_at`,
		a.EmployeeID, a.ConversationID, a.ActionType, a.Params, a.Summary, a.ExpiresAt,
	).Scan(&a.ID, &a.Status, &a.CreatedAt)
	if err != nil {
		return fmt.Errorf("create pending action: %w", err)
	}
	return nil
}

func (r *assistantRepo) SupersedePending(ctx context.Context, conversationID, employeeID, exceptID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE assistant_pending_actions
		 SET status = 'superseded', decided_at = now()
		 WHERE conversation_id = $1 AND employee_id = $2 AND id <> $3 AND status = 'pending'`,
		conversationID, employeeID, exceptID,
	)
	if err != nil {
		return fmt.Errorf("supersede pending actions: %w", err)
	}
	return nil
}

const pendingActionColumns = `id, employee_id, conversation_id, action_type, params, summary,
	 status, result, error, created_at, expires_at, decided_at`

func scanPendingAction(row pgx.Row) (*models.AssistantPendingAction, error) {
	a := &models.AssistantPendingAction{}
	err := row.Scan(&a.ID, &a.EmployeeID, &a.ConversationID, &a.ActionType, &a.Params, &a.Summary,
		&a.Status, &a.Result, &a.Error, &a.CreatedAt, &a.ExpiresAt, &a.DecidedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (r *assistantRepo) GetPendingAction(ctx context.Context, id, employeeID uuid.UUID) (*models.AssistantPendingAction, error) {
	a, err := scanPendingAction(r.db.QueryRow(ctx,
		`SELECT `+pendingActionColumns+`
		 FROM assistant_pending_actions
		 WHERE id = $1 AND employee_id = $2`,
		id, employeeID,
	))
	if err != nil {
		return nil, fmt.Errorf("get pending action: %w", err)
	}
	return a, nil
}

func (r *assistantRepo) ClaimForExecution(ctx context.Context, id, employeeID uuid.UUID) (*models.AssistantPendingAction, error) {
	// The compare-and-set on status is the whole security story here: only a
	// row still 'pending', still unexpired, and owned by this employee can be
	// claimed, and Postgres guarantees a single winner. A stale, replayed,
	// double-clicked or cross-user approval all fall into ErrActionNotPending.
	a, err := scanPendingAction(r.db.QueryRow(ctx,
		`UPDATE assistant_pending_actions
		 SET status = 'approved', decided_at = now()
		 WHERE id = $1 AND employee_id = $2 AND status = 'pending' AND expires_at > now()
		 RETURNING `+pendingActionColumns,
		id, employeeID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrActionNotPending
		}
		return nil, fmt.Errorf("claim action: %w", err)
	}
	return a, nil
}

func (r *assistantRepo) FinishExecution(ctx context.Context, id uuid.UUID, status string, result json.RawMessage, execErr string) error {
	var errPtr *string
	if execErr != "" {
		errPtr = &execErr
	}
	tag, err := r.db.Exec(ctx,
		`UPDATE assistant_pending_actions
		 SET status = $2, result = $3, error = $4
		 WHERE id = $1 AND status = 'approved'`,
		id, status, result, errPtr,
	)
	if err != nil {
		return fmt.Errorf("finish action: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("finish action: not in approved state")
	}
	return nil
}

func (r *assistantRepo) Reject(ctx context.Context, id, employeeID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE assistant_pending_actions
		 SET status = 'rejected', decided_at = now()
		 WHERE id = $1 AND employee_id = $2 AND status = 'pending'`,
		id, employeeID,
	)
	if err != nil {
		return fmt.Errorf("reject action: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrActionNotPending
	}
	return nil
}

func (r *assistantRepo) ExpireStale(ctx context.Context, employeeID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE assistant_pending_actions
		 SET status = 'expired', decided_at = now()
		 WHERE employee_id = $1 AND status = 'pending' AND expires_at <= now()`,
		employeeID,
	)
	if err != nil {
		return fmt.Errorf("expire stale actions: %w", err)
	}
	return nil
}

func (r *assistantRepo) LogRequest(ctx context.Context, l *models.AssistantRequestLog) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO assistant_requests
		     (id, employee_id, conversation_id, model, tool_calls, input_chars,
		      model_ms, tools_ms, total_ms, input_tokens, output_tokens, status, error_kind)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NULLIF($13,''))`,
		l.ID, l.EmployeeID, l.ConversationID, l.Model, l.ToolCalls, l.InputChars,
		l.ModelMs, l.ToolsMs, l.TotalMs, l.InputTokens, l.OutputTokens, l.Status, l.ErrorKind,
	)
	if err != nil {
		return fmt.Errorf("log assistant request: %w", err)
	}
	return nil
}

var _ AssistantRepository = (*assistantRepo)(nil)
