package models

import (
	"time"

	"github.com/google/uuid"
)

type ExternalLink struct {
	ID        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	URL       string     `json:"url"`
	IconName  string     `json:"icon_name"`
	CreatedAt time.Time  `json:"created_at"`
	CreatedBy *uuid.UUID `json:"created_by"`
}

type LinkDepartmentAccess struct {
	LinkID       uuid.UUID  `json:"link_id"`
	DepartmentID uuid.UUID  `json:"department_id"`
	GrantedBy    *uuid.UUID `json:"granted_by"`
	CreatedAt    time.Time  `json:"created_at"`
}

type LinkEmployeeExclusion struct {
	LinkID     uuid.UUID  `json:"link_id"`
	EmployeeID uuid.UUID  `json:"employee_id"`
	ExcludedBy *uuid.UUID `json:"excluded_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

// LinkAccessResponse is a helper struct for returning a single link's access state
type LinkAccessResponse struct {
	LinkID        uuid.UUID   `json:"link_id"`
	Title         string      `json:"title"`
	Departments   []uuid.UUID `json:"departments"` // Departments that have this link enabled
	ExcludedEmps  []uuid.UUID `json:"excluded_employees"` // Employees who are explicitly excluded
}
