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

// TaskRepository handles task schedules, assignments, executions, and recurring assignments.
type TaskRepository interface {
	// Task Schedules
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.TaskSchedule, error)
	GetSchedulesByType(ctx context.Context, scheduleType string) ([]models.TaskSchedule, error)
	GetSchedulesByShift(ctx context.Context, shiftID uuid.UUID) ([]models.TaskSchedule, error)
	GetSchedulesByBoard(ctx context.Context, boardID uuid.UUID) ([]models.TaskSchedule, error)
	GetAllSchedules(ctx context.Context) ([]models.TaskSchedule, error)
	GetActiveSchedules(ctx context.Context) ([]models.TaskSchedule, error)
	CreateSchedule(ctx context.Context, ts *models.TaskSchedule) error
	UpdateSchedule(ctx context.Context, ts *models.TaskSchedule) error
	ToggleScheduleActive(ctx context.Context, id uuid.UUID, isActive bool) error
	DeleteSchedule(ctx context.Context, id uuid.UUID) error

	// Board View & Tracker
	GetBoardView(ctx context.Context, boardID uuid.UUID, shiftID *uuid.UUID, fromDate *time.Time, toDate *time.Time) ([]models.BoardViewRow, error)
	GetBoardStats(ctx context.Context) ([]models.TaskBoardStats, error)
	GetBoardEligibleEmployees(ctx context.Context, shiftID *uuid.UUID, date *time.Time) ([]models.Employee, error)

	// Employee Weekly View
	GetMyWeeklyTasks(ctx context.Context, employeeID uuid.UUID, weekStart, weekEnd time.Time) ([]models.MyTaskRow, error)

	// Recurring Assignments
	CreateRecurringAssignment(ctx context.Context, ra *models.TaskRecurringAssignment) error
	DeleteRecurringAssignment(ctx context.Context, id uuid.UUID) error
	DeleteRecurringAssignmentByKey(ctx context.Context, scheduleID, employeeID uuid.UUID, dayOfWeek int) error
	GetRecurringAssignmentsByBoard(ctx context.Context, boardID uuid.UUID) ([]models.TaskRecurringAssignment, error)
	MaterializeRecurringForDateRange(ctx context.Context, boardID uuid.UUID, fromDate, toDate time.Time) error
	MaterializeAllRecurringForEmployee(ctx context.Context, employeeID uuid.UUID, fromDate, toDate time.Time) error

	// Task Assignments
	GetAssignmentsByDate(ctx context.Context, date time.Time) ([]models.TaskAssignment, error)
	GetAssignmentsByEmployee(ctx context.Context, employeeID uuid.UUID, date time.Time) ([]models.TaskAssignment, error)
	CountAssignmentsByScheduleAndDate(ctx context.Context, scheduleID uuid.UUID, date time.Time) (int, error)
	CountAssignmentsByScheduleDateAndShift(ctx context.Context, scheduleID uuid.UUID, date time.Time, shiftID uuid.UUID) (int, error)
	CreateAssignment(ctx context.Context, ta *models.TaskAssignment) error
	DeleteAssignment(ctx context.Context, id uuid.UUID) error
	SwapAssignmentsBetweenEmployees(ctx context.Context, empA, empB uuid.UUID, date time.Time) error

	// Task Executions
	GetExecutionByAssignment(ctx context.Context, assignmentID uuid.UUID) (*models.TaskExecution, error)
	CreateExecution(ctx context.Context, te *models.TaskExecution) error
	StartExecution(ctx context.Context, id uuid.UUID) error
	UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, notes *string) error
	CompleteExecution(ctx context.Context, id uuid.UUID, completionType string, notes *string) error

	// Task History (supervisor view)
	GetTaskHistory(ctx context.Context, date time.Time, boardID *uuid.UUID) ([]models.TaskHistoryRow, error)
}

type taskRepo struct {
	db *database.DB
}

func NewTaskRepository(db *database.DB) TaskRepository {
	return &taskRepo{db: db}
}

// ═══════════════════════════════════════════
// Task Schedules
// ═══════════════════════════════════════════

const scheduleColumns = `id, title, description, schedule_type, board_id, shift_id, recurrence, recurrence_days,
	max_assignees, is_active, created_by, created_at, updated_at`

