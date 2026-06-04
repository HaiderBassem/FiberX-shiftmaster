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
	GetPendingForApproval(ctx context.Context, approverRole string, approverDeptID *uuid.UUID) ([]models.Leave, error)
	Create(ctx context.Context, leave *models.Leave) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, approverID uuid.UUID, approverRole string) error
	Reject(ctx context.Context, id uuid.UUID, rejectedBy uuid.UUID, reason string) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Approval tracking
	RecordApproval(ctx context.Context, leaveID, approverID uuid.UUID, approverRole, action string, notes *string) error
	CountTLApprovals(ctx context.Context, leaveID uuid.UUID) (int, error)
	HasApproved(ctx context.Context, leaveID, approverID uuid.UUID) (bool, error)
	GetLeaveHistory(ctx context.Context, departmentID *uuid.UUID) ([]models.LeaveHistoryRow, error)
	GetPendingLeavesRich(ctx context.Context, approverRole string, approverDeptID *uuid.UUID) ([]models.PendingLeaveRich, error)
}

type leaveRepo struct {
	db *database.DB
}

func NewLeaveRepository(db *database.DB) LeaveRepository {
	return &leaveRepo{db: db}
}

const leaveColumns = `l.id, l.employee_id, l.leave_type_id, lt.name_ar as leave_type_name_ar, lt.name_en as leave_type_name_en, l.start_date, l.end_date, l.total_days, l.reason, l.status,
	l.applied_date, l.approved_by_team_leader, l.approved_by_manager, l.rejection_reason, l.attachments,
	l.start_time, l.end_time, l.created_at, l.updated_at`

func (r *leaveRepo) scanLeave(row pgx.Row) (*models.Leave, error) {
	var l models.Leave
	err := row.Scan(
		&l.ID, &l.EmployeeID, &l.LeaveTypeID, &l.LeaveTypeNameAr, &l.LeaveTypeNameEn, &l.StartDate, &l.EndDate, &l.TotalDays,
		&l.Reason, &l.Status, &l.AppliedDate, &l.ApprovedByTeamLeader, &l.ApprovedByManager,
		&l.RejectionReason, &l.Attachments, &l.StartTime, &l.EndTime, &l.CreatedAt, &l.UpdatedAt,
	)
	return &l, err
}

func (r *leaveRepo) scanLeaves(rows pgx.Rows) ([]models.Leave, error) {
	var leaves []models.Leave
	for rows.Next() {
		var l models.Leave
		if err := rows.Scan(
			&l.ID, &l.EmployeeID, &l.LeaveTypeID, &l.LeaveTypeNameAr, &l.LeaveTypeNameEn, &l.StartDate, &l.EndDate, &l.TotalDays,
			&l.Reason, &l.Status, &l.AppliedDate, &l.ApprovedByTeamLeader, &l.ApprovedByManager,
			&l.RejectionReason, &l.Attachments, &l.StartTime, &l.EndTime, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan leave: %w", err)
		}
		leaves = append(leaves, l)
	}
	return leaves, rows.Err()
}

func (r *leaveRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Leave, error) {
	l, err := r.scanLeave(r.db.QueryRow(ctx,
		`SELECT `+leaveColumns+` FROM leaves l LEFT JOIN leave_types lt ON lt.id = l.leave_type_id WHERE l.id = $1`, id))
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
		`SELECT `+leaveColumns+` FROM leaves l LEFT JOIN leave_types lt ON lt.id = l.leave_type_id WHERE l.employee_id = $1 ORDER BY l.start_date DESC`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get leaves by employee: %w", err)
	}
	defer rows.Close()
	return r.scanLeaves(rows)
}

