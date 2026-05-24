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
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
