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
	// Schedule Templates — the employee's fixed weekly pattern.
	GetTemplatesByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ScheduleTemplate, error)
	// GetTemplatesForEmployees loads the pattern for many employees in one query,
	// keyed by employee then day-of-week. Used by the per-week materialisation.
	GetTemplatesForEmployees(ctx context.Context, employeeIDs []uuid.UUID) (map[uuid.UUID]map[int]models.ScheduleTemplate, error)
	CreateTemplate(ctx context.Context, t *models.ScheduleTemplate) error
	UpdateTemplate(ctx context.Context, t *models.ScheduleTemplate) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	// UpsertTemplateForDay ensures a permanent off/working pattern entry exists for
	// the given employee + day-of-week, so the setting repeats every week.
	UpsertTemplateForDay(ctx context.Context, employeeID uuid.UUID, dayOfWeek int, isOff bool, shiftID *uuid.UUID) error
	// ReplaceEmployeePattern atomically replaces the employee's whole weekly pattern.
	ReplaceEmployeePattern(ctx context.Context, employeeID uuid.UUID, days []models.PatternDay) error

	// Weekly Schedules
	GetWeeklySchedule(ctx context.Context, weekStart time.Time) (*models.WeeklySchedule, error)
	GetWeeklyScheduleByID(ctx context.Context, id uuid.UUID) (*models.WeeklySchedule, error)
	CreateWeeklySchedule(ctx context.Context, ws *models.WeeklySchedule) error
	UpdateWeeklyScheduleStatus(ctx context.Context, id uuid.UUID, status string, publishedBy *uuid.UUID) error

	// Employee Shifts (daily assignments)
	GetEmployeeShiftsByDate(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.EmployeeShift, error)
	GetEmployeeShiftsByEmployee(ctx context.Context, employeeID uuid.UUID, from, to time.Time) ([]models.EmployeeShift, error)
	GetEmployeeShiftsInRange(ctx context.Context, from, to time.Time) ([]models.EmployeeShift, error)
	// GetEmployeeShiftsInRangeForDept is GetEmployeeShiftsInRange scoped to one department
	// (nil = all). Used by the per-week materialisation to avoid a query per employee-day.
	GetEmployeeShiftsInRangeForDept(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) ([]models.EmployeeShift, error)
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
	// UpsertGeneratedShift writes a pattern-derived day, leaving manual and leave days alone.
	UpsertGeneratedShift(ctx context.Context, es *models.EmployeeShift) error
	// ResyncGeneratedForWeekday pushes a pattern change onto the already-materialised
	// pattern-derived days for that weekday after `after`.
	ResyncGeneratedForWeekday(ctx context.Context, employeeID uuid.UUID, dayOfWeek int, after time.Time, status string, shiftID *uuid.UUID) error
	DeleteEmployeeShift(ctx context.Context, id uuid.UUID) error

	// Smart Replacement: employees who were off/on-leave the previous day
	GetAvailableReplacements(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.Employee, error)
	GetEligibleAssignees(ctx context.Context, shiftID uuid.UUID, date time.Time) ([]models.Employee, error)
	GetSwapEligibleEmployees(ctx context.Context, departmentID uuid.UUID, excludeEmployeeID uuid.UUID, date time.Time) ([]models.SwapEligibleEmployee, error)
	// GetDeptEmployeesNotInSameShift returns active dept employees whose shift on `date` differs from
	// the requester's shift. Used for "swap by shift" mode. employeeShiftID may be uuid.Nil if the
	// requester has no explicit daily row (fallback: matches by default_shift_id).
	GetDeptEmployeesNotInSameShift(ctx context.Context, departmentID uuid.UUID, excludeEmployeeID uuid.UUID, requesterShiftID *uuid.UUID, date time.Time) ([]models.SwapEligibleEmployee, error)
	GetShiftCoveragePreview(ctx context.Context, shiftID uuid.UUID, date time.Time) (*models.ShiftCoverage, error)
}

type scheduleRepo struct {
	db *database.DB
}

func NewScheduleRepository(db *database.DB) ScheduleRepository {
	return &scheduleRepo{db: db}
}

// --- Schedule Templates ---

