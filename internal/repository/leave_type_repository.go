package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type LeaveTypeRepository interface {
	GetAll(ctx context.Context) ([]models.LeaveType, error)
	GetActive(ctx context.Context) ([]models.LeaveType, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.LeaveType, error)
	Create(ctx context.Context, lt *models.LeaveType) error
	Update(ctx context.Context, lt *models.LeaveType) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type leaveTypeRepo struct {
	db *database.DB
}

func NewLeaveTypeRepository(db *database.DB) LeaveTypeRepository {
	return &leaveTypeRepo{db: db}
}

const leaveTypeCols = `id, name_ar, name_en, is_paid, color_code, is_active, requires_approval, created_at, updated_at`

func (r *leaveTypeRepo) scanRows(rows pgx.Rows) ([]models.LeaveType, error) {
	var lts []models.LeaveType
	for rows.Next() {
		var lt models.LeaveType
		if err := rows.Scan(
			&lt.ID, &lt.NameAr, &lt.NameEn, &lt.IsPaid, &lt.ColorCode,
			&lt.IsActive, &lt.RequiresApproval, &lt.CreatedAt, &lt.UpdatedAt,
		); err != nil {
			return nil, err
		}
		lts = append(lts, lt)
	}
	return lts, rows.Err()
}

func (r *leaveTypeRepo) GetAll(ctx context.Context) ([]models.LeaveType, error) {
	rows, err := r.db.Query(ctx, `SELECT `+leaveTypeCols+` FROM leave_types ORDER BY name_en`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

func (r *leaveTypeRepo) GetActive(ctx context.Context) ([]models.LeaveType, error) {
	rows, err := r.db.Query(ctx, `SELECT `+leaveTypeCols+` FROM leave_types WHERE is_active = true ORDER BY name_en`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

func (r *leaveTypeRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.LeaveType, error) {
	var lt models.LeaveType
	err := r.db.QueryRow(ctx, `SELECT `+leaveTypeCols+` FROM leave_types WHERE id = $1`, id).Scan(
		&lt.ID, &lt.NameAr, &lt.NameEn, &lt.IsPaid, &lt.ColorCode,
		&lt.IsActive, &lt.RequiresApproval, &lt.CreatedAt, &lt.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("leave type not found")
		}
		return nil, err
	}
	return &lt, nil
}

func (r *leaveTypeRepo) Create(ctx context.Context, lt *models.LeaveType) error {
	query := `INSERT INTO leave_types (name_ar, name_en, is_paid, color_code, is_active, requires_approval)
	          VALUES ($1, $2, $3, $4, $5, $6)
	          RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		lt.NameAr, lt.NameEn, lt.IsPaid, lt.ColorCode, lt.IsActive, lt.RequiresApproval,
	).Scan(&lt.ID, &lt.CreatedAt, &lt.UpdatedAt)
}

func (r *leaveTypeRepo) Update(ctx context.Context, lt *models.LeaveType) error {
	query := `UPDATE leave_types SET name_ar = $1, name_en = $2, is_paid = $3, color_code = $4,
	          is_active = $5, requires_approval = $6, updated_at = CURRENT_TIMESTAMP
	          WHERE id = $7 RETURNING updated_at`
	return r.db.QueryRow(ctx, query,
		lt.NameAr, lt.NameEn, lt.IsPaid, lt.ColorCode, lt.IsActive, lt.RequiresApproval, lt.ID,
	).Scan(&lt.UpdatedAt)
}

func (r *leaveTypeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM leave_types WHERE id = $1`, id)
	return err
}
