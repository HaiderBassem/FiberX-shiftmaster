package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// LeaveRepository defines the interface for leave request data access.
type LeaveRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Leave, error)
	GetByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.Leave, error)
	GetByStatus(ctx context.Context, status string) ([]models.Leave, error)
	GetByDateRange(ctx context.Context, from, to time.Time) ([]models.Leave, error)
	GetPendingForApproval(ctx context.Context, approverRole string) ([]models.Leave, error)
	Create(ctx context.Context, leave *models.Leave) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, approverID uuid.UUID, approverRole string) error
	Reject(ctx context.Context, id uuid.UUID, rejectedBy uuid.UUID, reason string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type leaveRepo struct {
	db *database.DB
}

func NewLeaveRepository(db *database.DB) LeaveRepository {
	return &leaveRepo{db: db}
}

const leaveColumns = `id, employee_id, leave_type, start_date, end_date, total_days, reason, status,
	applied_date, approved_by_team_leader, approved_by_manager, rejection_reason, attachments,
	created_at, updated_at`

func (r *leaveRepo) scanLeave(row pgx.Row) (*models.Leave, error) {
	var l models.Leave
	err := row.Scan(
		&l.ID, &l.EmployeeID, &l.LeaveType, &l.StartDate, &l.EndDate, &l.TotalDays,
		&l.Reason, &l.Status, &l.AppliedDate, &l.ApprovedByTeamLeader, &l.ApprovedByManager,
		&l.RejectionReason, &l.Attachments, &l.CreatedAt, &l.UpdatedAt,
	)
	return &l, err
}

func (r *leaveRepo) scanLeaves(rows pgx.Rows) ([]models.Leave, error) {
	var leaves []models.Leave
	for rows.Next() {
		var l models.Leave
		if err := rows.Scan(
			&l.ID, &l.EmployeeID, &l.LeaveType, &l.StartDate, &l.EndDate, &l.TotalDays,
			&l.Reason, &l.Status, &l.AppliedDate, &l.ApprovedByTeamLeader, &l.ApprovedByManager,
			&l.RejectionReason, &l.Attachments, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan leave: %w", err)
		}
		leaves = append(leaves, l)
	}
	return leaves, rows.Err()
}

func (r *leaveRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Leave, error) {
	l, err := r.scanLeave(r.db.QueryRow(ctx,
		`SELECT `+leaveColumns+` FROM leaves WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("leave not found: %w", err)
		}
		return nil, fmt.Errorf("get leave by id: %w", err)
	}
	return l, nil
}

func (r *leaveRepo) GetByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.Leave, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+leaveColumns+` FROM leaves WHERE employee_id = $1 ORDER BY start_date DESC`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get leaves by employee: %w", err)
	}
	defer rows.Close()
	return r.scanLeaves(rows)
}

func (r *leaveRepo) GetByStatus(ctx context.Context, status string) ([]models.Leave, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+leaveColumns+` FROM leaves WHERE status = $1 ORDER BY applied_date DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("get leaves by status: %w", err)
	}
	defer rows.Close()
	return r.scanLeaves(rows)
}

func (r *leaveRepo) GetByDateRange(ctx context.Context, from, to time.Time) ([]models.Leave, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+leaveColumns+` FROM leaves WHERE start_date <= $2 AND end_date >= $1 ORDER BY start_date`, from, to)
	if err != nil {
		return nil, fmt.Errorf("get leaves by date range: %w", err)
	}
	defer rows.Close()
	return r.scanLeaves(rows)
}

func (r *leaveRepo) GetPendingForApproval(ctx context.Context, approverRole string) ([]models.Leave, error) {
	var statusFilter string
	switch approverRole {
	case "team_leader":
		statusFilter = "pending"
	case "manager":
		statusFilter = "approved_by_team_leader"
	default:
		return nil, fmt.Errorf("invalid approver role: %s", approverRole)
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+leaveColumns+` FROM leaves WHERE status = $1 ORDER BY applied_date`, statusFilter)
	if err != nil {
		return nil, fmt.Errorf("get pending leaves: %w", err)
	}
	defer rows.Close()
	return r.scanLeaves(rows)
}

func (r *leaveRepo) Create(ctx context.Context, leave *models.Leave) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO leaves (employee_id, leave_type, start_date, end_date, reason, attachments)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, total_days, status, applied_date, created_at, updated_at`,
		leave.EmployeeID, leave.LeaveType, leave.StartDate, leave.EndDate, leave.Reason, leave.Attachments,
	).Scan(&leave.ID, &leave.TotalDays, &leave.Status, &leave.AppliedDate, &leave.CreatedAt, &leave.UpdatedAt)
}

func (r *leaveRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, approverID uuid.UUID, approverRole string) error {
	var query string
	switch approverRole {
	case "team_leader":
		query = `UPDATE leaves SET status=$1, approved_by_team_leader=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3`
	case "manager":
		query = `UPDATE leaves SET status=$1, approved_by_manager=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3`
	default:
		return fmt.Errorf("invalid approver role: %s", approverRole)
	}
	_, err := r.db.Exec(ctx, query, status, approverID, id)
	return err
}

func (r *leaveRepo) Reject(ctx context.Context, id uuid.UUID, rejectedBy uuid.UUID, reason string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE leaves SET status='rejected', rejection_reason=$1, approved_by_manager=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3`,
		reason, rejectedBy, id)
	return err
}

func (r *leaveRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM leaves WHERE id=$1`, id)
	return err
}