func (r *leaveRepo) GetByStatus(ctx context.Context, status string) ([]models.Leave, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+leaveColumns+` FROM leaves l LEFT JOIN leave_types lt ON lt.id = l.leave_type_id WHERE l.status = $1 ORDER BY l.applied_date DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("get leaves by status: %w", err)
	}
	defer rows.Close()
	return r.scanLeaves(rows)
}

func (r *leaveRepo) GetByDateRange(ctx context.Context, from, to time.Time) ([]models.Leave, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+leaveColumns+` FROM leaves l LEFT JOIN leave_types lt ON lt.id = l.leave_type_id WHERE l.start_date <= $2 AND l.end_date >= $1 ORDER BY l.start_date`, from, to)
	if err != nil {
		return nil, fmt.Errorf("get leaves by date range: %w", err)
	}
	defer rows.Close()
	return r.scanLeaves(rows)
}

func (r *leaveRepo) GetPendingForApproval(ctx context.Context, approverRole string, approverDeptID *uuid.UUID) ([]models.Leave, error) {
	var statusFilter string
	switch approverRole {
	case "team_leader":
		statusFilter = "pending"
	case "manager":
		statusFilter = "approved_by_team_leader"
	case "admin":
		statusFilter = "pending" // Admin can see all pending leaves initially (or maybe all statuses, but let's stick to pending for approval queues)
	default:
		return nil, fmt.Errorf("invalid approver role: %s", approverRole)
	}

	query := `SELECT l.id, l.employee_id, l.leave_type_id, lt.name_ar as leave_type_name_ar, lt.name_en as leave_type_name_en, l.start_date, l.end_date, l.total_days, l.reason, l.status,
	          l.applied_date, l.approved_by_team_leader, l.approved_by_manager, l.rejection_reason, l.attachments,
	          l.start_time, l.end_time, l.created_at, l.updated_at
	          FROM leaves l
	          JOIN employees e ON e.id = l.employee_id
			  LEFT JOIN leave_types lt ON lt.id = l.leave_type_id
	          WHERE l.status = $1`
	args := []interface{}{statusFilter}

	if approverRole != "admin" {
		if approverDeptID == nil {
			return []models.Leave{}, nil // non-admin without a department sees nothing
		}
		query += ` AND e.department_id = $2`
		args = append(args, *approverDeptID)
	}
	query += ` ORDER BY l.applied_date`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get pending leaves: %w", err)
	}
	defer rows.Close()
	return r.scanLeaves(rows)
}

// GetPendingLeavesRich returns pending leaves with employee details for the approval dashboard.
func (r *leaveRepo) GetPendingLeavesRich(ctx context.Context, approverRole string, approverDeptID *uuid.UUID) ([]models.PendingLeaveRich, error) {
	var whereClause string
	switch approverRole {
	case "team_leader":
		// Team leaders only see pending leaves from regular employees
		whereClause = "l.status = 'pending' AND e.role = 'employee'"
	case "manager":
		// Managers see leaves approved by TLs, PLUS pending leaves from TLs themselves
		whereClause = "(l.status = 'approved_by_team_leader' OR (l.status = 'pending' AND e.role = 'team_leader'))"
	case "admin":
		whereClause = "l.status IN ('pending', 'approved_by_team_leader')"
	default:
		return nil, fmt.Errorf("invalid approver role: %s", approverRole)
	}

	query := `SELECT l.id, l.employee_id, l.leave_type_id, lt.name_ar, lt.name_en, l.start_date, l.end_date, l.total_days,
		        l.reason, l.status, l.applied_date,
		        e.first_name || ' ' || e.last_name as employee_name,
		        e.employee_code,
		        COALESCE(e.default_shift_id::text, '') as default_shift_id,
		        COALESCE(s.name, '') as shift_name,
		        COALESCE(s.shift_code, '') as shift_code,
		        COALESCE(d.name, '') as department_name,
		        (SELECT COUNT(*) FROM leave_approvals la
		         WHERE la.leave_id = l.id AND la.approver_role = 'team_leader' AND la.action = 'approved') as tl_approvals,
		        (SELECT COUNT(*) FROM employees WHERE role = 'team_leader' AND status = 'active' AND (department_id = e.department_id OR role = 'admin')) as total_tls,
		        l.start_time, l.end_time
		 FROM leaves l
		 JOIN employees e ON e.id = l.employee_id
		 LEFT JOIN leave_types lt ON lt.id = l.leave_type_id
		 LEFT JOIN shifts s ON s.id = e.default_shift_id
		 LEFT JOIN departments d ON d.id = e.department_id
		 WHERE ` + whereClause

	args := []interface{}{}

	if approverRole != "admin" {
		if approverDeptID == nil {
			return []models.PendingLeaveRich{}, nil
		}
		query += ` AND e.department_id = $1`
		args = append(args, *approverDeptID)
	}
	query += ` ORDER BY l.applied_date`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get pending leaves rich: %w", err)
	}
	defer rows.Close()

	var result []models.PendingLeaveRich
	for rows.Next() {
		var p models.PendingLeaveRich
		if err := rows.Scan(
			&p.ID, &p.EmployeeID, &p.LeaveTypeID, &p.LeaveTypeNameAr, &p.LeaveTypeNameEn, &p.StartDate, &p.EndDate, &p.TotalDays,
			&p.Reason, &p.Status, &p.AppliedDate,
			&p.EmployeeName, &p.EmployeeCode, &p.DefaultShiftID, &p.ShiftName, &p.ShiftCode,
			&p.DepartmentName, &p.TLApprovals, &p.TotalTLs, &p.StartTime, &p.EndTime,
		); err != nil {
			return nil, fmt.Errorf("scan pending leave rich: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *leaveRepo) Create(ctx context.Context, leave *models.Leave) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO leaves (employee_id, leave_type_id, start_date, end_date, reason, attachments, start_time, end_time)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, total_days, status, applied_date, created_at, updated_at`,
		leave.EmployeeID, leave.LeaveTypeID, leave.StartDate, leave.EndDate, leave.Reason, leave.Attachments, leave.StartTime, leave.EndTime,
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

// ── Approval tracking ──

func (r *leaveRepo) RecordApproval(ctx context.Context, leaveID, approverID uuid.UUID, approverRole, action string, notes *string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO leave_approvals (leave_id, approver_id, approver_role, action, notes)
		 VALUES ($1, $2, $3, $4, $5)`,
		leaveID, approverID, approverRole, action, notes)
	return err
}

func (r *leaveRepo) CountTLApprovals(ctx context.Context, leaveID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM leave_approvals WHERE leave_id=$1 AND approver_role='team_leader' AND action='approved'`,
		leaveID).Scan(&count)
	return count, err
}

