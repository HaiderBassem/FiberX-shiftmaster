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
	ClaimerNotes  *string    `json:"claimer_notes"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Joined fields for UI
	CreatorName *string `json:"creator_name,omitempty" db:"-"`
	ClaimerName *string `json:"claimer_name,omitempty" db:"-"`
}
