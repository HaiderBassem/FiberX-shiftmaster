package models

import (
	"time"

	"github.com/google/uuid"
)

// LeaveType represents a dynamic leave type configurable by admins
type LeaveType struct {
	ID               uuid.UUID `json:"id"`
	NameAr           string    `json:"name_ar"`
	NameEn           string    `json:"name_en"`
	IsPaid           bool      `json:"is_paid"`
	ColorCode        string    `json:"color_code"`
	IsActive         bool      `json:"is_active"`
	RequiresApproval bool      `json:"requires_approval"`
	DaysPerYear      int       `json:"days_per_year"` // Keeps original name for JSON backward compat if needed, but we'll use it as Quota
	Unit             string    `json:"unit"`          // 'days' or 'hours'
	ResetCycle       string    `json:"reset_cycle"`   // 'annual' or 'monthly'
	CarriesForward   bool      `json:"carries_forward"`
	// IsHourly and BypassesDailyLimit carry semantics that used to be inferred by
	// matching name_en, which administrators can rename at will.
	IsHourly           bool      `json:"is_hourly"`
	BypassesDailyLimit bool      `json:"bypasses_daily_limit"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
