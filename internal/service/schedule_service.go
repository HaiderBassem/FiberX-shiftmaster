package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/pkg/database"
)

// ScheduleService handles schedule generation, publishing, and smart replacement.
type ScheduleService struct {
	scheduleRepo repository.ScheduleRepository
	employeeRepo repository.EmployeeRepository
	shiftRepo    repository.ShiftRepository
	leaveRepo    repository.LeaveRepository
	notifService *NotificationService
	emailService *EmailService
	db           *database.DB
}

func NewScheduleService(
	scheduleRepo repository.ScheduleRepository,
	employeeRepo repository.EmployeeRepository,
	shiftRepo repository.ShiftRepository,
	leaveRepo repository.LeaveRepository,
	notifService *NotificationService,
	emailService *EmailService,
	db *database.DB,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		employeeRepo: employeeRepo,
		shiftRepo:    shiftRepo,
		leaveRepo:    leaveRepo,
		notifService: notifService,
		emailService: emailService,
		db:           db,
	}
}

// maxEnsureWeeks bounds how many weeks a single range request will materialise,
// so a wide from/to cannot turn one HTTP call into unbounded work.
const maxEnsureWeeks = 14

func normalizeWeekStart(d time.Time) time.Time {
	d = d.UTC().Truncate(24 * time.Hour)
	for d.Weekday() != time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

func today() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour)
}

// baselineForDay resolves what an employee's day looks like according to their fixed
// weekly pattern. The pattern (schedule_templates) is authoritative; the employee's
// default_shift_id / weekly_off_days are only a fallback for employees who have no
// pattern row yet. Deliberately does NOT look at last week's actual rows: those carry
// one-off noise (swaps, replacements, cover) that must not become the recurring pattern.
func baselineForDay(emp models.Employee, day int, tmpl *models.ScheduleTemplate) (string, *uuid.UUID) {
	if tmpl != nil {
		if tmpl.IsOff {
			return "off", nil
		}
		shiftID := tmpl.ShiftID
		if shiftID == nil {
			shiftID = emp.DefaultShiftID
		}
		if shiftID == nil {
			return "off", nil
		}
		return "working", shiftID
	}

	if emp.WeeklyOffDays >= 0 && emp.WeeklyOffDays == day {
		return "off", nil
	}
	if emp.DefaultShiftID != nil {
		return "working", emp.DefaultShiftID
	}
	return "off", nil
}

func sameAssignment(es models.EmployeeShift, status string, shiftID *uuid.UUID) bool {
	if !strings.EqualFold(es.ShiftStatus, status) {
		return false
	}
	if (es.ShiftID == nil) != (shiftID == nil) {
		return false
	}
	return es.ShiftID == nil || *es.ShiftID == *shiftID
}

func dayKey(d time.Time) string { return d.UTC().Format("2006-01-02") }

