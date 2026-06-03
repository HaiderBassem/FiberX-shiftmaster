package models

import (
	"time"

	"github.com/google/uuid"
)

// EmployeeLeaveBalance tracks an employee's leave balance for a specific year and leave type.
type EmployeeLeaveBalance struct {
	ID            uuid.UUID `json:"id"`
	EmployeeID    uuid.UUID `json:"employee_id"`
	LeaveTypeID   uuid.UUID `json:"leave_type_id"`
	Year          int       `json:"year"`
	AllocatedDays float64   `json:"allocated_days"`
	UsedDays      float64   `json:"used_days"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Optional joined fields
	LeaveTypeNameAr string `json:"leave_type_name_ar,omitempty"`
	LeaveTypeNameEn string `json:"leave_type_name_en,omitempty"`
	ColorCode       string `json:"color_code,omitempty"`
}