// templateWinnerOrder decides which row wins when a weekday has more than one
// pattern entry — which the historical UNIQUE(employee, day, valid_from) allowed.
// Still-valid entries beat expired ones, then the newest valid_from, then the newest
// edit. Resolving it here rather than deleting rows keeps every historical entry on
// disk while making the answer deterministic.
const templateWinnerOrder = `(valid_to IS NULL) DESC, COALESCE(valid_from, DATE '0001-01-01') DESC, updated_at DESC, id DESC`

func (r *scheduleRepo) GetTemplatesByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ScheduleTemplate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (day_of_week)
		        id, employee_id, day_of_week, shift_id, is_off, valid_from, valid_to, created_at, updated_at
		 FROM schedule_templates
		 WHERE employee_id = $1 AND day_of_week IS NOT NULL
		 ORDER BY day_of_week, `+templateWinnerOrder, employeeID)
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

func (r *scheduleRepo) GetTemplatesForEmployees(ctx context.Context, employeeIDs []uuid.UUID) (map[uuid.UUID]map[int]models.ScheduleTemplate, error) {
	out := map[uuid.UUID]map[int]models.ScheduleTemplate{}
	if len(employeeIDs) == 0 {
		return out, nil
	}

	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (employee_id, day_of_week)
		        id, employee_id, day_of_week, shift_id, is_off, valid_from, valid_to, created_at, updated_at
		 FROM schedule_templates
		 WHERE employee_id = ANY($1) AND day_of_week IS NOT NULL
		 ORDER BY employee_id, day_of_week, `+templateWinnerOrder, employeeIDs)
	if err != nil {
		return nil, fmt.Errorf("get templates for employees: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t models.ScheduleTemplate
		if err := rows.Scan(&t.ID, &t.EmployeeID, &t.DayOfWeek, &t.ShiftID, &t.IsOff,
			&t.ValidFrom, &t.ValidTo, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		if out[t.EmployeeID] == nil {
			out[t.EmployeeID] = make(map[int]models.ScheduleTemplate, 7)
		}
		out[t.EmployeeID][t.DayOfWeek] = t
	}
	return out, rows.Err()
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

// Updates the still-valid entries in place rather than deleting and re-inserting, so
// no historical row is ever removed. A weekday carrying several still-valid entries
// gets them all updated to the same value, which converges on the right answer
// whichever one templateWinnerOrder later picks.
//
// valid_from is deliberately left untouched. It is part of the table's
// UNIQUE(employee_id, day_of_week, valid_from), so rewriting it to CURRENT_DATE across
// several duplicate entries collapses them onto one key and the update fails with
//
//	duplicate key value violates unique constraint
//	"schedule_templates_employee_id_day_of_week_valid_from_key"
//
// Older databases really do carry those duplicates. Since no column of the unique key
// is modified here, this update cannot collide regardless of how many entries exist.
func (r *scheduleRepo) UpsertTemplateForDay(
	ctx context.Context,
	employeeID uuid.UUID,
	dayOfWeek int,
	isOff bool,
	shiftID *uuid.UUID,
) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE schedule_templates
		    SET shift_id   = $1,
		        is_off     = $2,
		        updated_at = CURRENT_TIMESTAMP
		  WHERE employee_id = $3 AND day_of_week = $4 AND valid_to IS NULL`,
		shiftID, isOff, employeeID, dayOfWeek)
	if err != nil {
		return fmt.Errorf("upsert template for day (update): %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	// No valid entry for this weekday yet. The ON CONFLICT targets the table's
	// existing UNIQUE(employee_id, day_of_week, valid_from), which an expired entry
	// created earlier today could otherwise collide with.
	_, err = r.db.Exec(ctx,
		`INSERT INTO schedule_templates (employee_id, day_of_week, shift_id, is_off, valid_from, valid_to)
		 VALUES ($1, $2, $3, $4, CURRENT_DATE, NULL)
		 ON CONFLICT (employee_id, day_of_week, valid_from) DO UPDATE SET
		     shift_id   = EXCLUDED.shift_id,
		     is_off     = EXCLUDED.is_off,
		     valid_to   = NULL,
		     updated_at = CURRENT_TIMESTAMP`,
		employeeID, dayOfWeek, shiftID, isOff)
	if err != nil {
		return fmt.Errorf("upsert template for day (insert): %w", err)
	}
	return nil
}

// ReplaceEmployeePattern rewrites the employee's whole weekly pattern in one transaction.
func (r *scheduleRepo) ReplaceEmployeePattern(ctx context.Context, employeeID uuid.UUID, days []models.PatternDay) error {
	return r.db.ExecTx(ctx, func(txCtx context.Context, _ pgx.Tx) error {
		for _, d := range days {
			if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
				return fmt.Errorf("invalid day_of_week: %d", d.DayOfWeek)
			}
			shiftID := d.ShiftID
			if d.IsOff {
				shiftID = nil
			}
			if err := r.UpsertTemplateForDay(txCtx, employeeID, d.DayOfWeek, d.IsOff, shiftID); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- Weekly Schedules ---

func (r *scheduleRepo) GetWeeklySchedule(ctx context.Context, weekStart time.Time) (*models.WeeklySchedule, error) {
	var ws models.WeeklySchedule
	// Oldest-first, not newest-first: migration 018 dropped UNIQUE(week_start_date) in
	// favour of UNIQUE(department_id, week_start_date) while the application still
	// writes a NULL department_id, and NULLs are distinct in a unique constraint — so a
	// week can have several header rows. Always resolving to the first one ever created
	// keeps the choice stable, which is all the header is used for, without having to
	// delete the extra rows.
	err := r.db.QueryRow(ctx,
		`SELECT id, week_start_date, week_end_date, template_id, status, published_by, published_at, notes, created_at
		 FROM weekly_schedule WHERE week_start_date = $1::date
		 ORDER BY created_at ASC, id ASC LIMIT 1`, weekStart,
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
			&es.Source,
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
	created_at, updated_at, created_by, source`

// employeeShiftColumnsAliased is employeeShiftColumns qualified with the `es` table alias.
const employeeShiftColumnsAliased = `es.id, es.schedule_id, es.employee_id, es.shift_id, es.shift_date, es.shift_status,
	es.leave_reason, es.is_replacement, es.replaced_employee_id, es.replacement_approved_by,
	es.check_in_time, es.check_out_time, es.actual_worked_hours, es.overtime_hours,
	es.created_at, es.updated_at, es.created_by, es.source`

func (r *scheduleRepo) GetEmployeeShiftsByDate(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.EmployeeShift, error) {
	query := `SELECT ` + employeeShiftColumnsAliased + `
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

func (r *scheduleRepo) GetEmployeeShiftsInRangeForDept(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) ([]models.EmployeeShift, error) {
	if departmentID == nil {
		return r.GetEmployeeShiftsInRange(ctx, from, to)
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+employeeShiftColumnsAliased+` FROM employee_shifts es
		 JOIN employees e ON e.id = es.employee_id
		 WHERE es.shift_date BETWEEN $1 AND $2 AND e.department_id = $3
		 ORDER BY es.employee_id, es.shift_date`, from, to, *departmentID)
	if err != nil {
		return nil, fmt.Errorf("get shifts in range for dept: %w", err)
	}
	defer rows.Close()
	return r.scanEmployeeShifts(rows)
}

func (r *scheduleRepo) GetDepartmentShiftsInRange(ctx context.Context, from, to time.Time, departmentID uuid.UUID) ([]models.EmployeeShiftExtended, error) {
	query := `SELECT ` + employeeShiftColumnsAliased + `,
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
			&es.Source,
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
		&es.Source,
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
		&es.Source,
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
			leave_reason, is_replacement, replaced_employee_id, replacement_approved_by, created_by, source)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id, created_at, updated_at`,
		es.ScheduleID, es.EmployeeID, es.ShiftID, es.ShiftDate, es.ShiftStatus,
		es.LeaveReason, es.IsReplacement, es.ReplacedEmployeeID, es.ReplacementApprovedBy, es.CreatedBy,
		normalizeSource(es.Source),
	).Scan(&es.ID, &es.CreatedAt, &es.UpdatedAt)
}

// UpdateEmployeeShift edits an existing row. Any explicit edit (swap, replacement)
// pins the row as manual so the pattern materialiser will not re-derive it.
func (r *scheduleRepo) UpdateEmployeeShift(ctx context.Context, es *models.EmployeeShift) error {
	_, err := r.db.Exec(ctx,
		`UPDATE employee_shifts SET shift_id=$1, shift_status=$2, leave_reason=$3,
			is_replacement=$4, replaced_employee_id=$5, replacement_approved_by=$6,
			source=$7, updated_at=CURRENT_TIMESTAMP WHERE id=$8`,
		es.ShiftID, es.ShiftStatus, es.LeaveReason,
		es.IsReplacement, es.ReplacedEmployeeID, es.ReplacementApprovedBy,
		normalizeSource(es.Source), es.ID)
	return err
}

// normalizeSource defaults an unset provenance to "manual": callers that do not
// state a source are making an explicit change, which must never be re-derived.
func normalizeSource(s string) string {
	switch s {
	case models.ShiftSourceGenerated, models.ShiftSourceManual, models.ShiftSourceLeave:
		return s
	default:
		return models.ShiftSourceManual
	}
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
			schedule_id, employee_id, shift_id, shift_date, shift_status, leave_reason, created_by, source
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (employee_id, shift_date)
		DO UPDATE SET
			schedule_id = EXCLUDED.schedule_id,
			shift_id = EXCLUDED.shift_id,
			shift_status = EXCLUDED.shift_status,
			leave_reason = EXCLUDED.leave_reason,
			source = EXCLUDED.source,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`,
		es.ScheduleID, es.EmployeeID, es.ShiftID, es.ShiftDate, es.ShiftStatus, es.LeaveReason, es.CreatedBy,
		normalizeSource(es.Source),
	).Scan(&es.ID, &es.CreatedAt, &es.UpdatedAt)
}