func (r *leaveRepo) HasApproved(ctx context.Context, leaveID, approverID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM leave_approvals WHERE leave_id=$1 AND approver_id=$2`,
		leaveID, approverID).Scan(&count)
	return count > 0, err
}

func (r *leaveRepo) GetLeaveHistory(ctx context.Context, departmentID *uuid.UUID) ([]models.LeaveHistoryRow, error) {
	// Get all leaves with employee info
	rows, err := r.db.Query(ctx,
		`SELECT l.id, e.first_name || ' ' || e.last_name, e.employee_code,
		        l.leave_type_id, lt.name_ar, lt.name_en, l.start_date, l.end_date, l.total_days, l.reason,
		        l.status, l.applied_date, l.rejection_reason
		 FROM leaves l
		 JOIN employees e ON e.id = l.employee_id
		 LEFT JOIN leave_types lt ON lt.id = l.leave_type_id
		 WHERE ($1::uuid IS NULL OR e.department_id = $1)
		 ORDER BY l.applied_date DESC NULLS LAST
		 LIMIT 200`, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.LeaveHistoryRow
	for rows.Next() {
		var h models.LeaveHistoryRow
		if err := rows.Scan(&h.LeaveID, &h.EmployeeName, &h.EmployeeCode,
			&h.LeaveTypeID, &h.LeaveTypeNameAr, &h.LeaveTypeNameEn, &h.StartDate, &h.EndDate, &h.TotalDays, &h.Reason,
			&h.Status, &h.AppliedDate, &h.RejectionReason); err != nil {
			return nil, err
		}
		result = append(result, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch approvals for each leave
	for i := range result {
		apRows, err := r.db.Query(ctx,
			`SELECT e.first_name || ' ' || e.last_name, la.approver_role, la.action, la.notes, la.created_at
			 FROM leave_approvals la
			 JOIN employees e ON e.id = la.approver_id
			 WHERE la.leave_id = $1
			 ORDER BY la.created_at`, result[i].LeaveID)
		if err != nil {
			continue
		}
		for apRows.Next() {
			var d models.LeaveApprovalDetail
			if err := apRows.Scan(&d.ApproverName, &d.ApproverRole, &d.Action, &d.Notes, &d.CreatedAt); err != nil {
				break
			}
			result[i].Approvals = append(result[i].Approvals, d)
		}
		apRows.Close()
	}

	return result, nil
}
