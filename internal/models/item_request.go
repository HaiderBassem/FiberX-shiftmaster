package models

import (
	"time"

	"github.com/google/uuid"
)

type ItemRequestCategory struct {
	ID           uuid.UUID `json:"id"`
	DepartmentID uuid.UUID `json:"department_id"`
	Name         string    `json:"name"`
	ToEmails     string    `json:"to_emails"`
	CCEmails     *string   `json:"cc_emails"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ItemRequest struct {
	ID           uuid.UUID `json:"id"`
	EmployeeID   uuid.UUID `json:"employee_id"`
	CategoryID   uuid.UUID `json:"category_id"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Joined fields for frontend display
	CategoryName *string `json:"category_name,omitempty"`
	EmployeeName *string `json:"employee_name,omitempty"`
}
