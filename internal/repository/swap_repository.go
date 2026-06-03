package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// SwapRepository defines the interface for shift swap request data access.
type SwapRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.ShiftSwap, error)
	GetByRequester(ctx context.Context, requesterID uuid.UUID) ([]models.ShiftSwap, error)
	GetByTarget(ctx context.Context, targetID uuid.UUID) ([]models.ShiftSwap, error)
	GetPendingForEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ShiftSwap, error)
	GetPendingForManager(ctx context.Context, approverRole string, approverDeptID *uuid.UUID) ([]models.ShiftSwap, error)
	Create(ctx context.Context, swap *models.ShiftSwap) error
	EmployeeRespond(ctx context.Context, id uuid.UUID, accepted bool) error
	ManagerApprove(ctx context.Context, id uuid.UUID, approverID uuid.UUID, approverRole string) error
	Reject(ctx context.Context, id uuid.UUID, approverID uuid.UUID, approverRole string) error
	Cancel(ctx context.Context, id uuid.UUID) error
	GetHistoryForManager(ctx context.Context, approverRole string, approverDeptID *uuid.UUID) ([]models.ShiftSwap, error)
}

type swapRepo struct {
	db *database.DB
}

func NewSwapRepository(db *database.DB) SwapRepository {
	return &swapRepo{db: db}
}

const swapColumns = `id, requester_id, target_employee_id, shift_date, shift_id, reason, status,
	approved_by_team_leader, approved_by_manager, approval_date, created_at, updated_at`
const swapColumnsWithNames = `ss.id, ss.requester_id, ss.target_employee_id, ss.shift_date, ss.shift_id, ss.reason, ss.status,
	ss.approved_by_team_leader, ss.approved_by_manager, ss.approval_date, ss.created_at, ss.updated_at,
	(CONCAT(r.first_name, ' ', r.last_name)) AS requester_name,
	(CONCAT(t.first_name, ' ', t.last_name)) AS target_employee_name`

func (r *swapRepo) scanSwap(row pgx.Row) (*models.ShiftSwap, error) {
	var s models.ShiftSwap
	err := row.Scan(&s.ID, &s.RequesterID, &s.TargetEmployeeID, &s.ShiftDate, &s.ShiftID,
		&s.Reason, &s.Status, &s.ApprovedByTeamLeader, &s.ApprovedByManager, &s.ApprovalDate,
		&s.CreatedAt, &s.UpdatedAt, &s.RequesterName, &s.TargetEmployeeName)
	return &s, err
}

func (r *swapRepo) scanSwaps(rows pgx.Rows) ([]models.ShiftSwap, error) {
	var swaps []models.ShiftSwap
	for rows.Next() {
		var s models.ShiftSwap
		if err := rows.Scan(&s.ID, &s.RequesterID, &s.TargetEmployeeID, &s.ShiftDate, &s.ShiftID,
			&s.Reason, &s.Status, &s.ApprovedByTeamLeader, &s.ApprovedByManager, &s.ApprovalDate,
			&s.CreatedAt, &s.UpdatedAt, &s.RequesterName, &s.TargetEmployeeName); err != nil {
			return nil, fmt.Errorf("scan swap: %w", err)
		}
		swaps = append(swaps, s)
	}
	return swaps, rows.Err()
}

func (r *swapRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.ShiftSwap, error) {
	s, err := r.scanSwap(r.db.QueryRow(ctx, `
		SELECT `+swapColumnsWithNames+`
		FROM shift_swaps ss
		JOIN employees r ON r.id = ss.requester_id
		JOIN employees t ON t.id = ss.target_employee_id
		WHERE ss.id = $1
	`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("swap not found: %w", err)
		}
		return nil, fmt.Errorf("get swap by id: %w", err)
	}
	return s, nil
}

func (r *swapRepo) GetByRequester(ctx context.Context, requesterID uuid.UUID) ([]models.ShiftSwap, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+swapColumnsWithNames+`
		 FROM shift_swaps ss
		 JOIN employees r ON r.id = ss.requester_id
		 JOIN employees t ON t.id = ss.target_employee_id
		 WHERE ss.requester_id = $1
		 ORDER BY ss.created_at DESC`, requesterID)
	if err != nil {
		return nil, fmt.Errorf("get swaps by requester: %w", err)
	}
	defer rows.Close()
	return r.scanSwaps(rows)
}

func (r *swapRepo) GetByTarget(ctx context.Context, targetID uuid.UUID) ([]models.ShiftSwap, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+swapColumnsWithNames+`
		 FROM shift_swaps ss
		 JOIN employees r ON r.id = ss.requester_id
		 JOIN employees t ON t.id = ss.target_employee_id
		 WHERE ss.target_employee_id = $1
		 ORDER BY ss.created_at DESC`, targetID)
	if err != nil {
		return nil, fmt.Errorf("get swaps by target: %w", err)
	}
	defer rows.Close()
	return r.scanSwaps(rows)
}

