package service

import (
	"context"
	"fmt"
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
	notifService *NotificationService
	db           *database.DB
}

func NewScheduleService(
	scheduleRepo repository.ScheduleRepository,
	employeeRepo repository.EmployeeRepository,
	shiftRepo repository.ShiftRepository,
	notifService *NotificationService,
	db *database.DB,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		employeeRepo: employeeRepo,
		shiftRepo:    shiftRepo,
		notifService: notifService,
		db:           db,
	}
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
func (s *ScheduleService) GetAvailableReplacements(ctx context.Context, date time.Time) ([]models.Employee, error) {
	return s.scheduleRepo.GetAvailableReplacements(ctx, date)
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

	// Notify the replacement employee
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

// GetEmployeeShifts returns an employee's shifts for a date range.
func (s *ScheduleService) GetEmployeeShifts(ctx context.Context, employeeID uuid.UUID, from, to time.Time) ([]models.EmployeeShift, error) {
	return s.scheduleRepo.GetEmployeeShiftsByEmployee(ctx, employeeID, from, to)
}

// GetDailyShifts returns all shifts for a specific date.
func (s *ScheduleService) GetDailyShifts(ctx context.Context, date time.Time) ([]models.EmployeeShift, error) {
	return s.scheduleRepo.GetEmployeeShiftsByDate(ctx, date)
}

// SetEmployeeShift upserts a single employee shift for a day.
// If the weekly schedule record for that week doesn't exist, it will be created as a draft.
func (s *ScheduleService) SetEmployeeShift(ctx context.Context, employeeID uuid.UUID, shiftDate time.Time, shiftID *uuid.UUID, shiftStatus string, leaveReason *string, createdBy uuid.UUID) (*models.EmployeeShift, error) {
	valid := map[string]bool{"working": true, "off": true, "leave": true, "vacation": true}
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
	weekStart := shiftDate
	for weekStart.Weekday() != time.Sunday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekEnd := weekStart.AddDate(0, 0, 6)

	// Ensure weekly schedule exists (draft by default)
	ws, wsErr := s.scheduleRepo.GetWeeklySchedule(ctx, weekStart)
	if wsErr != nil {
		ws = &models.WeeklySchedule{WeekStartDate: weekStart, WeekEndDate: weekEnd, Status: "draft"}
		if err := s.scheduleRepo.CreateWeeklySchedule(ctx, ws); err != nil {
			return nil, fmt.Errorf("create weekly schedule: %w", err)
		}
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
	return es, nil
}

func strPtr(s string) *string {
	return &s
}