func (r *taskRepo) scanSchedule(row pgx.Row) (*models.TaskSchedule, error) {
	var ts models.TaskSchedule
	err := row.Scan(&ts.ID, &ts.Title, &ts.Description, &ts.ScheduleType, &ts.BoardID, &ts.ShiftID,
		&ts.Recurrence, &ts.RecurrenceDays, &ts.MaxAssignees, &ts.IsActive,
		&ts.CreatedBy, &ts.CreatedAt, &ts.UpdatedAt)
	return &ts, err
}

func (r *taskRepo) scanSchedules(rows pgx.Rows) ([]models.TaskSchedule, error) {
	var schedules []models.TaskSchedule
	for rows.Next() {
		var ts models.TaskSchedule
		if err := rows.Scan(&ts.ID, &ts.Title, &ts.Description, &ts.ScheduleType, &ts.BoardID, &ts.ShiftID,
			&ts.Recurrence, &ts.RecurrenceDays, &ts.MaxAssignees, &ts.IsActive,
			&ts.CreatedBy, &ts.CreatedAt, &ts.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task schedule: %w", err)
		}
		schedules = append(schedules, ts)
	}
	return schedules, rows.Err()
}

func (r *taskRepo) GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.TaskSchedule, error) {
	ts, err := r.scanSchedule(r.db.QueryRow(ctx,
		`SELECT `+scheduleColumns+` FROM task_schedules WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task schedule not found")
		}
		return nil, fmt.Errorf("get task schedule: %w", err)
	}
	return ts, nil
}

func (r *taskRepo) GetSchedulesByType(ctx context.Context, scheduleType string) ([]models.TaskSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+scheduleColumns+` FROM task_schedules WHERE schedule_type = $1 AND is_active = true ORDER BY title`, scheduleType)
	if err != nil {
		return nil, fmt.Errorf("get schedules by type: %w", err)
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

func (r *taskRepo) GetSchedulesByShift(ctx context.Context, shiftID uuid.UUID) ([]models.TaskSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+scheduleColumns+` FROM task_schedules WHERE shift_id = $1 AND is_active = true ORDER BY title`, shiftID)
	if err != nil {
		return nil, fmt.Errorf("get schedules by shift: %w", err)
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

func (r *taskRepo) GetSchedulesByBoard(ctx context.Context, boardID uuid.UUID) ([]models.TaskSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+scheduleColumns+` FROM task_schedules WHERE board_id = $1 AND is_active = true ORDER BY title`, boardID)
	if err != nil {
		return nil, fmt.Errorf("get schedules by board: %w", err)
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

func (r *taskRepo) GetActiveSchedules(ctx context.Context) ([]models.TaskSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+scheduleColumns+` FROM task_schedules WHERE is_active = true ORDER BY schedule_type, title`)
	if err != nil {
		return nil, fmt.Errorf("get active schedules: %w", err)
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

func (r *taskRepo) GetAllSchedules(ctx context.Context) ([]models.TaskSchedule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+scheduleColumns+` FROM task_schedules ORDER BY schedule_type, title`)
	if err != nil {
		return nil, fmt.Errorf("get all schedules: %w", err)
	}
	defer rows.Close()
	return r.scanSchedules(rows)
}

func (r *taskRepo) CreateSchedule(ctx context.Context, ts *models.TaskSchedule) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO task_schedules (title, description, schedule_type, board_id, shift_id, recurrence,
			recurrence_days, max_assignees, is_active, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, created_at, updated_at`,
		ts.Title, ts.Description, ts.ScheduleType, ts.BoardID, ts.ShiftID, ts.Recurrence,
		ts.RecurrenceDays, ts.MaxAssignees, ts.IsActive, ts.CreatedBy,
	).Scan(&ts.ID, &ts.CreatedAt, &ts.UpdatedAt)
}

func (r *taskRepo) UpdateSchedule(ctx context.Context, ts *models.TaskSchedule) error {
	_, err := r.db.Exec(ctx,
		`UPDATE task_schedules SET title=$1, description=$2, schedule_type=$3, board_id=$4, shift_id=$5,
			recurrence=$6, recurrence_days=$7, max_assignees=$8, is_active=$9, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$10`,
		ts.Title, ts.Description, ts.ScheduleType, ts.BoardID, ts.ShiftID, ts.Recurrence,
		ts.RecurrenceDays, ts.MaxAssignees, ts.IsActive, ts.ID)
	return err
}

func (r *taskRepo) ToggleScheduleActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	_, err := r.db.Exec(ctx,
		`UPDATE task_schedules SET is_active=$1, updated_at=CURRENT_TIMESTAMP WHERE id=$2`, isActive, id)
	return err
}