// GetPendingForEmployee returns swap requests waiting for an employee's acceptance.
func (r *swapRepo) GetPendingForEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ShiftSwap, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+swapColumnsWithNames+`
		 FROM shift_swaps ss
		 JOIN employees r ON r.id = ss.requester_id
		 JOIN employees t ON t.id = ss.target_employee_id
		 WHERE ss.target_employee_id = $1
		   AND ss.status = 'pending'
		 ORDER BY ss.created_at`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get pending swaps for employee: %w", err)
	}
	defer rows.Close()
	return r.scanSwaps(rows)
}

// GetPendingForManager returns swap requests accepted by both employees, awaiting manager/team_leader approval.
func (r *swapRepo) GetPendingForManager(ctx context.Context, approverRole string, approverDeptID *uuid.UUID) ([]models.ShiftSwap, error) {
	query := `SELECT ` + swapColumnsWithNames + `
		 FROM shift_swaps ss
		 JOIN employees r ON r.id = ss.requester_id
		 JOIN employees t ON t.id = ss.target_employee_id
		 WHERE ss.status::text = 'employee_accepted'
		   AND ss.approved_by_team_leader IS NULL
		   AND ss.approved_by_manager IS NULL`
	args := []interface{}{}

	if approverRole != "admin" {
		if approverDeptID == nil {
			return []models.ShiftSwap{}, nil
		}
		query += ` AND r.department_id = $1`
		args = append(args, *approverDeptID)
	}
	query += ` ORDER BY ss.created_at`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get pending swaps for manager: %w", err)
	}
	defer rows.Close()
	return r.scanSwaps(rows)
}

// GetHistoryForManager returns all swap requests for the manager's department.
func (r *swapRepo) GetHistoryForManager(ctx context.Context, approverRole string, approverDeptID *uuid.UUID) ([]models.ShiftSwap, error) {
	query := `SELECT ` + swapColumnsWithNames + `
		 FROM shift_swaps ss
		 JOIN employees r ON r.id = ss.requester_id
		 JOIN employees t ON t.id = ss.target_employee_id`
	args := []interface{}{}

	if approverRole != "admin" {
		if approverDeptID == nil {
			return []models.ShiftSwap{}, nil
		}
		query += ` WHERE r.department_id = $1`
		args = append(args, *approverDeptID)
	}
	query += ` ORDER BY ss.updated_at DESC LIMIT 100`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get swap history for manager: %w", err)
	}
	defer rows.Close()
	return r.scanSwaps(rows)
}

func (r *swapRepo) Create(ctx context.Context, swap *models.ShiftSwap) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO shift_swaps (requester_id, target_employee_id, shift_date, shift_id, reason)
		 VALUES ($1,$2,$3,$4,$5) RETURNING id, status, created_at, updated_at`,
		swap.RequesterID, swap.TargetEmployeeID, swap.ShiftDate, swap.ShiftID, swap.Reason,
	).Scan(&swap.ID, &swap.Status, &swap.CreatedAt, &swap.UpdatedAt)
}

// EmployeeRespond is called when the target employee accepts or rejects the swap.
func (r *swapRepo) EmployeeRespond(ctx context.Context, id uuid.UUID, accepted bool) error {
	if !accepted {
		_, err := r.db.Exec(ctx,
			`UPDATE shift_swaps SET status='rejected', updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id)
		return err
	}
	// Employee accepted -> move to manager approval phase
	_, err := r.db.Exec(ctx,
		`UPDATE shift_swaps SET status='employee_accepted', updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id)
	return err
}

// ManagerApprove approves the swap (by team_leader or manager).
func (r *swapRepo) ManagerApprove(ctx context.Context, id uuid.UUID, approverID uuid.UUID, approverRole string) error {
	var query string
	switch approverRole {
	case "team_leader":
		query = `UPDATE shift_swaps SET status='approved', approved_by_team_leader=$1, approval_date=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=$2`
	case "manager":
		query = `UPDATE shift_swaps SET status='approved', approved_by_manager=$1, approval_date=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=$2`
	default:
		return fmt.Errorf("invalid approver role: %s", approverRole)
	}
	_, err := r.db.Exec(ctx, query, approverID, id)
	return err
}

func (r *swapRepo) Reject(ctx context.Context, id uuid.UUID, approverID uuid.UUID, approverRole string) error {
	var query string
	switch approverRole {
	case "team_leader":
		query = `UPDATE shift_swaps SET status='rejected', approved_by_team_leader=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`
	case "manager":
		query = `UPDATE shift_swaps SET status='rejected', approved_by_manager=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`
	default:
		return fmt.Errorf("invalid approver role: %s", approverRole)
	}
	_, err := r.db.Exec(ctx, query, approverID, id)
	return err
}

func (r *swapRepo) Cancel(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE shift_swaps SET status='cancelled', updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id)
	return err
}
