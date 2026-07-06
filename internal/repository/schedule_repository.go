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

// ScheduleRepository handles schedule templates, weekly schedules, and employee shifts.
type ScheduleRepository interface {
	// Schedule Templates
	GetTemplatesByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ScheduleTemplate, error)
	CreateTemplate(ctx context.Context, t *models.ScheduleTemplate) error
	UpdateTemplate(ctx context.Context, t *models.ScheduleTemplate) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	// UpsertTemplateForDay ensures a permanent off/working template exists for the
	// given employee + day-of-week. Used by SetEmployeeShift so manual changes
	// carry forward into every future generated weekly schedule automatically.
	UpsertTemplateForDay(ctx context.Context, employeeID uuid.UUID, dayOfWeek int, isOff bool, shiftID *uuid.UUID) error

	// Weekly Schedules
	GetWeeklySchedule(ctx context.Context, weekStart time.Time) (*models.WeeklySchedule, error)
	GetWeeklyScheduleByID(ctx context.Context, id uuid.UUID) (*models.WeeklySchedule, error)
	CreateWeeklySchedule(ctx context.Context, ws *models.WeeklySchedule) error
	UpdateWeeklyScheduleStatus(ctx context.Context, id uuid.UUID, status string, publishedBy *uuid.UUID) error

	// Employee Shifts (daily assignments)
	GetEmployeeShiftsByDate(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.EmployeeShift, error)
	GetEmployeeShiftsByEmployee(ctx context.Context, employeeID uuid.UUID, from, to time.Time) ([]models.EmployeeShift, error)
	GetEmployeeShiftsInRange(ctx context.Context, from, to time.Time) ([]models.EmployeeShift, error)
	GetDepartmentShiftsInRange(ctx context.Context, from, to time.Time, departmentID uuid.UUID) ([]models.EmployeeShiftExtended, error)
	GetEmployeeShift(ctx context.Context, employeeID uuid.UUID, date time.Time) (*models.EmployeeShift, error)
	GetEmployeeShiftByID(ctx context.Context, id uuid.UUID) (*models.EmployeeShift, error)
	CreateEmployeeShift(ctx context.Context, es *models.EmployeeShift) error
	UpdateEmployeeShift(ctx context.Context, es *models.EmployeeShift) error
	UpdateShiftStatus(ctx context.Context, id uuid.UUID, status string, reason *string) error
	AssignReplacement(ctx context.Context, id uuid.UUID, replacementEmployeeID uuid.UUID, approvedBy uuid.UUID) error
	CheckIn(ctx context.Context, id uuid.UUID) error
	CheckOut(ctx context.Context, id uuid.UUID) error
	UpsertEmployeeShift(ctx context.Context, es *models.EmployeeShift) error
	DeleteEmployeeShift(ctx context.Context, id uuid.UUID) error

	// Smart Replacement: employees who were off/on-leave the previous day
	GetAvailableReplacements(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.Employee, error)
	GetEligibleAssignees(ctx context.Context, shiftID uuid.UUID, date time.Time) ([]models.Employee, error)
	GetSwapEligibleEmployees(ctx context.Context, departmentID uuid.UUID, excludeEmployeeID uuid.UUID, date time.Time) ([]models.SwapEligibleEmployee, error)
	GetShiftCoveragePreview(ctx context.Context, shiftID uuid.UUID, date time.Time) (*models.ShiftCoverage, error)
}

type scheduleRepo struct {
	db *database.DB
}

func NewScheduleRepository(db *database.DB) ScheduleRepository {
	return &scheduleRepo{db: db}
}

// --- Schedule Templates ---

func (r *scheduleRepo) GetTemplatesByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ScheduleTemplate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, employee_id, day_of_week, shift_id, is_off, valid_from, valid_to, created_at, updated_at
		 FROM schedule_templates WHERE employee_id = $1 ORDER BY day_of_week`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get templates by employee: %w", err)
	}
	defer rows.Close()

	var templates []models.ScheduleTemplate
	for rows.Next() {
		var t models.ScheduleTemplate
		if err := rows.Scan(&t.ID, &t.EmployeeID, &t.DayOfWeek, &t.ShiftID, &t.IsOff,
			&t.ValidFrom, &t.ValidTo, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *scheduleRepo) CreateTemplate(ctx context.Context, t *models.ScheduleTemplate) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO schedule_templates (employee_id, day_of_week, shift_id, is_off, valid_from, valid_to)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at, updated_at`,
		t.EmployeeID, t.DayOfWeek, t.ShiftID, t.IsOff, t.ValidFrom, t.ValidTo,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *scheduleRepo) UpdateTemplate(ctx context.Context, t *models.ScheduleTemplate) error {
	_, err := r.db.Exec(ctx,
		`UPDATE schedule_templates SET shift_id=$1, is_off=$2, valid_from=$3, valid_to=$4, updated_at=CURRENT_TIMESTAMP
		 WHERE id=$5`,
		t.ShiftID, t.IsOff, t.ValidFrom, t.ValidTo, t.ID)
	return err
}

