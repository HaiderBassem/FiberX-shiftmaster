package models

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID                 uuid.UUID `json:"id"`
	SourceDepartmentID uuid.UUID `json:"source_department_id"`
	TargetDepartmentID uuid.UUID `json:"target_department_id"`
	CreatorID          uuid.UUID `json:"creator_id"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Status             string    `json:"status"` // open, closed
	ClosedBy           *uuid.UUID `json:"closed_by"`
	Attachments        *string   `json:"attachments"` // JSONB array of strings
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	// Joined fields for UI
	CreatorName        *string          `json:"creator_name,omitempty" db:"-"`
	CreatorProfileImage *string         `json:"creator_profile_image,omitempty" db:"-"`
	SourceDepartment   *string          `json:"source_department,omitempty" db:"-"`
	TargetDepartment   *string          `json:"target_department,omitempty" db:"-"`
	ClosedByName       *string          `json:"closed_by_name,omitempty" db:"-"`
	Comments           []TicketComment `json:"comments,omitempty" db:"-"`
}

type TicketComment struct {
	ID          uuid.UUID `json:"id"`
	TicketID    uuid.UUID `json:"ticket_id"`
	EmployeeID  uuid.UUID `json:"employee_id"`
	AuthorName  string    `json:"author_name"`
	AuthorImage *string   `json:"author_image,omitempty"`
	Comment     string    `json:"comment"`
	Attachments *string   `json:"attachments"` // JSONB array of strings
	CreatedAt   time.Time `json:"created_at"`
}
