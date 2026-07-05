package models

import (
	"time"

	"github.com/google/uuid"
)

type FiberxData struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	DepartmentID   uuid.UUID  `json:"department_id" db:"department_id"`
	Title          string     `json:"title" db:"title"`
	Content        string     `json:"content" db:"content"`
	CreatedBy      *uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	
	// Virtual fields for frontend
	CreatorName    string     `json:"creator_name" db:"-"`
	DepartmentName string     `json:"department_name" db:"-"`
}

type FiberxDataDepartmentShare struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	DataID         uuid.UUID  `json:"data_id" db:"data_id"`
	DepartmentID   uuid.UUID  `json:"department_id" db:"department_id"`
	AccessLevel    string     `json:"access_level" db:"access_level"` // 'read', 'write'
	GrantedBy      *uuid.UUID `json:"granted_by" db:"granted_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	
	// Virtual field
	DepartmentName string     `json:"department_name" db:"-"`
}

type FiberxDataEmployeeAccess struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	DataID         uuid.UUID  `json:"data_id" db:"data_id"`
	EmployeeID     uuid.UUID  `json:"employee_id" db:"employee_id"`
	AccessLevel    string     `json:"access_level" db:"access_level"` // 'read', 'write', 'hide'
	GrantedBy      *uuid.UUID `json:"granted_by" db:"granted_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	
	// Virtual fields
	EmployeeName   string     `json:"employee_name" db:"-"`
}

type FiberxDataResponse struct {
	FiberxData
	IsShared    bool   `json:"is_shared"`
	AccessLevel string `json:"access_level"` // Effective access level for the current user
}