// EnsureWeekSchedule materialises one week from each employee's fixed weekly pattern,
// then overlays fully approved leaves.
//
// Rows are only ever created or corrected when they are pattern-derived
// (source='generated') and dated today or later. A manual edit, an approved leave, a
// swap, and every past day are left untouched. That makes the call idempotent and
// self-healing: a week that was materialised early — e.g. by opening a month view —
// is re-derived on the next read instead of freezing whatever was seeded at the time.
//
// departmentID scopes the work to one department (nil = every active employee).
func (s *ScheduleService) EnsureWeekSchedule(ctx context.Context, refDate time.Time, departmentID *uuid.UUID) error {
	weekStart := normalizeWeekStart(refDate)
	weekEnd := weekStart.AddDate(0, 0, 6)

	ws, err := s.getOrCreateWeeklySchedule(ctx, weekStart, weekEnd)
	if err != nil {
		return err
	}

	var employees []models.Employee
	if departmentID != nil {
		employees, err = s.employeeRepo.GetByDepartment(ctx, *departmentID)
		if err != nil {
			return fmt.Errorf("get department employees: %w", err)
		}
	} else {
		employees, err = s.employeeRepo.GetActive(ctx)
		if err != nil {
			return fmt.Errorf("get active employees: %w", err)
		}
	}

	active := make([]models.Employee, 0, len(employees))
	ids := make([]uuid.UUID, 0, len(employees))
	for _, emp := range employees {
		if !strings.EqualFold(emp.Status, "active") {
			continue
		}
		active = append(active, emp)
		ids = append(ids, emp.ID)
	}
	if len(active) == 0 {
		return nil
	}

	// Two batch reads replace the previous query-per-employee-per-day.
	patterns, err := s.scheduleRepo.GetTemplatesForEmployees(ctx, ids)
	if err != nil {
		return fmt.Errorf("get weekly patterns: %w", err)
	}
	existingRows, err := s.scheduleRepo.GetEmployeeShiftsInRangeForDept(ctx, weekStart, weekEnd, departmentID)
	if err != nil {
		return fmt.Errorf("get existing week rows: %w", err)
	}
	existing := map[uuid.UUID]map[string]models.EmployeeShift{}
	for _, row := range existingRows {
		if existing[row.EmployeeID] == nil {
			existing[row.EmployeeID] = map[string]models.EmployeeShift{}
		}
		existing[row.EmployeeID][dayKey(row.ShiftDate)] = row
	}

	cutoff := today()

	for _, emp := range active {
		for day := 0; day < 7; day++ {
			shiftDate := weekStart.AddDate(0, 0, day)

			var tmpl *models.ScheduleTemplate
			if t, ok := patterns[emp.ID][day]; ok {
				tmpl = &t
			}
			status, shiftID := baselineForDay(emp, day, tmpl)

			if cur, ok := existing[emp.ID][dayKey(shiftDate)]; ok {
				// Never touch a human decision, an approved leave, or a past day.
				if cur.Source != models.ShiftSourceGenerated || shiftDate.Before(cutoff) {
					continue
				}
				if sameAssignment(cur, status, shiftID) {
					continue
				}
			}

			es := &models.EmployeeShift{
				ScheduleID:  ws.ID,
				EmployeeID:  emp.ID,
				ShiftID:     shiftID,
				ShiftDate:   shiftDate,
				ShiftStatus: status,
				Source:      models.ShiftSourceGenerated,
			}
			if upsertErr := s.scheduleRepo.UpsertGeneratedShift(ctx, es); upsertErr != nil {
				return fmt.Errorf("materialise shift for %s on %s: %w", emp.EmployeeCode, shiftDate.Format("2006-01-02"), upsertErr)
			}
		}
	}

	return s.applyApprovedLeavesForRange(ctx, ws, weekStart, weekEnd)
}

// ensureRange materialises every week the requested range touches. The previous code
// only ensured the first and last week, which left the middle weeks of a month view
// with no rows at all.
func (s *ScheduleService) ensureRange(ctx context.Context, from, to time.Time, departmentID *uuid.UUID) error {
	first := normalizeWeekStart(from)
	last := normalizeWeekStart(to)
	if last.Before(first) {
		first, last = last, first
	}

	weeks := 0
	for w := first; !w.After(last) && weeks < maxEnsureWeeks; w = w.AddDate(0, 0, 7) {
		if err := s.EnsureWeekSchedule(ctx, w, departmentID); err != nil {
			return fmt.Errorf("ensure week %s: %w", w.Format("2006-01-02"), err)
		}
		weeks++
	}
	return nil
}

