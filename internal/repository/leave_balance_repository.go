package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

type LeaveBalanceRepository interface {
	GetByEmployeeAndYear(ctx context.Context, employeeID uuid.UUID, year int) ([]models.EmployeeLeaveBalance, error)
	GetByEmployeeLeaveTypeAndYear(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int) (*models.EmployeeLeaveBalance, error)
	UpsertBalance(ctx context.Context, balance *models.EmployeeLeaveBalance) error
	IncrementUsedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, days float64) error
	SyncAllocatedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, allocatedDays float64) error
	UpdateAllocatedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, allocatedDays float64) error
}

type leaveBalanceRepo struct {
	db *database.DB
}

func NewLeaveBalanceRepository(db *database.DB) LeaveBalanceRepository {
	return &leaveBalanceRepo{db: db}
}

const leaveBalanceCols = `b.id, b.employee_id, b.leave_type_id, b.year, b.allocated_days, b.used_days, b.created_at, b.updated_at`

func (r *leaveBalanceRepo) GetByEmployeeAndYear(ctx context.Context, employeeID uuid.UUID, year int) ([]models.EmployeeLeaveBalance, error) {
	query := `
		SELECT ` + leaveBalanceCols + `, t.name_ar, t.name_en, t.color_code
		FROM employee_leave_balances b
		JOIN leave_types t ON b.leave_type_id = t.id
		WHERE b.employee_id = $1 AND b.year = $2
		ORDER BY t.name_en
	`
	rows, err := r.db.Query(ctx, query, employeeID, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []models.EmployeeLeaveBalance
	for rows.Next() {
		var b models.EmployeeLeaveBalance
		if err := rows.Scan(
			&b.ID, &b.EmployeeID, &b.LeaveTypeID, &b.Year, &b.AllocatedDays, &b.UsedDays, &b.CreatedAt, &b.UpdatedAt,
			&b.LeaveTypeNameAr, &b.LeaveTypeNameEn, &b.ColorCode,
		); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}
	return balances, rows.Err()
}

func (r *leaveBalanceRepo) GetByEmployeeLeaveTypeAndYear(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int) (*models.EmployeeLeaveBalance, error) {
	query := `
		SELECT ` + leaveBalanceCols + `
		FROM employee_leave_balances b
		WHERE b.employee_id = $1 AND b.leave_type_id = $2 AND b.year = $3
	`
	var b models.EmployeeLeaveBalance
	err := r.db.QueryRow(ctx, query, employeeID, leaveTypeID, year).Scan(
		&b.ID, &b.EmployeeID, &b.LeaveTypeID, &b.Year, &b.AllocatedDays, &b.UsedDays, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("leave balance not found")
		}
		return nil, err
	}
	return &b, nil
}

func (r *leaveBalanceRepo) UpsertBalance(ctx context.Context, b *models.EmployeeLeaveBalance) error {
	query := `
		INSERT INTO employee_leave_balances (employee_id, leave_type_id, year, allocated_days, used_days)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (employee_id, leave_type_id, year)
		DO UPDATE SET allocated_days = EXCLUDED.allocated_days, used_days = EXCLUDED.used_days, updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, b.EmployeeID, b.LeaveTypeID, b.Year, b.AllocatedDays, b.UsedDays).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
}

func (r *leaveBalanceRepo) IncrementUsedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, days float64) error {
	query := `
		UPDATE employee_leave_balances
		SET used_days = used_days + $1, updated_at = CURRENT_TIMESTAMP
		WHERE employee_id = $2 AND leave_type_id = $3 AND year = $4
	`
	tag, err := r.db.Exec(ctx, query, days, employeeID, leaveTypeID, year)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("leave balance record not found to increment")
	}
	return nil
}

func (r *leaveBalanceRepo) SyncAllocatedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, allocatedDays float64) error {
	query := `
		INSERT INTO employee_leave_balances (employee_id, leave_type_id, year, allocated_days, used_days)
		VALUES ($1, $2, $3, $4, 0)
		ON CONFLICT (employee_id, leave_type_id, year)
		DO UPDATE SET allocated_days = EXCLUDED.allocated_days, updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(ctx, query, employeeID, leaveTypeID, year, allocatedDays)
	return err
}

func (r *leaveBalanceRepo) UpdateAllocatedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, allocatedDays float64) error {
	query := `
		UPDATE employee_leave_balances
		SET allocated_days = $1, updated_at = CURRENT_TIMESTAMP
		WHERE employee_id = $2 AND leave_type_id = $3 AND year = $4
	`
	tag, err := r.db.Exec(ctx, query, allocatedDays, employeeID, leaveTypeID, year)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// If no record exists yet, insert it with 0 used days
		return r.SyncAllocatedDays(ctx, employeeID, leaveTypeID, year, allocatedDays)
	}
	return nil
}
