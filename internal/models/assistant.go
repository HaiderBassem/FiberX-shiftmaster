package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AssistantConversation is one dialogue between an employee and the assistant.
// Role and DepartmentID snapshot the employee's authority when the dialogue
// began; the service refuses to continue after either changes, because stored
// tool results may describe data the employee could see then but not now.
type AssistantConversation struct {
	ID           uuid.UUID  `json:"id"`
	EmployeeID   uuid.UUID  `json:"employee_id"`
	Role         string     `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id"`
	MessageCount int        `json:"message_count"`
	// Summary is a short recap of turns that have scrolled out of the model's
	// context window, and SummarizedSeq is how far it reaches. Conversational
	// memory only — never a source of facts, and never the basis of an action.
	Summary       string    `json:"-"`
	SummarizedSeq int       `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AssistantMessage is one stored turn. Content is the provider-format block
// array; the models package treats it as opaque JSON so it does not depend on
// the provider types.
type AssistantMessage struct {
	ID             uuid.UUID       `json:"id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	Seq            int             `json:"seq"`
	Role           string          `json:"role"`
	Content        json.RawMessage `json:"content"`
	CreatedAt      time.Time       `json:"created_at"`
}

// Pending-action lifecycle. Exactly one path mutates state ('pending' →
// 'approved' → 'executed'/'failed'), guarded by compare-and-set updates.
const (
	ActionPending    = "pending"
	ActionApproved   = "approved" // claimed for execution; transient
	ActionRejected   = "rejected"
	ActionExpired    = "expired"
	ActionSuperseded = "superseded"
	ActionExecuted   = "executed"
	ActionFailed     = "failed"
)

// AssistantPendingAction is a validated, frozen state-changing operation
// awaiting an explicit human decision. Params is the complete input the
// executor will use; nothing from the model or the client is read at execution
// time.
type AssistantPendingAction struct {
	ID             uuid.UUID       `json:"id"`
	EmployeeID     uuid.UUID       `json:"employee_id"`
	ConversationID *uuid.UUID      `json:"conversation_id"`
	ActionType     string          `json:"action_type"`
	Params         json.RawMessage `json:"params"`
	Summary        json.RawMessage `json:"summary"`
	Status         string          `json:"status"`
	Result         json.RawMessage `json:"result,omitempty"`
	Error          *string         `json:"error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	ExpiresAt      time.Time       `json:"expires_at"`
	DecidedAt      *time.Time      `json:"decided_at,omitempty"`
}

// AssistantRequestLog is one chat turn's observability record. No message
// content is stored here.
type AssistantRequestLog struct {
	ID             uuid.UUID
	EmployeeID     uuid.UUID
	ConversationID *uuid.UUID
	Model          string
	ToolCalls      int
	InputChars     int
	ModelMs        int
	ToolsMs        int
	TotalMs        int
	InputTokens    int
	OutputTokens   int
	// Rounds is how many model↔tool iterations the turn took.
	Rounds int
	// ToolFree records that the model answered having explicitly decided the
	// turn needed no ShiftMaster data. Correct for small talk, wrong for a
	// question about real state — and indistinguishable in the reply itself,
	// which is why it is recorded here.
	ToolFree  bool
	Status    string
	ErrorKind string
}
