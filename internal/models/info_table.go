package models

import (
	"time"

	"github.com/google/uuid"
)

type InfoTableColumn struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"` // text, number, date, link, select
	Order int    `json:"order"`
}

type InfoTable struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Description  *string           `json:"description"`
	Columns      []InfoTableColumn `json:"columns"`
	DepartmentID  *uuid.UUID        `json:"department_id"`
	CreatedBy     *uuid.UUID        `json:"created_by"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	MyAccessLevel string            `json:"my_access_level,omitempty"`
}

type InfoTableDepartmentAccess struct {
	ID           uuid.UUID  `json:"id"`
	TableID      uuid.UUID  `json:"table_id"`
	DepartmentID uuid.UUID  `json:"department_id"`
	GrantedBy    *uuid.UUID `json:"granted_by"`
	CreatedAt    time.Time  `json:"created_at"`
}

type InfoTableEmployeeAccess struct {
	ID          uuid.UUID  `json:"id"`
	TableID     uuid.UUID  `json:"table_id"`
	EmployeeID  uuid.UUID  `json:"employee_id"`
	AccessLevel string     `json:"access_level"` // read, write
	GrantedBy   *uuid.UUID `json:"granted_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

type InfoTableRow struct {
	ID        uuid.UUID              `json:"id"`
	TableID   uuid.UUID              `json:"table_id"`
	Data      map[string]interface{} `json:"data"` // JSONB
	CreatedBy *uuid.UUID             `json:"created_by"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}