func (r *taskRepo) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM task_schedules WHERE id=$1`, id)
	return err
}

// ═══════════════════════════════════════════
// Board View & Tracker
// ═══════════════════════════════════════════

func (r *taskRepo) GetBoardView(ctx context.Context, boardID uuid.UUID, shiftID *uuid.UUID, fromDate *time.Time, toDate *time.Time) ([]models.BoardViewRow, error) {
	query := `
		SELECT
			e.id AS employee_id,
			CONCAT(e.first_name, ' ', e.last_name) AS employee_name,
			e.employee_code,
			EXTRACT(DOW FROM ta.assigned_date)::int AS day_of_week,
			ta.assigned_date,
			ts.id AS task_id,
			ts.title AS task_title,
			ta.id AS assignment_id,
			te.id AS execution_id,
			COALESCE(te.status, 'pending') AS status,
			te.started_at,
			te.completed_at
		FROM task_schedules ts
		JOIN task_assignments ta ON ta.schedule_id = ts.id
		JOIN employees e ON e.id = ta.employee_id
		LEFT JOIN task_executions te ON te.assignment_id = ta.id
		WHERE ts.board_id = $1
		  AND ts.is_active = true
		  AND e.role IN ('employee', 'team_leader')
		  AND e.status = 'active'`

	args := []interface{}{boardID}

	argIdx := 2
	if shiftID != nil {
		// When filtering by shift, also include board tasks without a bound shift.
		// Unbound tasks are considered visible across all shifts.
		query += fmt.Sprintf(` AND (ts.shift_id = $%d OR ts.shift_id IS NULL)`, argIdx)
		// Keep employee rows scoped to the selected shift while filtering.
		query += fmt.Sprintf(` AND e.default_shift_id = $%d`, argIdx)
		args = append(args, *shiftID)
		argIdx++
	}
	if fromDate != nil {
		query += fmt.Sprintf(` AND ta.assigned_date >= $%d`, argIdx)
		args = append(args, *fromDate)
		argIdx++
	}
	if toDate != nil {
		query += fmt.Sprintf(` AND ta.assigned_date <= $%d`, argIdx)
		args = append(args, *toDate)
		argIdx++
	}

	query += ` ORDER BY e.first_name, e.last_name, ta.assigned_date`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get board view: %w", err)
	}
	defer rows.Close()

	var results []models.BoardViewRow
	for rows.Next() {
		var row models.BoardViewRow
		if err := rows.Scan(&row.EmployeeID, &row.EmployeeName, &row.EmployeeCode,
			&row.DayOfWeek, &row.AssignedDate, &row.TaskID, &row.TaskTitle,
			&row.AssignmentID, &row.ExecutionID, &row.Status,
			&row.StartedAt, &row.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan board view row: %w", err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func (r *taskRepo) GetBoardStats(ctx context.Context) ([]models.TaskBoardStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			tb.id,
			tb.name,
			COUNT(ta.id) AS total_assigned,
			COUNT(ta.id) FILTER (WHERE COALESCE(te.status, 'pending') = 'pending') AS total_pending,
			COUNT(ta.id) FILTER (WHERE te.status = 'in_progress') AS total_in_progress,
			COUNT(ta.id) FILTER (WHERE te.status = 'completed') AS total_completed,
			CASE WHEN COUNT(ta.id) > 0
				THEN ROUND(COUNT(ta.id) FILTER (WHERE te.status = 'completed')::numeric / COUNT(ta.id) * 100, 1)
				ELSE 0 END AS completion_pct
		FROM task_boards tb
		LEFT JOIN task_schedules ts ON ts.board_id = tb.id AND ts.is_active = true
		LEFT JOIN task_assignments ta ON ta.schedule_id = ts.id
		LEFT JOIN task_executions te ON te.assignment_id = ta.id
		WHERE tb.is_active = true
		GROUP BY tb.id, tb.name
		ORDER BY tb.name`)
	if err != nil {
		return nil, fmt.Errorf("get board stats: %w", err)
	}
	defer rows.Close()

	var stats []models.TaskBoardStats
	for rows.Next() {
		var s models.TaskBoardStats
		if err := rows.Scan(&s.BoardID, &s.BoardName, &s.TotalAssigned,
			&s.TotalPending, &s.TotalInProgress, &s.TotalCompleted, &s.CompletionPct); err != nil {
			return nil, fmt.Errorf("scan board stats: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetBoardEligibleEmployees returns active employees eligible for the board tracker.
// Managers and admins are excluded (team leaders are included).
// Optionally filters by default_shift_id.
// When a date is provided, employees on approved leave or with an off-day are excluded.
func (r *taskRepo) GetBoardEligibleEmployees(ctx context.Context, shiftID *uuid.UUID, date *time.Time) ([]models.Employee, error) {
	query := `
		SELECT id, employee_code, first_name, last_name, gender, phone, email,
			hire_date, role, department_id, position, default_shift_id,
			weekly_off_days, can_cover_night_shift, status, profile_image,
			last_login, created_at, updated_at, created_by
		FROM employees
		WHERE role = 'employee'
		  AND status = 'active'`

	args := []interface{}{}
	argIdx := 1

	if shiftID != nil {
		query += fmt.Sprintf(` AND default_shift_id = $%d`, argIdx)
		args = append(args, *shiftID)
		argIdx++
	}

	if date != nil {
		// Exclude employees on approved leave that covers this date
		query += fmt.Sprintf(` AND NOT EXISTS (
			SELECT 1 FROM leaves l
			WHERE l.employee_id = employees.id
			  AND l.status IN ('approved_by_manager', 'approved_by_team_leader')
			  AND $%d::date BETWEEN l.start_date AND l.end_date
		)`, argIdx)
		args = append(args, *date)
		argIdx++

		// Exclude employees whose shift record for this date is off/leave/vacation
		query += fmt.Sprintf(` AND NOT EXISTS (
			SELECT 1 FROM employee_shifts es
			WHERE es.employee_id = employees.id
			  AND es.shift_date = $%d::date
			  AND es.shift_status IN ('off', 'leave', 'vacation')
		)`, argIdx)
		args = append(args, *date)
		argIdx++
	}

	query += ` ORDER BY first_name, last_name`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get eligible employees: %w", err)
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var e models.Employee
		if err := rows.Scan(&e.ID, &e.EmployeeCode, &e.FirstName, &e.LastName, &e.Gender,
			&e.Phone, &e.Email, &e.HireDate, &e.Role, &e.DepartmentID, &e.Position,
			&e.DefaultShiftID, &e.WeeklyOffDays, &e.CanCoverNightShift, &e.Status,
			&e.ProfileImage, &e.LastLogin, &e.CreatedAt, &e.UpdatedAt, &e.CreatedBy); err != nil {
			return nil, fmt.Errorf("scan eligible employee: %w", err)
		}
		employees = append(employees, e)
	}
	return employees, rows.Err()
}

// ═══════════════════════════════════════════
// Employee Weekly Tasks
// ═══════════════════════════════════════════

func (r *taskRepo) GetMyWeeklyTasks(ctx context.Context, employeeID uuid.UUID, weekStart, weekEnd time.Time) ([]models.MyTaskRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			ta.id AS assignment_id,
			ta.assigned_date,
			ts.title AS task_title,
			ts.description AS task_description,
			tb.name AS board_name,
			sh.name AS shift_name,
			sh.shift_code,
			sh.color_code AS shift_color,
			te.id AS execution_id,
			COALESCE(te.status, 'pending') AS status,
			te.completion_type,
			te.started_at,
			te.completed_at,
			te.notes
		FROM task_assignments ta
		JOIN task_schedules ts ON ts.id = ta.schedule_id
		LEFT JOIN task_boards tb ON tb.id = ts.board_id
		LEFT JOIN shifts sh ON sh.id = ts.shift_id
		LEFT JOIN task_executions te ON te.assignment_id = ta.id
		WHERE ta.employee_id = $1
		  AND ta.assigned_date >= $2
		  AND ta.assigned_date <= $3
		ORDER BY ta.assigned_date, ts.title`,
		employeeID, weekStart, weekEnd)
	if err != nil {
		return nil, fmt.Errorf("get weekly tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.MyTaskRow
	for rows.Next() {
		var t models.MyTaskRow
		if err := rows.Scan(&t.AssignmentID, &t.AssignedDate, &t.TaskTitle, &t.TaskDesc,
			&t.BoardName, &t.ShiftName, &t.ShiftCode, &t.ShiftColor,
			&t.ExecutionID, &t.Status, &t.CompletionType, &t.StartedAt, &t.CompletedAt, &t.Notes); err != nil {
			return nil, fmt.Errorf("scan weekly task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// ═══════════════════════════════════════════
// Task Assignments
// ═══════════════════════════════════════════

func (r *taskRepo) GetAssignmentsByDate(ctx context.Context, date time.Time) ([]models.TaskAssignment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ta.id, ta.schedule_id, ta.employee_id, ta.assigned_date, ta.assigned_by, ta.created_at
		 FROM task_assignments ta
		 JOIN employees e ON e.id = ta.employee_id
		 WHERE ta.assigned_date = $1
		   AND e.role IN ('employee', 'team_leader')
		   AND e.status = 'active'
		 ORDER BY ta.created_at`, date)
	if err != nil {
		return nil, fmt.Errorf("get assignments by date: %w", err)
	}
	defer rows.Close()

	var assignments []models.TaskAssignment
	for rows.Next() {
		var ta models.TaskAssignment
		if err := rows.Scan(&ta.ID, &ta.ScheduleID, &ta.EmployeeID, &ta.AssignedDate,
			&ta.AssignedBy, &ta.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		assignments = append(assignments, ta)
	}
	return assignments, rows.Err()
}

func (r *taskRepo) GetAssignmentsByEmployee(ctx context.Context, employeeID uuid.UUID, date time.Time) ([]models.TaskAssignment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ta.id, ta.schedule_id, ta.employee_id, ta.assigned_date, ta.assigned_by, ta.created_at
		 FROM task_assignments ta
		 JOIN employees e ON e.id = ta.employee_id
		 WHERE ta.employee_id = $1
		   AND ta.assigned_date = $2
		   AND e.role IN ('employee', 'team_leader')
		   AND e.status = 'active'
		 ORDER BY ta.created_at`, employeeID, date)
	if err != nil {
		return nil, fmt.Errorf("get assignments by employee: %w", err)
	}
	defer rows.Close()

	var assignments []models.TaskAssignment
	for rows.Next() {
		var ta models.TaskAssignment
		if err := rows.Scan(&ta.ID, &ta.ScheduleID, &ta.EmployeeID, &ta.AssignedDate,
			&ta.AssignedBy, &ta.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		assignments = append(assignments, ta)
	}
	return assignments, rows.Err()
}

func (r *taskRepo) CountAssignmentsByScheduleAndDate(ctx context.Context, scheduleID uuid.UUID, date time.Time) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM task_assignments WHERE schedule_id = $1 AND assigned_date = $2`,
		scheduleID, date,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count assignments by schedule/date: %w", err)
	}
	return count, nil
}

func (r *taskRepo) CountAssignmentsByScheduleDateAndShift(ctx context.Context, scheduleID uuid.UUID, date time.Time, shiftID uuid.UUID) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM task_assignments ta
		   JOIN employees e ON e.id = ta.employee_id
		  WHERE ta.schedule_id = $1
		    AND ta.assigned_date = $2
		    AND e.default_shift_id = $3`,
		scheduleID, date, shiftID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count assignments by schedule/date/shift: %w", err)
	}
	return count, nil
}

func (r *taskRepo) CreateAssignment(ctx context.Context, ta *models.TaskAssignment) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO task_assignments (schedule_id, employee_id, assigned_date, assigned_by)
		 VALUES ($1,$2,$3,$4) RETURNING id, created_at`,
		ta.ScheduleID, ta.EmployeeID, ta.AssignedDate, ta.AssignedBy,
	).Scan(&ta.ID, &ta.CreatedAt)
}

