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
	Year            int       `json:"year"`
	Month           int       `json:"month"` // 0 for annual, 1-12 for monthly
	AllocatedAmount float64   `json:"allocated_amount"`
	UsedAmount      float64   `json:"used_amount"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Optional joined fields
	LeaveTypeNameAr string `json:"leave_type_name_ar,omitempty" db:"-"`
	LeaveTypeNameEn string `json:"leave_type_name_en,omitempty" db:"-"`
	ColorCode       string `json:"color_code,omitempty" db:"-"`
	Unit            string `json:"unit,omitempty" db:"-"`
	ResetCycle      string `json:"reset_cycle,omitempty" db:"-"`
}
