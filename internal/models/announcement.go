package models

import (
	"time"

	"github.com/google/uuid"
)

type Announcement struct {
	ID           uuid.UUID `json:"id"`
	DepartmentID uuid.UUID `json:"department_id"`
	Title        string    `json:"title"`
	Message      string    `json:"message"`
	Priority     string    `json:"priority"` // 'info', 'normal', 'important', 'critical'
	IsActive     bool      `json:"is_active"`
	IsTicker     bool      `json:"is_ticker"`
	Images       []string  `json:"images"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Joins
	CreatorName string `json:"creator_name,omitempty"`
}
