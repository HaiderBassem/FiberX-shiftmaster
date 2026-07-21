package models

import (
	"time"

	"github.com/google/uuid"
)

// Province represents a location option for services.
type Province struct {
	ID        uuid.UUID `json:"id" db:"id"`
	DepartmentID uuid.UUID `json:"department_id" db:"department_id"`
	Name      string    `json:"name" db:"name"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedBy uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Joined/virtual fields
	CreatorName    *string `json:"creator_name,omitempty" db:"-"`
	DepartmentName string  `json:"department_name" db:"-"`
	IsShared       bool    `json:"is_shared" db:"-"`
}

// ProvinceShare represents a cross-department sharing relationship
type ProvinceShare struct {
	ID             uuid.UUID `json:"id" db:"id"`
	ProvinceID     uuid.UUID `json:"province_id" db:"province_id"`
	DepartmentID   uuid.UUID `json:"department_id" db:"department_id"`
	GrantedBy      uuid.UUID `json:"granted_by" db:"granted_by"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`

	// Virtual field
	DepartmentName string    `json:"department_name" db:"-"`
}