func (r *taskRepo) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	// Also delete associated execution
	_, _ = r.db.Exec(ctx, `DELETE FROM task_executions WHERE assignment_id=$1`, id)
	_, err := r.db.Exec(ctx, `DELETE FROM task_assignments WHERE id=$1`, id)
	return err
}

// SwapAssignmentsBetweenEmployees swaps task assignments between two employees on a specific date.
// Employee A's assignments go to B, and B's assignments go to A, for that day only.
func (r *taskRepo) SwapAssignmentsBetweenEmployees(ctx context.Context, empA, empB uuid.UUID, date time.Time) error {
	// Swap in one statement (no temp placeholder needed).
	// Since both target employee IDs already exist, FK constraints remain valid.
	// Postgres validates unique constraints against the final values of the statement.
	_, err := r.db.Exec(ctx, `
		UPDATE task_assignments
		SET employee_id = CASE
			WHEN employee_id = $1 THEN $2
			WHEN employee_id = $2 THEN $1
			ELSE employee_id
		END
		WHERE assigned_date = $3
		  AND employee_id IN ($1, $2)
	`, empA, empB, date)
	if err != nil {
		return fmt.Errorf("swap assignments between employees: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════
// Task Executions
// ═══════════════════════════════════════════

func (r *taskRepo) GetExecutionByAssignment(ctx context.Context, assignmentID uuid.UUID) (*models.TaskExecution, error) {
	var te models.TaskExecution
	err := r.db.QueryRow(ctx,
		`SELECT id, assignment_id, status, started_at, completed_at, notes, attachments, created_at, updated_at
		 FROM task_executions WHERE assignment_id = $1`, assignmentID,
	).Scan(&te.ID, &te.AssignmentID, &te.Status, &te.StartedAt, &te.CompletedAt, &te.Notes,
		&te.Attachments, &te.CreatedAt, &te.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No execution yet — not an error
		}
		return nil, fmt.Errorf("get execution: %w", err)
	}
	return &te, nil
}

func (r *taskRepo) CreateExecution(ctx context.Context, te *models.TaskExecution) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO task_executions (assignment_id, status) VALUES ($1,$2) RETURNING id, created_at, updated_at`,
		te.AssignmentID, te.Status,
	).Scan(&te.ID, &te.CreatedAt, &te.UpdatedAt)
}

func (r *taskRepo) StartExecution(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE task_executions SET status='in_progress', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=$1`,
		id)
	return err
}