func (r *scheduleRepo) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM schedule_templates WHERE id=$1`, id)
	return err
}

// UpsertTemplateForDay sets a permanent off/working template for an employee day.
// It updates the existing row if one exists for (employee_id, day_of_week),
// otherwise inserts a new one with valid_to=NULL (permanent).
func (r *scheduleRepo) UpsertTemplateForDay(
	ctx context.Context,
	employeeID uuid.UUID,
	dayOfWeek int,
	isOff bool,
	shiftID *uuid.UUID,
) error {
	// Attempt to update existing template for this employee + day
	tag, err := r.db.Exec(ctx,
		`UPDATE schedule_templates
		 SET is_off=$3, shift_id=$4, valid_to=NULL, updated_at=CURRENT_TIMESTAMP
		 WHERE employee_id=$1 AND day_of_week=$2`,
		employeeID, dayOfWeek, isOff, shiftID)
	if err != nil {
		return fmt.Errorf("upsert template (update): %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil // updated in place
	}
	// No existing row — insert a new permanent template
	_, err = r.db.Exec(ctx,
		`INSERT INTO schedule_templates (employee_id, day_of_week, shift_id, is_off, valid_from, valid_to)
		 VALUES ($1,$2,$3,$4,CURRENT_TIMESTAMP,NULL)`,
		employeeID, dayOfWeek, shiftID, isOff)
	if err != nil {
		return fmt.Errorf("upsert template (insert): %w", err)
	}
	return nil
}

// --- Weekly Schedules ---

func (r *scheduleRepo) GetWeeklySchedule(ctx context.Context, weekStart time.Time) (*models.WeeklySchedule, error) {
	var ws models.WeeklySchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, week_start_date, week_end_date, template_id, status, published_by, published_at, notes, created_at
		 FROM weekly_schedule WHERE week_start_date = $1::date
		 ORDER BY created_at DESC LIMIT 1`, weekStart,
	).Scan(&ws.ID, &ws.WeekStartDate, &ws.WeekEndDate, &ws.TemplateID, &ws.Status,
		&ws.PublishedBy, &ws.PublishedAt, &ws.Notes, &ws.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("weekly schedule not found: %w", err)
		}
		return nil, fmt.Errorf("get weekly schedule: %w", err)
	}
	return &ws, nil
}

func (r *scheduleRepo) GetWeeklyScheduleByID(ctx context.Context, id uuid.UUID) (*models.WeeklySchedule, error) {
	var ws models.WeeklySchedule
	err := r.db.QueryRow(ctx,
		`SELECT id, week_start_date, week_end_date, template_id, status, published_by, published_at, notes, created_at
		 FROM weekly_schedule WHERE id = $1`, id,
	).Scan(&ws.ID, &ws.WeekStartDate, &ws.WeekEndDate, &ws.TemplateID, &ws.Status,
		&ws.PublishedBy, &ws.PublishedAt, &ws.Notes, &ws.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("weekly schedule not found: %w", err)
		}
		return nil, fmt.Errorf("get weekly schedule by id: %w", err)
	}
	return &ws, nil
}

func (r *scheduleRepo) CreateWeeklySchedule(ctx context.Context, ws *models.WeeklySchedule) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO weekly_schedule (week_start_date, week_end_date, template_id, status, published_by, notes)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at`,
		ws.WeekStartDate, ws.WeekEndDate, ws.TemplateID, ws.Status, ws.PublishedBy, ws.Notes,
	).Scan(&ws.ID, &ws.CreatedAt)
}

func (r *scheduleRepo) UpdateWeeklyScheduleStatus(ctx context.Context, id uuid.UUID, status string, publishedBy *uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE weekly_schedule SET status=$1, published_by=$2, published_at=CURRENT_TIMESTAMP WHERE id=$3`,
		status, publishedBy, id)
	return err
}

// --- Employee Shifts ---