func (s *ScheduleService) getOrCreateWeeklySchedule(ctx context.Context, weekStart, weekEnd time.Time) (*models.WeeklySchedule, error) {
	ws, err := s.scheduleRepo.GetWeeklySchedule(ctx, weekStart)
	if err == nil {
		return ws, nil
	}
	ws = &models.WeeklySchedule{
		WeekStartDate: weekStart,
		WeekEndDate:   weekEnd,
		Status:        "draft",
	}
	if createErr := s.scheduleRepo.CreateWeeklySchedule(ctx, ws); createErr != nil {
		if existing, getErr := s.scheduleRepo.GetWeeklySchedule(ctx, weekStart); getErr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("create weekly schedule: %w", createErr)
	}
	return ws, nil
}

func (s *ScheduleService) applyApprovedLeavesForRange(ctx context.Context, ws *models.WeeklySchedule, from, to time.Time) error {
	leaves, err := s.leaveRepo.GetApprovedForSchedule(ctx, from, to)
	if err != nil {
		return fmt.Errorf("get approved leaves: %w", err)
	}

	for _, leave := range leaves {
		leaveReason := "leave"
		if leave.Reason != nil && *leave.Reason != "" {
			leaveReason = *leave.Reason
		}
		leaveReasonPtr := &leaveReason

		emp, _ := s.employeeRepo.GetByID(ctx, leave.EmployeeID)

		start := leave.StartDate.UTC().Truncate(24 * time.Hour)
		end := leave.EndDate.UTC().Truncate(24 * time.Hour)
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}

		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			var shiftID *uuid.UUID
			if existing, existErr := s.scheduleRepo.GetEmployeeShift(ctx, leave.EmployeeID, d); existErr == nil && existing != nil {
				shiftID = existing.ShiftID
			} else if emp != nil {
				shiftID = emp.DefaultShiftID
			}

			shiftStatus := "leave"
			// Driven by leave_types.is_hourly rather than by matching the type's
			// name, which administrators can rename and translate. The previous
			// check also compared an Arabic literal against the English column,
			// so that half of it could never match.
			if leave.LeaveTypeIsHourly {
				leaveReason = models.HourlyLeaveReasonPrefix + leaveReason
			}

			es := &models.EmployeeShift{
				ScheduleID:  ws.ID,
				EmployeeID:  leave.EmployeeID,
				ShiftID:     shiftID,
				ShiftDate:   d,
				ShiftStatus: shiftStatus,
				LeaveReason: leaveReasonPtr,
				Source:      models.ShiftSourceLeave,
			}
			if upsertErr := s.scheduleRepo.UpsertEmployeeShift(ctx, es); upsertErr != nil {
				return fmt.Errorf("apply leave shift for %s on %s: %w", leave.EmployeeID, d.Format("2006-01-02"), upsertErr)
			}
		}
	}
	return nil
}