func (r *taskRepo) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, notes *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE task_executions SET status=$1, notes=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3`,
		status, notes, id)
	return err
}

func (r *taskRepo) CompleteExecution(ctx context.Context, id uuid.UUID, completionType string, notes *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE task_executions SET status='completed', completion_type=$1, completed_at=CURRENT_TIMESTAMP, notes=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3`,
		completionType, notes, id)
	return err
}

// GetTaskHistory returns all completed/in_progress tasks for a given date,
// optionally filtered by board. Used by team leaders and managers.
func (r *taskRepo) GetTaskHistory(ctx context.Context, date time.Time, boardID *uuid.UUID) ([]models.TaskHistoryRow, error) {
	query := `
		SELECT
			ta.id AS assignment_id,
			te.id AS execution_id,
			ta.assigned_date,
			ts.title AS task_title,
			ts.description AS task_description,
			tb.name AS board_name,
			e.id AS employee_id,
			e.first_name || ' ' || e.last_name AS employee_name,
			e.employee_code,
			te.status,
			te.completion_type,
			te.started_at,
			te.completed_at,
			te.notes
		FROM task_assignments ta
		JOIN task_schedules ts ON ts.id = ta.schedule_id
		JOIN employees e ON e.id = ta.employee_id
		JOIN task_executions te ON te.assignment_id = ta.id
		LEFT JOIN task_boards tb ON tb.id = ts.board_id
		WHERE ta.assigned_date = $1
	`
	args := []interface{}{date}
	if boardID != nil {
		query += " AND ts.board_id = $2"
		args = append(args, *boardID)
	}
	query += " ORDER BY te.completed_at DESC NULLS LAST, e.first_name"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get task history: %w", err)
	}
	defer rows.Close()

	var results []models.TaskHistoryRow
	for rows.Next() {
		var row models.TaskHistoryRow
		if err := rows.Scan(
			&row.AssignmentID, &row.ExecutionID, &row.AssignedDate,
			&row.TaskTitle, &row.TaskDesc, &row.BoardName,
			&row.EmployeeID, &row.EmployeeName, &row.EmployeeCode,
			&row.Status, &row.CompletionType, &row.StartedAt, &row.CompletedAt, &row.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan task history row: %w", err)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// ═══════════════════════════════════════════
// Recurring Assignments
// ═══════════════════════════════════════════

func (r *taskRepo) CreateRecurringAssignment(ctx context.Context, ra *models.TaskRecurringAssignment) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO task_recurring_assignments (schedule_id, employee_id, day_of_week, assigned_by)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (schedule_id, employee_id, day_of_week) DO NOTHING
		 RETURNING id, created_at`,
		ra.ScheduleID, ra.EmployeeID, ra.DayOfWeek, ra.AssignedBy,
	).Scan(&ra.ID, &ra.CreatedAt)
}

func (r *taskRepo) DeleteRecurringAssignment(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM task_recurring_assignments WHERE id=$1`, id)
	return err
}

