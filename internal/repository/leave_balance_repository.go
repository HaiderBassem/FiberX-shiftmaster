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
	GetByEmployeeLeaveTypeAndYear(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, month int) (*models.EmployeeLeaveBalance, error)
	UpsertBalance(ctx context.Context, balance *models.EmployeeLeaveBalance) error
	IncrementUsedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, month int, days float64) error
	SyncAllocatedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, month int, allocatedDays float64) error
	UpdateAllocatedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, month int, allocatedDays float64) error
}

type leaveBalanceRepo struct {
	db *database.DB
}

func NewLeaveBalanceRepository(db *database.DB) LeaveBalanceRepository {
	return &leaveBalanceRepo{db: db}
}

const leaveBalanceCols = `b.id, b.employee_id, b.leave_type_id, b.year, b.month, b.allocated_amount, b.used_amount, b.created_at, b.updated_at`

func (r *leaveBalanceRepo) GetByEmployeeAndYear(ctx context.Context, employeeID uuid.UUID, year int) ([]models.EmployeeLeaveBalance, error) {
	query := `
		SELECT ` + leaveBalanceCols + `, t.name_ar, t.name_en, t.color_code, t.unit, t.reset_cycle
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
			&b.ID, &b.EmployeeID, &b.LeaveTypeID, &b.Year, &b.Month, &b.AllocatedAmount, &b.UsedAmount, &b.CreatedAt, &b.UpdatedAt,
			&b.LeaveTypeNameAr, &b.LeaveTypeNameEn, &b.ColorCode, &b.Unit, &b.ResetCycle,
		); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}
	return balances, rows.Err()
}

func (r *leaveBalanceRepo) GetByEmployeeLeaveTypeAndYear(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, month int) (*models.EmployeeLeaveBalance, error) {
	query := `
		SELECT ` + leaveBalanceCols + `
		FROM employee_leave_balances b
		WHERE b.employee_id = $1 AND b.leave_type_id = $2 AND b.year = $3 AND b.month = $4
	`
	var b models.EmployeeLeaveBalance
	err := r.db.QueryRow(ctx, query, employeeID, leaveTypeID, year, month).Scan(
		&b.ID, &b.EmployeeID, &b.LeaveTypeID, &b.Year, &b.Month, &b.AllocatedAmount, &b.UsedAmount, &b.CreatedAt, &b.UpdatedAt,
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
		INSERT INTO employee_leave_balances (employee_id, leave_type_id, year, month, allocated_amount, used_amount)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (employee_id, leave_type_id, year, month)
		DO UPDATE SET allocated_amount = EXCLUDED.allocated_amount, used_amount = EXCLUDED.used_amount, updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, b.EmployeeID, b.LeaveTypeID, b.Year, b.Month, b.AllocatedAmount, b.UsedAmount).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
}

func (r *leaveBalanceRepo) IncrementUsedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, month int, amount float64) error {
	query := `
		UPDATE employee_leave_balances
		SET used_amount = used_amount + $1, updated_at = CURRENT_TIMESTAMP
		WHERE employee_id = $2 AND leave_type_id = $3 AND year = $4 AND month = $5
	`
	tag, err := r.db.Exec(ctx, query, amount, employeeID, leaveTypeID, year, month)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("leave balance record not found to increment")
	}
	return nil
}

func (r *leaveBalanceRepo) SyncAllocatedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, month int, allocatedAmount float64) error {
	query := `
		INSERT INTO employee_leave_balances (employee_id, leave_type_id, year, month, allocated_amount, used_amount)
		VALUES ($1, $2, $3, $4, $5, 0)
		ON CONFLICT (employee_id, leave_type_id, year, month)
		DO UPDATE SET allocated_amount = EXCLUDED.allocated_amount, updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(ctx, query, employeeID, leaveTypeID, year, month, allocatedAmount)
	return err
}

func (r *leaveBalanceRepo) UpdateAllocatedDays(ctx context.Context, employeeID, leaveTypeID uuid.UUID, year int, month int, allocatedAmount float64) error {
	query := `
		UPDATE employee_leave_balances
		SET allocated_amount = $1, updated_at = CURRENT_TIMESTAMP
		WHERE employee_id = $2 AND leave_type_id = $3 AND year = $4 AND month = $5
	`
	tag, err := r.db.Exec(ctx, query, allocatedAmount, employeeID, leaveTypeID, year, month)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// If no record exists yet, insert it with 0 used amount
		return r.SyncAllocatedDays(ctx, employeeID, leaveTypeID, year, month, allocatedAmount)
	}
	return nil
}
