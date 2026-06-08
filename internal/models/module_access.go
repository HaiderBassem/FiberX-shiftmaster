package models

import (
	"time"

	"github.com/google/uuid"
)

type ModuleDepartmentAccess struct {
	ModuleName   string     `json:"module_name"`
	DepartmentID uuid.UUID  `json:"department_id"`
	GrantedBy    *uuid.UUID `json:"granted_by"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ModuleEmployeeExclusion struct {
	ModuleName string     `json:"module_name"`
	EmployeeID uuid.UUID  `json:"employee_id"`
	ExcludedBy *uuid.UUID `json:"excluded_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ModuleAccessResponse is a helper struct for returning a single module's access state
type ModuleAccessResponse struct {
	ModuleName    string      `json:"module_name"`
	Departments   []uuid.UUID `json:"departments"` // Departments that have this module enabled
	ExcludedEmps  []uuid.UUID `json:"excluded_employees"` // Employees who are explicitly excluded
}