// GenerateWeeklySchedule creates a weekly schedule from employee templates.
func (s *ScheduleService) GenerateWeeklySchedule(ctx context.Context, weekStart time.Time, createdBy uuid.UUID) (*models.WeeklySchedule, error) {
	// Ensure weekStart is a Sunday (day 0)
	for weekStart.Weekday() != time.Sunday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekEnd := weekStart.AddDate(0, 0, 6)

	// Check if schedule already exists
	existing, _ := s.scheduleRepo.GetWeeklySchedule(ctx, weekStart)
	if existing != nil {
		return nil, fmt.Errorf("weekly schedule already exists for %s", weekStart.Format("2006-01-02"))
	}

	ws := &models.WeeklySchedule{
		WeekStartDate: weekStart,
		WeekEndDate:   weekEnd,
		Status:        "draft",
	}

	// Use transaction for atomic creation
	err := s.db.ExecTx(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		// Create weekly schedule record
		if err := s.scheduleRepo.CreateWeeklySchedule(txCtx, ws); err != nil {
			return fmt.Errorf("create weekly schedule: %w", err)
		}

		// Get all active employees
		employees, err := s.employeeRepo.GetActive(txCtx)
		if err != nil {
			return fmt.Errorf("get active employees: %w", err)
		}

		// For each employee, get their templates and create daily shifts
		for _, emp := range employees {
			templates, err := s.scheduleRepo.GetTemplatesByEmployee(txCtx, emp.ID)
			if err != nil {
				return fmt.Errorf("get templates for %s: %w", emp.EmployeeCode, err)
			}

			for _, tmpl := range templates {
				shiftDate := weekStart.AddDate(0, 0, tmpl.DayOfWeek)
				status := "working"
				if tmpl.IsOff {
					status = "off"
				}

				es := &models.EmployeeShift{
					ScheduleID:  ws.ID,
					EmployeeID:  emp.ID,
					ShiftID:     tmpl.ShiftID,
					ShiftDate:   shiftDate,
					ShiftStatus: status,
					CreatedBy:   &createdBy,
					// Pattern-derived, so a later pattern change still reaches these days.
					Source: models.ShiftSourceGenerated,
				}
				if err := s.scheduleRepo.CreateEmployeeShift(txCtx, es); err != nil {
					return fmt.Errorf("create shift for %s on %s: %w",
						emp.EmployeeCode, shiftDate.Format("2006-01-02"), err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return ws, nil
}

// PublishSchedule publishes a draft schedule so employees can see it.
func (s *ScheduleService) PublishSchedule(ctx context.Context, scheduleID uuid.UUID, publishedBy uuid.UUID) error {
	ws, err := s.scheduleRepo.GetWeeklyScheduleByID(ctx, scheduleID)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}
	if ws.Status != "draft" {
		return fmt.Errorf("can only publish draft schedules, current status: %s", ws.Status)
	}

	if err := s.scheduleRepo.UpdateWeeklyScheduleStatus(ctx, scheduleID, "published", &publishedBy); err != nil {
		return err
	}

	// Notify all active employees
	employees, _ := s.employeeRepo.GetActive(ctx)
	for _, emp := range employees {
		_ = s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       emp.ID,
			SenderID:          &publishedBy,
			Type:              "shift_change",
			Title:             "New Schedule Published",
			Message:           strPtr(fmt.Sprintf("Weekly schedule for %s has been published", ws.WeekStartDate.Format("2006-01-02"))),
			RelatedEntityType: strPtr("schedule"),
			RelatedEntityID:   &scheduleID,
			Priority:          "medium",
		})
	}

	return nil
}

// GetAvailableReplacements returns employees who were off/on-leave the previous day.
// These employees are the best candidates to cover a morning shift today.
func (s *ScheduleService) GetAvailableReplacements(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.Employee, error) {
	return s.scheduleRepo.GetAvailableReplacements(ctx, date, departmentID)
}

// AssignReplacement assigns a replacement employee for a shift, marking the original as replaced.
func (s *ScheduleService) AssignReplacement(ctx context.Context, shiftID uuid.UUID, replacementEmployeeID uuid.UUID, approvedBy uuid.UUID) error {
	// Verify replacement employee exists and is active
	replacement, err := s.employeeRepo.GetByID(ctx, replacementEmployeeID)
	if err != nil {
		return fmt.Errorf("replacement employee not found: %w", err)
	}
	if replacement.Status != "active" {
		return fmt.Errorf("replacement employee is not active")
	}

	if err := s.scheduleRepo.AssignReplacement(ctx, shiftID, replacementEmployeeID, approvedBy); err != nil {
		return err
	}

	// Fetch shift details for the notification
	es, err := s.scheduleRepo.GetEmployeeShiftByID(ctx, shiftID)
	if err == nil && es != nil {
		origEmp, _ := s.employeeRepo.GetByID(ctx, es.EmployeeID)
		var shiftName, shiftTimes string
		if es.ShiftID != nil {
			sh, _ := s.shiftRepo.GetByID(ctx, *es.ShiftID)
			if sh != nil {
				shiftName = sh.Name
				shiftTimes = fmt.Sprintf("%s - %s", sh.StartTime.Format("15:04"), sh.EndTime.Format("15:04"))
			}
		}

		shiftDateStr := es.ShiftDate.Format("2006-01-02")
		origName := "an employee"
		if origEmp != nil {
			origName = fmt.Sprintf("%s %s", origEmp.FirstName, origEmp.LastName)
		}

		msg := fmt.Sprintf("You have been assigned as a replacement for %s on %s. Shift: %s (%s)", origName, shiftDateStr, shiftName, shiftTimes)

		// Notify the replacement employee
		_ = s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       replacementEmployeeID,
			SenderID:          &approvedBy,
			Type:              "shift_change",
			Title:             "Replacement Assignment",
			Message:           strPtr(msg),
			RelatedEntityType: strPtr("schedule"),
			RelatedEntityID:   &shiftID,
			Priority:          "high",
		})

		// Send email if possible
		if s.emailService != nil && replacement.Email != "" {
			s.emailService.SendEmailAsync(
				[]string{replacement.Email},
				"Shift Replacement Assignment",
				msg,
			)
		}
	} else {
		// Fallback if unable to fetch details
		_ = s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       replacementEmployeeID,
			SenderID:          &approvedBy,
			Type:              "shift_change",
			Title:             "Replacement Assignment",
			Message:           strPtr("You have been assigned as a replacement for a shift"),
			RelatedEntityType: strPtr("schedule"),
			RelatedEntityID:   &shiftID,
			Priority:          "high",
		})
	}

	return nil
}