func (r *taskRepo) DeleteRecurringAssignmentByKey(ctx context.Context, scheduleID, employeeID uuid.UUID, dayOfWeek int) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM task_recurring_assignments WHERE schedule_id=$1 AND employee_id=$2 AND day_of_week=$3`,
		scheduleID, employeeID, dayOfWeek)
	return err
}

func (r *taskRepo) GetRecurringAssignmentsByBoard(ctx context.Context, boardID uuid.UUID) ([]models.TaskRecurringAssignment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ra.id, ra.schedule_id, ra.employee_id, ra.day_of_week, ra.assigned_by, ra.created_at
		 FROM task_recurring_assignments ra
		 JOIN task_schedules ts ON ts.id = ra.schedule_id
		 WHERE ts.board_id = $1 AND ts.is_active = true
		 ORDER BY ra.day_of_week, ra.employee_id`, boardID)
	if err != nil {
		return nil, fmt.Errorf("get recurring assignments by board: %w", err)
	}
	defer rows.Close()

	var results []models.TaskRecurringAssignment
	for rows.Next() {
		var ra models.TaskRecurringAssignment
		if err := rows.Scan(&ra.ID, &ra.ScheduleID, &ra.EmployeeID, &ra.DayOfWeek, &ra.AssignedBy, &ra.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recurring assignment: %w", err)
		}
		results = append(results, ra)
	}
	return results, rows.Err()
}

