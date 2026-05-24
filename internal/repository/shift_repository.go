package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// ShiftRepository defines the interface for shift data access.
type ShiftRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Shift, error)
	GetByCode(ctx context.Context, code string) (*models.Shift, error)
	GetAll(ctx context.Context, departmentID *uuid.UUID) ([]models.Shift, error)
	Create(ctx context.Context, shift *models.Shift) error
	Update(ctx context.Context, shift *models.Shift) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type shiftRepo struct {
	db *database.DB
}

func NewShiftRepository(db *database.DB) ShiftRepository {
	return &shiftRepo{db: db}
}

func (r *shiftRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Shift, error) {
	var s models.Shift
	err := r.db.QueryRow(ctx,
		`SELECT id, shift_code, name, name_en, start_time, end_time, color_code,
				requires_vehicle, min_rest_hours, department_id, created_at
		 FROM shifts WHERE id = $1`, id,
	).Scan(&s.ID, &s.ShiftCode, &s.Name, &s.NameEn, &s.StartTime, &s.EndTime,
		&s.ColorCode, &s.RequiresVehicle, &s.MinRestHours, &s.DepartmentID, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("shift not found: %w", err)
		}
		return nil, fmt.Errorf("get shift by id: %w", err)
	}
	return &s, nil
}

func (r *shiftRepo) GetByCode(ctx context.Context, code string) (*models.Shift, error) {
	var s models.Shift
	err := r.db.QueryRow(ctx,
		`SELECT id, shift_code, name, name_en, start_time, end_time, color_code,
				requires_vehicle, min_rest_hours, department_id, created_at
		 FROM shifts WHERE shift_code = $1`, code,
	).Scan(&s.ID, &s.ShiftCode, &s.Name, &s.NameEn, &s.StartTime, &s.EndTime,
		&s.ColorCode, &s.RequiresVehicle, &s.MinRestHours, &s.DepartmentID, &s.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("shift not found: %w", err)
		}
		return nil, fmt.Errorf("get shift by code: %w", err)
	}
	return &s, nil
}

func (r *shiftRepo) GetAll(ctx context.Context, departmentID *uuid.UUID) ([]models.Shift, error) {
	query := `SELECT id, shift_code, name, name_en, start_time, end_time, color_code, is_active, department_id, created_at, updated_at
		 FROM shifts WHERE ($1::uuid IS NULL OR department_id = $1 OR department_id IS NULL) ORDER BY start_time`
	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, fmt.Errorf("get all shifts: %w", err)
	}
	defer rows.Close()

	var shifts []models.Shift
	for rows.Next() {
		var s models.Shift
		if err := rows.Scan(&s.ID, &s.ShiftCode, &s.Name, &s.NameEn, &s.StartTime, &s.EndTime,
			&s.ColorCode, &s.RequiresVehicle, &s.MinRestHours, &s.DepartmentID, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan shift: %w", err)
		}
		shifts = append(shifts, s)
	}
	return shifts, rows.Err()
}

func (r *shiftRepo) Create(ctx context.Context, shift *models.Shift) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO shifts (shift_code, name, name_en, start_time, end_time, color_code,
			requires_vehicle, min_rest_hours, department_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, created_at`,
		shift.ShiftCode, shift.Name, shift.NameEn, shift.StartTime, shift.EndTime,
		shift.ColorCode, shift.RequiresVehicle, shift.MinRestHours, shift.DepartmentID,
	).Scan(&shift.ID, &shift.CreatedAt)
}

func (r *shiftRepo) Update(ctx context.Context, shift *models.Shift) error {
	_, err := r.db.Exec(ctx,
		`UPDATE shifts SET shift_code=$1, name=$2, name_en=$3, start_time=$4, end_time=$5, color_code=$6,
			requires_vehicle=$7, min_rest_hours=$8, department_id=$9 WHERE id=$10`,
		shift.ShiftCode, shift.Name, shift.NameEn, shift.StartTime, shift.EndTime,
		shift.ColorCode, shift.RequiresVehicle, shift.MinRestHours, shift.DepartmentID, shift.ID)
	return err
}

func (r *shiftRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM shifts WHERE id=$1`, id)
	return err
}