// CheckIn records an employee's check-in time.
func (s *ScheduleService) CheckIn(ctx context.Context, shiftID uuid.UUID) error {
	return s.scheduleRepo.CheckIn(ctx, shiftID)
}

// CheckOut records an employee's check-out time and calculates hours.
func (s *ScheduleService) CheckOut(ctx context.Context, shiftID uuid.UUID) error {
	return s.scheduleRepo.CheckOut(ctx, shiftID)
}

// GetDailyShifts returns all shifts for a specific date.
func (s *ScheduleService) GetDailyShifts(ctx context.Context, date time.Time, departmentID *uuid.UUID) ([]models.EmployeeShift, error) {
	if err := s.EnsureWeekSchedule(ctx, date, departmentID); err != nil {
		return nil, fmt.Errorf("ensure week schedule: %w", err)
	}
	return s.scheduleRepo.GetEmployeeShiftsByDate(ctx, date, departmentID)
}

// GetEmployeeShifts returns an employee's shifts for a date range.
func (s *ScheduleService) GetEmployeeShifts(ctx context.Context, employeeID uuid.UUID, from, to time.Time) ([]models.EmployeeShift, error) {
	// Scope the materialisation to this employee's department rather than every
	// employee in the company.
	var departmentID *uuid.UUID
	if emp, err := s.employeeRepo.GetByID(ctx, employeeID); err == nil && emp != nil {
		departmentID = emp.DepartmentID
	}
	if err := s.ensureRange(ctx, from, to, departmentID); err != nil {
		return nil, fmt.Errorf("ensure week schedule: %w", err)
	}
	return s.scheduleRepo.GetEmployeeShiftsByEmployee(ctx, employeeID, from, to)
}

// GetDepartmentShiftsInRange gets shifts for a whole department in a date range.
func (s *ScheduleService) GetDepartmentShiftsInRange(ctx context.Context, from, to time.Time, departmentID uuid.UUID) ([]models.EmployeeShiftExtended, error) {
	if err := s.ensureRange(ctx, from, to, &departmentID); err != nil {
		return nil, fmt.Errorf("ensure week schedule: %w", err)
	}
	return s.scheduleRepo.GetDepartmentShiftsInRange(ctx, from, to, departmentID)
}

