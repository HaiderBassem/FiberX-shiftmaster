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

func normalizeWeekStart(d time.Time) time.Time {
	d = d.UTC().Truncate(24 * time.Hour)
	for d.Weekday() != time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// EnsureWeekSchedule fills missing off/working rows for the week (from previous week or templates),
// then overlays fully approved leaves. Safe to call repeatedly — never overwrites existing rows
// during base fill; leave overlay upserts approved leave days.
func (s *ScheduleService) EnsureWeekSchedule(ctx context.Context, refDate time.Time) error {
	weekStart := normalizeWeekStart(refDate)
	weekEnd := weekStart.AddDate(0, 0, 6)

	ws, err := s.getOrCreateWeeklySchedule(ctx, weekStart, weekEnd)
	if err != nil {
		return err
	}

	prevWeekStart := weekStart.AddDate(0, 0, -7)
	prevWeekEnd := prevWeekStart.AddDate(0, 0, 6)
	prevShifts, _ := s.scheduleRepo.GetEmployeeShiftsInRange(ctx, prevWeekStart, prevWeekEnd)
	prevByEmployeeDay := map[uuid.UUID]map[int]models.EmployeeShift{}
	for _, ps := range prevShifts {
		st := strings.ToLower(ps.ShiftStatus)
		if st != "working" && st != "off" {
			continue
		}
		dow := int(ps.ShiftDate.UTC().Weekday())
		if prevByEmployeeDay[ps.EmployeeID] == nil {
			prevByEmployeeDay[ps.EmployeeID] = make(map[int]models.EmployeeShift)
		}
		prevByEmployeeDay[ps.EmployeeID][dow] = ps
	}

	employees, err := s.employeeRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("get active employees: %w", err)
	}

	for _, emp := range employees {
		templates, tmplErr := s.scheduleRepo.GetTemplatesByEmployee(ctx, emp.ID)
		tmplByDay := map[int]*models.ScheduleTemplate{}
		if tmplErr == nil {
			for i := range templates {
				tmplByDay[templates[i].DayOfWeek] = &templates[i]
			}
		}

		for day := 0; day < 7; day++ {
			shiftDate := weekStart.AddDate(0, 0, day)
			if _, existErr := s.scheduleRepo.GetEmployeeShift(ctx, emp.ID, shiftDate); existErr == nil {
				continue
			}

			status := "off"
			var shiftID *uuid.UUID

			if prevDay, ok := prevByEmployeeDay[emp.ID][day]; ok {
				status = strings.ToLower(prevDay.ShiftStatus)
				shiftID = prevDay.ShiftID
			} else if tmpl, ok := tmplByDay[day]; ok {
				if tmpl.IsOff {
					status = "off"
					shiftID = nil
				} else {
					status = "working"
					shiftID = tmpl.ShiftID
				}
			} else if emp.WeeklyOffDays >= 0 && emp.WeeklyOffDays == day {
				status = "off"
			} else if emp.DefaultShiftID != nil {
				status = "working"
				shiftID = emp.DefaultShiftID
			}

			if status == "working" && shiftID == nil {
				status = "off"
			}

			es := &models.EmployeeShift{
				ScheduleID:  ws.ID,
				EmployeeID:  emp.ID,
				ShiftID:     shiftID,
				ShiftDate:   shiftDate,
				ShiftStatus: status,
			}
			if upsertErr := s.scheduleRepo.UpsertEmployeeShift(ctx, es); upsertErr != nil {
				return fmt.Errorf("seed shift for %s on %s: %w", emp.EmployeeCode, shiftDate.Format("2006-01-02"), upsertErr)
			}
		}
	}

	return s.applyApprovedLeavesForRange(ctx, ws, weekStart, weekEnd)
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
			if leave.LeaveTypeNameEn != nil && (strings.ToLower(*leave.LeaveTypeNameEn) == "hourly" || strings.ToLower(*leave.LeaveTypeNameEn) == "زمنية") {
				// Hourly leaves are still "leave" in the DB enum, but we tag the reason
				// so the frontend can distinguish them.
				leaveReason = "[hourly] " + leaveReason
			}

			es := &models.EmployeeShift{
				ScheduleID:  ws.ID,
				EmployeeID:  leave.EmployeeID,
				ShiftID:     shiftID,
				ShiftDate:   d,
				ShiftStatus: shiftStatus,
				LeaveReason: leaveReasonPtr,
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
	if err := s.EnsureWeekSchedule(ctx, date); err != nil {
		return nil, fmt.Errorf("ensure week schedule: %w", err)
	}
	return s.scheduleRepo.GetEmployeeShiftsByDate(ctx, date, departmentID)
}

// GetEmployeeShifts returns an employee's shifts for a date range.
func (s *ScheduleService) GetEmployeeShifts(ctx context.Context, employeeID uuid.UUID, from, to time.Time) ([]models.EmployeeShift, error) {
	if err := s.EnsureWeekSchedule(ctx, from); err != nil {
		return nil, fmt.Errorf("ensure week schedule: %w", err)
	}
	// If range spans a second week, ensure that week too.
	secondWeekStart := normalizeWeekStart(to)
	firstWeekStart := normalizeWeekStart(from)
	if !secondWeekStart.Equal(firstWeekStart) {
		if err := s.EnsureWeekSchedule(ctx, to); err != nil {
			return nil, fmt.Errorf("ensure week schedule: %w", err)
		}
	}
	return s.scheduleRepo.GetEmployeeShiftsByEmployee(ctx, employeeID, from, to)
}

// GetDepartmentShiftsInRange gets shifts for a whole department in a date range.
func (s *ScheduleService) GetDepartmentShiftsInRange(ctx context.Context, from, to time.Time, departmentID uuid.UUID) ([]models.EmployeeShiftExtended, error) {
	if err := s.EnsureWeekSchedule(ctx, from); err != nil {
		return nil, fmt.Errorf("ensure week schedule: %w", err)
	}
	secondWeekStart := normalizeWeekStart(to)
	firstWeekStart := normalizeWeekStart(from)
	if !secondWeekStart.Equal(firstWeekStart) {
		if err := s.EnsureWeekSchedule(ctx, to); err != nil {
			return nil, fmt.Errorf("ensure week schedule: %w", err)
		}
	}
	return s.scheduleRepo.GetDepartmentShiftsInRange(ctx, from, to, departmentID)
}

// SetEmployeeShift upserts a single employee shift for a day.
// If the weekly schedule record for that week doesn't exist, it will be created as a draft.
func (s *ScheduleService) SetEmployeeShift(ctx context.Context, employeeID uuid.UUID, shiftDate time.Time, shiftID *uuid.UUID, shiftStatus string, leaveReason *string, createdBy uuid.UUID, creatorRole string) (*models.EmployeeShift, error) {
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

	// Enforce role hierarchy: TLs cannot set schedule for Managers/Admins, Managers cannot set for Admins.
	if creatorRole == "team_leader" && (emp.Role == "manager" || emp.Role == "admin") {
		return nil, fmt.Errorf("team leaders cannot modify schedules for managers or admins")
	}
	if creatorRole == "manager" && emp.Role == "admin" {
		return nil, fmt.Errorf("managers cannot modify schedules for admins")
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
	}

	if err := s.scheduleRepo.UpsertEmployeeShift(ctx, es); err != nil {
		return nil, err
	}

	// ── Persist off/working into the schedule template so the setting
	// survives future GenerateWeeklySchedule calls automatically.
	// "leave" and "vacation" are intentionally excluded — they are
	// one-off temporary states, not permanent schedule patterns.
	if shiftStatus == "off" || shiftStatus == "working" {
		dayOfWeek := int(shiftDate.Weekday()) // 0=Sunday … 6=Saturday
		isOff := shiftStatus == "off"
		if tmplErr := s.scheduleRepo.UpsertTemplateForDay(ctx, employeeID, dayOfWeek, isOff, shiftID); tmplErr != nil {
			// Log the failure but don't roll back the shift — the weekly entry
			// was already saved successfully.
			_ = tmplErr
		}
	}

	return es, nil
}

// DeleteEmployeeShift removes an employee shift record (e.g. removing an off-day assignment).
func (s *ScheduleService) DeleteEmployeeShift(ctx context.Context, shiftID uuid.UUID) error {
	return s.scheduleRepo.DeleteEmployeeShift(ctx, shiftID)
}

func strPtr(s string) *string {
	return &s
}