func (r *scheduleRepo) scanEmployeeShifts(rows pgx.Rows) ([]models.EmployeeShift, error) {
	var shifts []models.EmployeeShift
	for rows.Next() {
		var es models.EmployeeShift
		if err := rows.Scan(
			&es.ID, &es.ScheduleID, &es.EmployeeID, &es.ShiftID, &es.ShiftDate,
			&es.ShiftStatus, &es.LeaveReason, &es.IsReplacement, &es.ReplacedEmployeeID,
			&es.ReplacementApprovedBy, &es.CheckInTime, &es.CheckOutTime,
			&es.ActualWorkedHours, &es.OvertimeHours, &es.CreatedAt, &es.UpdatedAt, &es.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan employee shift: %w", err)
		}
		shifts = append(shifts, es)
	}
	return shifts, rows.Err()
}

const employeeShiftColumns = `id, schedule_id, employee_id, shift_id, shift_date, shift_status,
	leave_reason, is_replacement, replaced_employee_id, replacement_approved_by,
	check_in_time, check_out_time, actual_worked_hours, overtime_hours,
	created_at, updated_at, created_by`

func (r *scheduleRepo) GetEmployeeShiftsByDate(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.EmployeeShift, error) {
	query := `SELECT es.id, es.schedule_id, es.employee_id, es.shift_id, es.shift_date, es.shift_status,
		es.leave_reason, es.is_replacement, es.replaced_employee_id, es.replacement_approved_by,
		es.check_in_time, es.check_out_time, es.actual_worked_hours, es.overtime_hours,
		es.created_at, es.updated_at, es.created_by
		FROM employee_shifts es
		JOIN employees e ON e.id = es.employee_id
		WHERE es.shift_date = $1`
	args := []interface{}{date}

	if departmentID != nil {
		query += ` AND e.department_id = $2`
		args = append(args, *departmentID)
	}
	query += ` ORDER BY es.employee_id`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get shifts by date: %w", err)
	}
	defer rows.Close()
	return r.scanEmployeeShifts(rows)
}

func (r *scheduleRepo) GetEmployeeShiftsByEmployee(ctx context.Context, employeeID uuid.UUID, from, to time.Time) ([]models.EmployeeShift, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+employeeShiftColumns+` FROM employee_shifts
		 WHERE employee_id = $1 AND shift_date BETWEEN $2 AND $3 ORDER BY shift_date`, employeeID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get shifts by employee: %w", err)
	}
	defer rows.Close()
	return r.scanEmployeeShifts(rows)
}

func (r *scheduleRepo) GetEmployeeShiftsInRange(ctx context.Context, from, to time.Time) ([]models.EmployeeShift, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+employeeShiftColumns+` FROM employee_shifts
		 WHERE shift_date BETWEEN $1 AND $2 ORDER BY employee_id, shift_date`, from, to)
	if err != nil {
		return nil, fmt.Errorf("get shifts in range: %w", err)
	}
	defer rows.Close()
	return r.scanEmployeeShifts(rows)
}

func (r *scheduleRepo) GetDepartmentShiftsInRange(ctx context.Context, from, to time.Time, departmentID uuid.UUID) ([]models.EmployeeShiftExtended, error) {
	query := `SELECT es.id, es.schedule_id, es.employee_id, es.shift_id, es.shift_date, es.shift_status,
		es.leave_reason, es.is_replacement, es.replaced_employee_id, es.replacement_approved_by,
		es.check_in_time, es.check_out_time, es.actual_worked_hours, es.overtime_hours,
		es.created_at, es.updated_at, es.created_by,
		e.first_name, e.last_name, e.role, e.default_shift_id
		FROM employee_shifts es
		JOIN employees e ON e.id = es.employee_id
		WHERE es.shift_date BETWEEN $1 AND $2
		  AND e.department_id = $3
		ORDER BY e.first_name, e.last_name, es.shift_date`
		
	rows, err := r.db.Query(ctx, query, from, to, departmentID)
	if err != nil {
		return nil, fmt.Errorf("get department shifts in range: %w", err)
	}
	defer rows.Close()

	var shifts []models.EmployeeShiftExtended
	for rows.Next() {
		var es models.EmployeeShiftExtended
		if err := rows.Scan(
			&es.ID, &es.ScheduleID, &es.EmployeeID, &es.ShiftID, &es.ShiftDate,
			&es.ShiftStatus, &es.LeaveReason, &es.IsReplacement, &es.ReplacedEmployeeID,
			&es.ReplacementApprovedBy, &es.CheckInTime, &es.CheckOutTime,
			&es.ActualWorkedHours, &es.OvertimeHours, &es.CreatedAt, &es.UpdatedAt, &es.CreatedBy,
			&es.FirstName, &es.LastName, &es.EmployeeRole, &es.DefaultShiftID,
		); err != nil {
			return nil, fmt.Errorf("scan department shift: %w", err)
		}
		shifts = append(shifts, es)
	}
	return shifts, rows.Err()
}