// MaterializeRecurringForDateRange creates task_assignments and task_executions
// for all recurring assignments of a board within the given date range,
// skipping any that already exist (idempotent).
func (r *taskRepo) MaterializeRecurringForDateRange(ctx context.Context, boardID uuid.UUID, fromDate, toDate time.Time) error {
	// This query inserts task_assignments for each recurring assignment
	// where the date falls within the range and matches the day_of_week,
	// and the assignment doesn't already exist.
	_, err := r.db.Exec(ctx, `
		WITH date_series AS (
			SELECT d::date AS assigned_date, EXTRACT(DOW FROM d)::int AS dow
			FROM generate_series($2::date, $3::date, '1 day'::interval) AS d
		),
		new_assignments AS (
			INSERT INTO task_assignments (schedule_id, employee_id, assigned_date, assigned_by)
			SELECT ra.schedule_id, ra.employee_id, ds.assigned_date, ra.assigned_by
			FROM task_recurring_assignments ra
			JOIN task_schedules ts ON ts.id = ra.schedule_id
			CROSS JOIN date_series ds
			WHERE ts.board_id = $1
			  AND ts.is_active = true
			  AND ds.dow = ra.day_of_week
			ON CONFLICT (schedule_id, employee_id, assigned_date) DO NOTHING
			RETURNING id
		)
		INSERT INTO task_executions (assignment_id, status)
		SELECT id, 'pending' FROM new_assignments
	`, boardID, fromDate, toDate)
	if err != nil {
		return fmt.Errorf("materialize recurring assignments: %w", err)
	}
	return nil
}

func (r *taskRepo) MaterializeAllRecurringForEmployee(ctx context.Context, employeeID uuid.UUID, fromDate, toDate time.Time) error {
	_, err := r.db.Exec(ctx, `
		WITH date_series AS (
			SELECT d::date AS assigned_date, EXTRACT(DOW FROM d)::int AS dow
			FROM generate_series($2::date, $3::date, '1 day'::interval) AS d
		),
		new_assignments AS (
			INSERT INTO task_assignments (schedule_id, employee_id, assigned_date, assigned_by)
			SELECT ra.schedule_id, ra.employee_id, ds.assigned_date, ra.assigned_by
			FROM task_recurring_assignments ra
			JOIN task_schedules ts ON ts.id = ra.schedule_id
			CROSS JOIN date_series ds
			WHERE ra.employee_id = $1
			  AND ts.is_active = true
			  AND ds.dow = ra.day_of_week
			ON CONFLICT (schedule_id, employee_id, assigned_date) DO NOTHING
			RETURNING id
		)
		INSERT INTO task_executions (assignment_id, status)
		SELECT id, 'pending' FROM new_assignments
	`, employeeID, fromDate, toDate)
	if err != nil {
		return fmt.Errorf("materialize all recurring for employee: %w", err)
	}
	return nil
}