// assertCanManage enforces the role hierarchy: TLs cannot touch managers/admins,
// managers cannot touch admins.
func assertCanManage(actorRole string, target *models.Employee) error {
	if actorRole == "team_leader" && (target.Role == "manager" || target.Role == "admin") {
		return fmt.Errorf("team leaders cannot modify schedules for managers or admins")
	}
	if actorRole == "manager" && target.Role == "admin" {
		return fmt.Errorf("managers cannot modify schedules for admins")
	}
	return nil
}

// SetEmployeeShift upserts a single employee shift for a day.
// If the weekly schedule record for that week doesn't exist, it will be created as a draft.
//
// permanent controls whether this becomes part of the employee's fixed weekly pattern:
//   - true  → also written to schedule_templates and pushed onto every future
//     pattern-derived day for that weekday, so it repeats every week.
//   - false → this date only. Useful for one-off cover, so a temporary change does
//     not silently become the recurring schedule.
//
// Either way the day itself is pinned as a manual row and will never be re-derived.
func (s *ScheduleService) SetEmployeeShift(ctx context.Context, employeeID uuid.UUID, shiftDate time.Time, shiftID *uuid.UUID, shiftStatus string, leaveReason *string, createdBy uuid.UUID, creatorRole string, permanent bool) (*models.EmployeeShift, error) {
	valid := map[string]bool{"working": true, "off": true, "leave": true, "vacation": true, "hourly": true}
	if !valid[shiftStatus] {
		return nil, fmt.Errorf("invalid shift_status: %s", shiftStatus)
	}

	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}
	if emp.Status != "active" {
		return nil, fmt.Errorf("employee is not active")
	}

	if err := assertCanManage(creatorRole, emp); err != nil {
		return nil, err
	}

	// For working shifts, shift_id is required.
	if shiftStatus == "working" && shiftID == nil {
		return nil, fmt.Errorf("shift_id is required when shift_status=working")
	}

	// Validate shift exists if provided
	if shiftID != nil {
		if _, err := s.shiftRepo.GetByID(ctx, *shiftID); err != nil {
			return nil, fmt.Errorf("shift not found: %w", err)
		}
	}

	// Normalize week start to Sunday
	weekStart := normalizeWeekStart(shiftDate)
	weekEnd := weekStart.AddDate(0, 0, 6)

	// Ensure weekly schedule exists (draft by default)
	ws, err := s.getOrCreateWeeklySchedule(ctx, weekStart, weekEnd)
	if err != nil {
		return nil, err
	}

	es := &models.EmployeeShift{
		ScheduleID:  ws.ID,
		EmployeeID:  employeeID,
		ShiftID:     shiftID,
		ShiftDate:   shiftDate,
		ShiftStatus: shiftStatus,
		LeaveReason: leaveReason,
		CreatedBy:   &createdBy,
		Source:      models.ShiftSourceManual,
	}

	// The day, the pattern entry and the push onto future weeks are one unit of work.
	// Writing the day first and only then failing on the pattern left the caller with a
	// 400 describing a change that had in fact been saved — the UI treats that as a
	// failure and never refreshes, so the edit looks lost until a manual reload.
	err = s.db.ExecTx(ctx, func(txCtx context.Context, _ pgx.Tx) error {
		if upsertErr := s.scheduleRepo.UpsertEmployeeShift(txCtx, es); upsertErr != nil {
			return upsertErr
		}

		// Write the change into the fixed weekly pattern so it repeats every week.
		// "leave"/"vacation"/"hourly" are always one-off states, never a pattern.
		if !permanent || (shiftStatus != "off" && shiftStatus != "working") {
			return nil
		}

		dayOfWeek := int(shiftDate.UTC().Weekday()) // 0=Sunday … 6=Saturday
		isOff := shiftStatus == "off"
		if tmplErr := s.scheduleRepo.UpsertTemplateForDay(txCtx, employeeID, dayOfWeek, isOff, shiftID); tmplErr != nil {
			return fmt.Errorf("weekly pattern update failed (change would not repeat next week): %w", tmplErr)
		}
		// Push it onto future weeks that were already materialised from the old pattern.
		if resyncErr := s.scheduleRepo.ResyncGeneratedForWeekday(txCtx, employeeID, dayOfWeek, shiftDate, shiftStatus, shiftID); resyncErr != nil {
			return fmt.Errorf("future weeks were not updated: %w", resyncErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return es, nil
}

// GetEmployeePattern returns the employee's fixed weekly pattern, always as 7 days.
// Days with no stored pattern fall back to default_shift_id / weekly_off_days so the
// caller sees the schedule that will actually be materialised.
func (s *ScheduleService) GetEmployeePattern(ctx context.Context, employeeID uuid.UUID) ([]models.PatternDay, error) {
	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}

	templates, err := s.scheduleRepo.GetTemplatesByEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("get weekly pattern: %w", err)
	}
	byDay := map[int]models.ScheduleTemplate{}
	for _, t := range templates {
		byDay[t.DayOfWeek] = t
	}

	out := make([]models.PatternDay, 0, 7)
	for day := 0; day < 7; day++ {
		var tmpl *models.ScheduleTemplate
		if t, ok := byDay[day]; ok {
			tmpl = &t
		}
		status, shiftID := baselineForDay(*emp, day, tmpl)
		out = append(out, models.PatternDay{
			DayOfWeek: day,
			IsOff:     status == "off",
			ShiftID:   shiftID,
		})
	}
	return out, nil
}

// SetEmployeePattern replaces the employee's whole fixed weekly pattern and pushes it
// onto every future pattern-derived day. Past days and manual/leave days are untouched.
func (s *ScheduleService) SetEmployeePattern(ctx context.Context, employeeID uuid.UUID, days []models.PatternDay, actorRole string) ([]models.PatternDay, error) {
	if len(days) != 7 {
		return nil, fmt.Errorf("weekly pattern must contain exactly 7 days, got %d", len(days))
	}

	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}
	if err := assertCanManage(actorRole, emp); err != nil {
		return nil, err
	}

	seen := map[int]bool{}
	for i := range days {
		d := &days[i]
		if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
			return nil, fmt.Errorf("invalid day_of_week: %d", d.DayOfWeek)
		}
		if seen[d.DayOfWeek] {
			return nil, fmt.Errorf("duplicate day_of_week: %d", d.DayOfWeek)
		}
		seen[d.DayOfWeek] = true

		if d.IsOff {
			d.ShiftID = nil
			continue
		}
		if d.ShiftID == nil {
			return nil, fmt.Errorf("shift_id is required for working day %d", d.DayOfWeek)
		}
		if _, err := s.shiftRepo.GetByID(ctx, *d.ShiftID); err != nil {
			return nil, fmt.Errorf("shift not found for day %d: %w", d.DayOfWeek, err)
		}
	}

	if err := s.scheduleRepo.ReplaceEmployeePattern(ctx, employeeID, days); err != nil {
		return nil, fmt.Errorf("save weekly pattern: %w", err)
	}

	// Apply from tomorrow onward: today's roster is already in use by the floor.
	from := today()
	for _, d := range days {
		status := "working"
		if d.IsOff {
			status = "off"
		}
		if err := s.scheduleRepo.ResyncGeneratedForWeekday(ctx, employeeID, d.DayOfWeek, from, status, d.ShiftID); err != nil {
			return nil, fmt.Errorf("apply pattern to future weeks: %w", err)
		}
	}

	return s.GetEmployeePattern(ctx, employeeID)
}

// DeleteEmployeeShift removes an employee shift record (e.g. removing an off-day assignment).
// The day reverts to the employee's fixed weekly pattern on the next read.
func (s *ScheduleService) DeleteEmployeeShift(ctx context.Context, shiftID uuid.UUID) error {
	return s.scheduleRepo.DeleteEmployeeShift(ctx, shiftID)
}

func strPtr(s string) *string {
	return &s
}