func (r *scheduleRepo) GetEmployeeShift(ctx context.Context, employeeID uuid.UUID, date time.Time) (*models.EmployeeShift, error) {
	var es models.EmployeeShift
	err := r.db.QueryRow(ctx,
		`SELECT `+employeeShiftColumns+` FROM employee_shifts WHERE employee_id = $1 AND shift_date = $2`,
		employeeID, date,
	).Scan(
		&es.ID, &es.ScheduleID, &es.EmployeeID, &es.ShiftID, &es.ShiftDate,
		&es.ShiftStatus, &es.LeaveReason, &es.IsReplacement, &es.ReplacedEmployeeID,
		&es.ReplacementApprovedBy, &es.CheckInTime, &es.CheckOutTime,
		&es.ActualWorkedHours, &es.OvertimeHours, &es.CreatedAt, &es.UpdatedAt, &es.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("employee shift not found: %w", err)
		}
		return nil, fmt.Errorf("get employee shift: %w", err)
	}
	return &es, nil
}

func (r *scheduleRepo) GetEmployeeShiftByID(ctx context.Context, id uuid.UUID) (*models.EmployeeShift, error) {
	var es models.EmployeeShift
	err := r.db.QueryRow(ctx,
		`SELECT `+employeeShiftColumns+` FROM employee_shifts WHERE id = $1`,
		id,
	).Scan(
		&es.ID, &es.ScheduleID, &es.EmployeeID, &es.ShiftID, &es.ShiftDate,
		&es.ShiftStatus, &es.LeaveReason, &es.IsReplacement, &es.ReplacedEmployeeID,
		&es.ReplacementApprovedBy, &es.CheckInTime, &es.CheckOutTime,
		&es.ActualWorkedHours, &es.OvertimeHours, &es.CreatedAt, &es.UpdatedAt, &es.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("employee shift not found by id: %w", err)
		}
		return nil, fmt.Errorf("get employee shift by id: %w", err)
	}
	return &es, nil
}

func (r *scheduleRepo) CreateEmployeeShift(ctx context.Context, es *models.EmployeeShift) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO employee_shifts (schedule_id, employee_id, shift_id, shift_date, shift_status,
			leave_reason, is_replacement, replaced_employee_id, replacement_approved_by, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, created_at, updated_at`,
		es.ScheduleID, es.EmployeeID, es.ShiftID, es.ShiftDate, es.ShiftStatus,
		es.LeaveReason, es.IsReplacement, es.ReplacedEmployeeID, es.ReplacementApprovedBy, es.CreatedBy,
	).Scan(&es.ID, &es.CreatedAt, &es.UpdatedAt)
}

func (r *scheduleRepo) UpdateEmployeeShift(ctx context.Context, es *models.EmployeeShift) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employee_shifts SET shift_id=$1, shift_status=$2, leave_reason=$3,
			is_replacement=$4, replaced_employee_id=$5, replacement_approved_by=$6,
			updated_at=CURRENT_TIMESTAMP WHERE id=$7`,
		es.ShiftID, es.ShiftStatus, es.LeaveReason,
		es.IsReplacement, es.ReplacedEmployeeID, es.ReplacementApprovedBy, es.ID)
	return err
}

func (r *scheduleRepo) UpdateShiftStatus(ctx context.Context, id uuid.UUID, status string, reason *string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employee_shifts SET shift_status=$1, leave_reason=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3`,
		status, reason, id)
	return err
}

func (r *scheduleRepo) AssignReplacement(ctx context.Context, id uuid.UUID, replacementEmployeeID uuid.UUID, approvedBy uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employee_shifts SET is_replacement=true, replaced_employee_id=$1,
			replacement_approved_by=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3`,
		replacementEmployeeID, approvedBy, id)
	return err
}

func (r *scheduleRepo) CheckIn(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employee_shifts SET check_in_time=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id)
	return err
}