// The `WHERE employee_shifts.source = 'generated'` on the conflict branch is the
// safety rail of the whole design: a day a human touched, or that an approved leave
// owns, is left exactly as it is even under a concurrent write. Everything else stays
// free to be re-derived, which is what stops a week from freezing into whatever was
// seeded into it first.
func (r *scheduleRepo) UpsertGeneratedShift(ctx context.Context, es *models.EmployeeShift) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO employee_shifts (
			schedule_id, employee_id, shift_id, shift_date, shift_status, leave_reason, created_by, source
		) VALUES ($1,$2,$3,$4,$5,NULL,$6,'generated')
		ON CONFLICT (employee_id, shift_date)
		DO UPDATE SET
			schedule_id  = EXCLUDED.schedule_id,
			shift_id     = EXCLUDED.shift_id,
			shift_status = EXCLUDED.shift_status,
			leave_reason = NULL,
			updated_at   = CURRENT_TIMESTAMP
		WHERE employee_shifts.source = 'generated'
	`,
		es.ScheduleID, es.EmployeeID, es.ShiftID, es.ShiftDate, es.ShiftStatus, es.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("upsert generated shift: %w", err)
	}
	return nil
}

// Without this, a permanent edit would only reach weeks nobody had opened yet: weeks
// already materialised from the previous pattern would keep the stale value.
func (r *scheduleRepo) ResyncGeneratedForWeekday(
	ctx context.Context,
	employeeID uuid.UUID,
	dayOfWeek int,
	after time.Time,
	status string,
	shiftID *uuid.UUID,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE employee_shifts
		   SET shift_id     = $1,
		       shift_status = $2,
		       leave_reason = NULL,
		       updated_at   = CURRENT_TIMESTAMP
		 WHERE employee_id = $3
		   AND source = 'generated'
		   AND shift_date > $4
		   AND EXTRACT(DOW FROM shift_date)::int = $5
	`, shiftID, status, employeeID, after, dayOfWeek)
	if err != nil {
		return fmt.Errorf("resync generated weekday: %w", err)
	}
	return nil
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
		       SELECT 1 FROM leaves lr
		       WHERE lr.employee_id = e.id
		         -- Matches GetApprovedForSchedule: manager approval is the terminal
		         -- approved state. 'approved' is not a leave_status label at all.
		         AND lr.status = 'approved_by_manager'
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

