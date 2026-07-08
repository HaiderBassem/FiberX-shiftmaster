package models

import (
	"time"

	"github.com/google/uuid"
)

// ServiceCategory is a top-level grouping card for FTTH service plans.
type ServiceCategory struct {
	ID          uuid.UUID `json:"id"`
	ProvinceID  uuid.UUID `json:"province_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsActive     bool      `json:"is_active"`
	SortOrder   int       `json:"sort_order"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Joined / computed fields
	CreatorName  *string `json:"creator_name,omitempty" db:"-"`
	ProvinceName string  `json:"province_name" db:"-"`
	PlanCount    int     `json:"plan_count" db:"-"`
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
