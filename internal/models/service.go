package models

import (
	"time"

	"github.com/google/uuid"
)

// ServiceCategory is a top-level grouping card for FTTH service plans.
type ServiceCategory struct {
	ID           uuid.UUID `json:"id"`
	DepartmentID uuid.UUID `json:"department_id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	IsActive     bool      `json:"is_active"`
	SortOrder   int       `json:"sort_order"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Joined / computed fields
	CreatorName    *string `json:"creator_name,omitempty" db:"-"`
	DepartmentName string  `json:"department_name" db:"-"`
	PlanCount      int     `json:"plan_count" db:"-"`
	IsShared       bool    `json:"is_shared" db:"-"`
}

// ServiceCategoryShare represents a cross-department sharing relationship
type ServiceCategoryShare struct {
	ID             uuid.UUID `json:"id" db:"id"`
	CategoryID     uuid.UUID `json:"category_id" db:"category_id"`
	DepartmentID   uuid.UUID `json:"department_id" db:"department_id"`
	GrantedBy      uuid.UUID `json:"granted_by" db:"granted_by"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`

	// Virtual field
	DepartmentName string    `json:"department_name" db:"-"`
}

// ServicePlan is an FTTH internet package within a category.
type ServicePlan struct {
	ID              uuid.UUID `json:"id"`
	CategoryID      uuid.UUID `json:"category_id"`
	Name            string    `json:"name"`
	Price           float64   `json:"price"`
	DurationDays    int       `json:"duration_days"`
	SpeedDownload   *string   `json:"speed_download,omitempty"`
	SpeedUpload     *string   `json:"speed_upload,omitempty"`
	DataCap         *string   `json:"data_cap,omitempty"`
	Province        string    `json:"province"`
	ConnectionType  string    `json:"connection_type"`
	InstallationFee float64   `json:"installation_fee"`
	RouterIncluded  bool      `json:"router_included"`
	IPType          string    `json:"ip_type"`
	Description     *string   `json:"description,omitempty"`
	CabinetNotes    *string   `json:"cabinet_notes,omitempty"`
	Features        *string   `json:"features,omitempty"` // JSONB as string
	IsActive        bool      `json:"is_active"`
	SortOrder       int       `json:"sort_order"`
	CreatedBy       uuid.UUID `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Joined fields
	CreatorName  *string `json:"creator_name,omitempty" db:"-"`
	CategoryName *string `json:"category_name,omitempty" db:"-"`
}
