package models

import (
	"time"

	"github.com/google/uuid"
)

type Handover struct {
	ID            uuid.UUID  `json:"id"`
	DepartmentID  uuid.UUID  `json:"department_id"`
	CreatorID     uuid.UUID  `json:"creator_id"`
	ShiftSummary  string     `json:"shift_summary"`
	PendingIssues string     `json:"pending_issues"`
	Status        string     `json:"status"` // open, claimed, completed
	ClaimedBy     *uuid.UUID `json:"claimed_by"`
	DoneBy        *uuid.UUID `json:"done_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Joined fields for UI
	CreatorName *string           `json:"creator_name,omitempty" db:"-"`
	ClaimerName *string           `json:"claimer_name,omitempty" db:"-"`
	DoneByName  *string           `json:"done_by_name,omitempty" db:"-"`
	Comments    []HandoverComment `json:"comments,omitempty" db:"-"`
}

type HandoverComment struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	AuthorName string    `json:"author_name"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}