// GetSwapEligibleEmployees returns employees in the department who have shift_status='off'
// (NOT leave/vacation) on the requested date. These are the only valid swap targets.
func (r *scheduleRepo) GetSwapEligibleEmployees(ctx context.Context, departmentID uuid.UUID, excludeEmployeeID uuid.UUID, date time.Time) ([]models.SwapEligibleEmployee, error) {
	rows, err := r.db.Query(ctx,
		`SELECT e.id, e.employee_code, e.first_name, e.last_name, e.gender, e.phone, e.email, e.password_hash,
				e.hire_date, e.role, e.department_id, e.position, e.default_shift_id, e.weekly_off_days,
				e.can_cover_night_shift, e.status, e.profile_image, e.remember_token, e.last_login, e.secondary_phone, e.secondary_email,
				e.created_at, e.updated_at, e.created_by
		 FROM employees e
		 INNER JOIN employee_shifts es ON e.id = es.employee_id
		                               AND es.shift_date = $1
		                               AND es.shift_status = 'off'
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
			&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan swap eligible employee: %w", err)
		}
		emp.IsOff = true // always true since we filter by shift_status='off'
		employees = append(employees, emp)
	}
	return employees, rows.Err()
}

// GetDeptEmployeesNotInSameShift returns all active employees in `departmentID` (excluding the requester)
// whose effective shift on `date` is different from `requesterShiftID`.
// "Effective shift" = employee_shifts.shift_id for that date if a row exists,
// otherwise the employee's default_shift_id.
// If `requesterShiftID` is nil (requester has no shift), we return all dept employees.
func (r *scheduleRepo) GetDeptEmployeesNotInSameShift(
	ctx context.Context,
	departmentID uuid.UUID,
	excludeEmployeeID uuid.UUID,
	requesterShiftID *uuid.UUID,
	date time.Time,
) ([]models.SwapEligibleEmployee, error) {
	// COALESCE(es.shift_id, e.default_shift_id) gives the "effective shift"
	// for the employee on the requested date.
	query := `
		SELECT e.id, e.employee_code, e.first_name, e.last_name, e.gender, e.phone, e.email, e.password_hash,
			   e.hire_date, e.role, e.department_id, e.position, e.default_shift_id, e.weekly_off_days,
			   e.can_cover_night_shift, e.status, e.profile_image, e.remember_token, e.last_login, e.secondary_phone, e.secondary_email,
			   e.created_at, e.updated_at, e.created_by,
			   COALESCE(es.shift_status, 'working') AS effective_status
		FROM employees e
		LEFT JOIN employee_shifts es ON es.employee_id = e.id AND es.shift_date = $1::date
		WHERE e.department_id = $2
		  AND e.id != $3
		  AND e.status = 'active'`

	args := []interface{}{date, departmentID, excludeEmployeeID}

	if requesterShiftID != nil {
		// Exclude employees that share the same effective shift_id as the requester
		query += `
		  AND COALESCE(es.shift_id, e.default_shift_id) IS DISTINCT FROM $4`
		args = append(args, *requesterShiftID)
	}

	query += `
		ORDER BY e.first_name, e.last_name`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get dept employees not in same shift: %w", err)
	}
	defer rows.Close()

	var employees []models.SwapEligibleEmployee
	for rows.Next() {
		var emp models.SwapEligibleEmployee
		var effectiveStatus string
		if err := rows.Scan(
			&emp.ID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName, &emp.Gender,
			&emp.Phone, &emp.Email, &emp.PasswordHash,
			&emp.HireDate, &emp.Role, &emp.DepartmentID, &emp.Position,
			&emp.DefaultShiftID, &emp.WeeklyOffDays, &emp.CanCoverNightShift,
			&emp.Status, &emp.ProfileImage, &emp.RememberToken, &emp.LastLogin, &emp.SecondaryPhone, &emp.SecondaryEmail,
			&emp.CreatedAt, &emp.UpdatedAt, &emp.CreatedBy,
			&effectiveStatus,
		); err != nil {
			return nil, fmt.Errorf("scan dept employee not in same shift: %w", err)
		}
		emp.IsOff = effectiveStatus == "off"
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