func (r *scheduleRepo) CheckOut(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employee_shifts SET check_out_time=CURRENT_TIMESTAMP,
			actual_worked_hours=EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - check_in_time))/3600,
			updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id)
	return err
}

func (r *scheduleRepo) UpsertEmployeeShift(ctx context.Context, es *models.EmployeeShift) error {
	// Upsert by UNIQUE(employee_id, shift_date)
	return r.db.QueryRow(ctx, `
		INSERT INTO employee_shifts (
			schedule_id, employee_id, shift_id, shift_date, shift_status, leave_reason, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (employee_id, shift_date)
		DO UPDATE SET
			schedule_id = EXCLUDED.schedule_id,
			shift_id = EXCLUDED.shift_id,
			shift_status = EXCLUDED.shift_status,
			leave_reason = EXCLUDED.leave_reason,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`,
		es.ScheduleID, es.EmployeeID, es.ShiftID, es.ShiftDate, es.ShiftStatus, es.LeaveReason, es.CreatedBy,
	).Scan(&es.ID, &es.CreatedAt, &es.UpdatedAt)
}

func (r *scheduleRepo) DeleteEmployeeShift(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM employee_shifts WHERE id=$1`, id)
	return err
}

// GetAvailableReplacements returns employees who were OFF or on LEAVE the previous day.
// These are the best candidates to cover a shift today since they had rest yesterday.
func (r *scheduleRepo) GetAvailableReplacements(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.Employee, error) {
	previousDay := date.AddDate(0, 0, -1)
	
	query := `SELECT e.id, e.employee_code, e.first_name, e.last_name, e.gender, e.phone, e.email, e.password_hash,
				e.hire_date, e.role, e.department_id, e.position, e.default_shift_id, e.weekly_off_days,
				e.can_cover_night_shift, e.status, e.profile_image, e.remember_token, e.last_login, e.secondary_phone, e.secondary_email,
				e.created_at, e.updated_at, e.created_by
		 FROM employees e
		 JOIN employee_shifts es ON e.id = es.employee_id
		 WHERE es.shift_date = $1
		   AND es.shift_status IN ('off', 'leave', 'vacation')
		   AND e.status = 'active'
		   AND e.id NOT IN (
		       SELECT employee_id FROM employee_shifts
		       WHERE shift_date = $2 AND shift_status = 'working'
		   )`
	
	args := []interface{}{previousDay, date}
	if departmentID != nil {
		query += ` AND e.department_id = $3`
		args = append(args, *departmentID)
	}
	query += ` ORDER BY e.first_name, e.last_name`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get available replacements: %w", err)
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var emp models.Employee
		if err := rows.Scan(
			&emp.ID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName, &emp.Gender,
			&emp.Phone, &emp.Email, &emp.PasswordHash,
			&emp.HireDate, &emp.Role, &emp.DepartmentID, &emp.Position,
			&emp.DefaultShiftID, &emp.WeeklyOffDays, &emp.CanCoverNightShift,
			&emp.Status, &emp.ProfileImage, &emp.RememberToken, &emp.LastLogin, &emp.SecondaryPhone, &emp.SecondaryEmail,
			&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan replacement employee: %w", err)
		}
		employees = append(employees, emp)
	}
	return employees, rows.Err()
}

// GetEligibleAssignees returns active employees whose default shift matches, excluding those on leave on the given date.
func (r *scheduleRepo) GetEligibleAssignees(ctx context.Context, shiftID uuid.UUID, date time.Time) ([]models.Employee, error) {
	rows, err := r.db.Query(ctx,
		`SELECT e.id, e.employee_code, e.first_name, e.last_name, e.gender, e.phone, e.email, e.password_hash,
				e.hire_date, e.role, e.department_id, e.position, e.default_shift_id, e.weekly_off_days,
				e.can_cover_night_shift, e.status, e.profile_image, e.remember_token, e.last_login, e.secondary_phone, e.secondary_email,
				e.created_at, e.updated_at, e.created_by
		 FROM employees e
		 WHERE e.default_shift_id = $1
		   AND e.status = 'active'
		   AND e.role = 'employee'
		   AND NOT EXISTS (
		       SELECT 1 FROM leave_requests lr
		       WHERE lr.employee_id = e.id
		         AND lr.status = 'approved'
		         AND $2::date BETWEEN lr.start_date AND lr.end_date
		   )
		   AND NOT EXISTS (
		       SELECT 1 FROM employee_shifts es
		       WHERE es.employee_id = e.id
		         AND es.shift_date = $2::date
		         AND es.shift_status IN ('off', 'leave', 'vacation')
		   )
		 ORDER BY e.first_name, e.last_name`, shiftID, date)
	if err != nil {
		return nil, fmt.Errorf("get eligible assignees: %w", err)
	}
	defer rows.Close()

	var employees []models.Employee
	for rows.Next() {
		var emp models.Employee
		if err := rows.Scan(
			&emp.ID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName, &emp.Gender,
			&emp.Phone, &emp.Email, &emp.PasswordHash,
			&emp.HireDate, &emp.Role, &emp.DepartmentID, &emp.Position,
			&emp.DefaultShiftID, &emp.WeeklyOffDays, &emp.CanCoverNightShift,
			&emp.Status, &emp.ProfileImage, &emp.RememberToken, &emp.LastLogin, &emp.SecondaryPhone, &emp.SecondaryEmail,
			&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan eligible assignee: %w", err)
		}
		employees = append(employees, emp)
	}
	return employees, rows.Err()
}

// GetSwapEligibleEmployees returns all employees in the department and indicates if they are off on the requested date.
func (r *scheduleRepo) GetSwapEligibleEmployees(ctx context.Context, departmentID uuid.UUID, excludeEmployeeID uuid.UUID, date time.Time) ([]models.SwapEligibleEmployee, error) {
	rows, err := r.db.Query(ctx,
		`SELECT e.id, e.employee_code, e.first_name, e.last_name, e.gender, e.phone, e.email, e.password_hash,
				e.hire_date, e.role, e.department_id, e.position, e.default_shift_id, e.weekly_off_days,
				e.can_cover_night_shift, e.status, e.profile_image, e.remember_token, e.last_login, e.secondary_phone, e.secondary_email,
				e.created_at, e.updated_at, e.created_by,
				CASE WHEN es.shift_status IN ('off', 'leave', 'vacation') THEN true ELSE false END as is_off
		 FROM employees e
		 LEFT JOIN employee_shifts es ON e.id = es.employee_id AND es.shift_date = $1
		 WHERE e.department_id = $2
		   AND e.id != $3
		   AND e.status = 'active'
		 ORDER BY e.first_name, e.last_name`, date, departmentID, excludeEmployeeID)
	if err != nil {
		return nil, fmt.Errorf("get swap eligible employees: %w", err)
	}
	defer rows.Close()

	var employees []models.SwapEligibleEmployee
	for rows.Next() {
		var emp models.SwapEligibleEmployee
		if err := rows.Scan(
			&emp.ID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName, &emp.Gender,
			&emp.Phone, &emp.Email, &emp.PasswordHash,
			&emp.HireDate, &emp.Role, &emp.DepartmentID, &emp.Position,
			&emp.DefaultShiftID, &emp.WeeklyOffDays, &emp.CanCoverNightShift,
			&emp.Status, &emp.ProfileImage, &emp.RememberToken, &emp.LastLogin, &emp.SecondaryPhone, &emp.SecondaryEmail,
			&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy, &emp.IsOff,
		); err != nil {
			return nil, fmt.Errorf("scan swap eligible employee: %w", err)
		}
		employees = append(employees, emp)
	}
	return employees, rows.Err()
}

// GetShiftCoveragePreview calculates total assignments grouped by status to inform approval decisions
func (r *scheduleRepo) GetShiftCoveragePreview(ctx context.Context, shiftID uuid.UUID, date time.Time) (*models.ShiftCoverage, error) {
	var coverage models.ShiftCoverage
	coverage.ShiftID = shiftID
	coverage.ShiftDate = date

	err := r.db.QueryRow(ctx,
		`SELECT 
			COUNT(id) as total_assigned,
			SUM(CASE WHEN shift_status = 'working' THEN 1 ELSE 0 END) as total_working,
			SUM(CASE WHEN shift_status = 'off' THEN 1 ELSE 0 END) as total_off,
			SUM(CASE WHEN shift_status IN ('leave', 'vacation') THEN 1 ELSE 0 END) as total_on_leave
		 FROM employee_shifts 
		 WHERE shift_date = $1 AND shift_id = $2`, date, shiftID,
	).Scan(&coverage.TotalAssigned, &coverage.TotalWorking, &coverage.TotalOff, &coverage.TotalOnLeave)
	
	if err != nil {
		return nil, fmt.Errorf("get shift coverage preview: %w", err)
	}
	return &coverage, nil
}
