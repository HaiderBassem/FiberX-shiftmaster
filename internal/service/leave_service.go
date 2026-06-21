package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/notification"
	"shiftmaster-backend/internal/repository"
)

// LeaveService handles leave request business logic with approval chain.
type LeaveService struct {
	leaveRepo        repository.LeaveRepository
	employeeRepo     repository.EmployeeRepository
	departmentRepo   repository.DepartmentRepository
	scheduleRepo     repository.ScheduleRepository
	leaveBalanceRepo repository.LeaveBalanceRepository
	leaveTypeRepo    repository.LeaveTypeRepository
	notifService     *NotificationService
	emailService     *EmailService
	pushService      notification.PushService
}

func NewLeaveService(
	leaveRepo repository.LeaveRepository,
	employeeRepo repository.EmployeeRepository,
	departmentRepo repository.DepartmentRepository,
	scheduleRepo repository.ScheduleRepository,
	leaveBalanceRepo repository.LeaveBalanceRepository,
	leaveTypeRepo repository.LeaveTypeRepository,
	notifService *NotificationService,
	emailService *EmailService,
	pushService notification.PushService,
) *LeaveService {
	return &LeaveService{
		leaveRepo:        leaveRepo,
		employeeRepo:     employeeRepo,
		departmentRepo:   departmentRepo,
		scheduleRepo:     scheduleRepo,
		leaveBalanceRepo: leaveBalanceRepo,
		leaveTypeRepo:    leaveTypeRepo,
		notifService:     notifService,
		emailService:     emailService,
		pushService:      pushService,
	}
}

func (s *LeaveService) GetEmployeeLeaveBalances(ctx context.Context, empID uuid.UUID, year int) ([]models.EmployeeLeaveBalance, error) {
	return s.leaveBalanceRepo.GetByEmployeeAndYear(ctx, empID, year)
}

func (s *LeaveService) UpdateEmployeeLeaveBalance(ctx context.Context, empID, leaveTypeID uuid.UUID, year int, month int, allocatedAmount float64) error {
	return s.leaveBalanceRepo.UpdateAllocatedDays(ctx, empID, leaveTypeID, year, month, allocatedAmount)
}

// SyncLeaveBalances assigns the days_per_year from all active leave types to all active employees for the specified year.
func (s *LeaveService) SyncLeaveBalances(ctx context.Context, year int) error {
	// Get all active employees
	employees, err := s.employeeRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active employees: %w", err)
	}

	// Get all active leave types
	leaveTypes, err := s.leaveTypeRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active leave types: %w", err)
	}

	// For each employee and leave type, sync allocated days
	for _, emp := range employees {
		for _, lt := range leaveTypes {
			// Only sync if days_per_year > 0 to avoid creating unnecessary zero-balance records
			if lt.DaysPerYear > 0 {
				if lt.ResetCycle == "monthly" {
					for m := 1; m <= 12; m++ {
						err := s.leaveBalanceRepo.SyncAllocatedDays(ctx, emp.ID, lt.ID, year, m, float64(lt.DaysPerYear))
						if err != nil {
							fmt.Printf("Error syncing monthly balance for emp %s and type %s (month %d): %v\n", emp.ID, lt.ID, m, err)
						}
					}
				} else {
					err := s.leaveBalanceRepo.SyncAllocatedDays(ctx, emp.ID, lt.ID, year, 0, float64(lt.DaysPerYear))
					if err != nil {
						fmt.Printf("Error syncing annual balance for emp %s and type %s: %v\n", emp.ID, lt.ID, err)
					}
				}
			}
		}
	}

	return nil
}

// RequestLeave creates a new leave request.
// - If the requester is a team_leader → notifies managers directly (skips TL step).
// - If the requester is an employee → notifies all team leaders in the same department.
func (s *LeaveService) RequestLeave(ctx context.Context, leave *models.Leave) error {
	// Validate dates
	if leave.EndDate.Before(leave.StartDate) {
		return fmt.Errorf("end date cannot be before start date")
	}
	if leave.StartDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return fmt.Errorf("cannot request leave for past dates")
	}

	// Get leave type if specified
	var leaveType *models.LeaveType
	var err error
	if leave.LeaveTypeID != uuid.Nil {
		leaveType, err = s.leaveTypeRepo.GetByID(ctx, leave.LeaveTypeID)
		if err != nil {
			return fmt.Errorf("leave type not found")
		}
	}

	// Calculate requested amount (days or hours)
	requestedAmount := 0.0
	isHourly := leaveType != nil && leaveType.Unit == "hours"
	
	if leave.StartTime != nil && leave.EndTime != nil && isHourly {
		startTime, err1 := time.Parse("15:04", *leave.StartTime)
		endTime, err2 := time.Parse("15:04", *leave.EndTime)
		if err1 == nil && err2 == nil {
			requestedAmount = endTime.Sub(startTime).Hours()
			if requestedAmount <= 0 {
				return fmt.Errorf("end time must be after start time")
			}
		}
	} else {
		// Count days inclusive
		for d := leave.StartDate.UTC().Truncate(24 * time.Hour); !d.After(leave.EndDate.UTC().Truncate(24 * time.Hour)); d = d.AddDate(0, 0, 1) {
			requestedAmount += 1.0
		}
	}

	if leaveType != nil && leaveType.DaysPerYear > 0 {
		year := leave.StartDate.Year()
		month := 0
		if leaveType.ResetCycle == "monthly" {
			month = int(leave.StartDate.Month())
		}

		balance, err := s.leaveBalanceRepo.GetByEmployeeLeaveTypeAndYear(ctx, leave.EmployeeID, leave.LeaveTypeID, year, month)
		var allocated float64 = float64(leaveType.DaysPerYear)
		var used float64 = 0
		if err == nil && balance != nil {
			allocated = balance.AllocatedAmount
			used = balance.UsedAmount
		}

		// Calculate pending amount to prevent overdrafting through multiple pending requests
		pendingAmount := 0.0
		existingLeaves, err := s.leaveRepo.GetByEmployee(ctx, leave.EmployeeID)
		if err == nil {
			for _, l := range existingLeaves {
				if l.Status == "pending" || l.Status == "approved_by_team_leader" {
					if l.LeaveTypeID == leave.LeaveTypeID {
						y := l.StartDate.Year()
						m := 0
						if leaveType.ResetCycle == "monthly" {
							m = int(l.StartDate.Month())
						}
						if y == year && m == month {
							if isHourly && l.StartTime != nil && l.EndTime != nil {
								st, err1 := time.Parse("15:04", *l.StartTime)
								en, err2 := time.Parse("15:04", *l.EndTime)
								if err1 == nil && err2 == nil {
									pendingAmount += en.Sub(st).Hours()
								}
							} else if !isHourly {
								for d := l.StartDate.UTC().Truncate(24 * time.Hour); !d.After(l.EndDate.UTC().Truncate(24 * time.Hour)); d = d.AddDate(0, 0, 1) {
									pendingAmount += 1.0
								}
							}
						}
					}
				}
			}
		}

		if used+pendingAmount+requestedAmount > allocated {
			unitStr := "days"
			if isHourly {
				unitStr = "hours"
			}
			return fmt.Errorf("insufficient leave balance. You have %.1f %s remaining (%.1f %s pending)", allocated-used-pendingAmount, unitStr, pendingAmount, unitStr)
		}
	}

	// Validate employee exists
	emp, err := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
	if err != nil {
		return fmt.Errorf("employee not found: %w", err)
	}

	// Check department leave limits (if set) and not an Emergency leave
	isEmergency := false
	if leaveType != nil {
		if leaveType.NameEn == "Emergency" || leaveType.NameEn == "emergency" {
			isEmergency = true
		}
	}

	if !isEmergency && emp.DepartmentID != nil {
		dept, err := s.departmentRepo.GetByID(ctx, *emp.DepartmentID)
		if err == nil && dept.MaxLeavesPerDay != nil {
			overlappingCount, err := s.leaveRepo.GetOverlappingLeavesCount(ctx, *emp.DepartmentID, leave.StartDate, leave.EndDate)
			if err == nil {
				// Each hourly leave is counted as 1 leave for that day.
				if overlappingCount >= *dept.MaxLeavesPerDay {
					return fmt.Errorf("The maximum number of allowed leaves per day for your department has been reached.")
				}
			}
		}
	}

	if err := s.leaveRepo.Create(ctx, leave); err != nil {
		return fmt.Errorf("create leave request: %w", err)
	}

	// Build notification message
	var msg string
	if leave.StartTime != nil && leave.EndTime != nil {
		timeInfo := fmt.Sprintf(" from %s to %s", *leave.StartTime, *leave.EndTime)
		msg = fmt.Sprintf("%s %s requested hourly leave on %s%s", emp.FirstName, emp.LastName, leave.StartDate.Format("2006-01-02"), timeInfo)
	} else {
		msg = fmt.Sprintf("%s %s requested a leave from %s to %s", emp.FirstName, emp.LastName, leave.StartDate.Format("2006-01-02"), leave.EndDate.Format("2006-01-02"))
	}

	actionUrl := "/approvals"

	// Team leader requests go directly to managers — skip TL approval step.
	if emp.Role == "team_leader" {
		var managers []models.Employee
		if emp.DepartmentID != nil {
			deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *emp.DepartmentID)
			for _, e := range deptEmps {
				if e.Role == "manager" {
					managers = append(managers, e)
				}
			}
		}
		// If no manager found in dept, fall back to all managers
		if len(managers) == 0 {
			managers, _ = s.employeeRepo.GetByRole(ctx, "manager")
		}
		for _, mgr := range managers {
			if err := s.notifService.SendNotification(ctx, &models.Notification{
				RecipientID:       mgr.ID,
				SenderID:          &leave.EmployeeID,
				Type:              "leave_request",
				Title:             "New Leave Request (Team Leader)",
				Message:           strPtr(msg),
				RelatedEntityType: strPtr("leave"),
				RelatedEntityID:   &leave.ID,
				Priority:          "high",
				ActionUrl:         &actionUrl,
			}); err != nil {
				fmt.Printf("Failed to send leave notification to manager: %v\n", err)
			}
			
			// Send email to manager
			if mgr.Email != "" {
				s.emailService.SendEmailAsync(
					[]string{mgr.Email},
					"New Leave Request (Team Leader)",
					msg+"\n\nPlease review it in the Approval Center.",
				)
			}
			if s.pushService != nil {
				go func(mID uuid.UUID) {
					_ = s.pushService.SendToEmployee(context.Background(), mID, "New Leave Request (Team Leader)", msg, "/approvals")
				}(mgr.ID)
			}
		}
		return nil
	}

	// Employee requests → notify all team leaders in the same department.
	var teamLeaders []models.Employee
	if emp.DepartmentID != nil {
		deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *emp.DepartmentID)
		for _, e := range deptEmps {
			if e.Role == "team_leader" {
				teamLeaders = append(teamLeaders, e)
			}
		}
	}
	for _, tl := range teamLeaders {
		if err := s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       tl.ID,
			SenderID:          &leave.EmployeeID,
			Type:              "leave_request",
			Title:             "New Leave Request",
			Message:           strPtr(msg),
			RelatedEntityType: strPtr("leave"),
			RelatedEntityID:   &leave.ID,
			Priority:          "high",
			ActionUrl:         &actionUrl,
		}); err != nil {
			fmt.Printf("Failed to send new leave notification: %v\n", err)
		}

		// Send email to team leader
		if tl.Email != "" {
			s.emailService.SendEmailAsync(
				[]string{tl.Email},
				"New Leave Request",
				msg+"\n\nPlease review it in the Approval Center.",
			)
		}
		if s.pushService != nil {
			go func(tID uuid.UUID) {
				_ = s.pushService.SendToEmployee(context.Background(), tID, "New Leave Request", msg, "/approvals")
			}(tl.ID)
		}
	}

	// Send email to department managers
	var managers []models.Employee
	if emp.DepartmentID != nil {
		deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *emp.DepartmentID)
		for _, e := range deptEmps {
			if e.Role == "manager" {
				managers = append(managers, e)
			}
		}
	}
	if len(managers) == 0 {
		managers, _ = s.employeeRepo.GetByRole(ctx, "manager")
	}

	for _, mgr := range managers {
		if mgr.Email != "" {
			s.emailService.SendEmailAsync(
				[]string{mgr.Email},
				"New Leave Request Pending Team Leader Approval",
				msg+"\n\nThis request is currently pending approval by the team leader.",
			)
		}
	}

	return nil
}

// ApproveByTeamLeader grants final approval to a leave request.
// The FIRST team leader to approve immediately finalizes the leave — no other TLs
// or manager approval is required for employee leaves.
func (s *LeaveService) ApproveByTeamLeader(ctx context.Context, leaveID uuid.UUID, teamLeaderID uuid.UUID) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}
	if leave.Status != "pending" {
		return fmt.Errorf("leave is not pending, current status: %s", leave.Status)
	}

	// Check if this TL already approved
	already, _ := s.leaveRepo.HasApproved(ctx, leaveID, teamLeaderID)
	if already {
		return fmt.Errorf("you have already approved this leave request")
	}

	// Get the TL's name for the notification
	tlEmployee, _ := s.employeeRepo.GetByID(ctx, teamLeaderID)
	tlName := "A team leader"
	if tlEmployee != nil {
		tlName = tlEmployee.FirstName + " " + tlEmployee.LastName
	}

	// Record the individual approval
	if err := s.leaveRepo.RecordApproval(ctx, leaveID, teamLeaderID, "team_leader", "approved", nil); err != nil {
		return err
	}

	// Immediately finalize: set status to approved_by_manager so the leave is fully approved.
	if err := s.leaveRepo.UpdateStatus(ctx, leaveID, "approved_by_manager", teamLeaderID, "manager"); err != nil {
		return err
	}

	// Apply leave to employee_shifts (same logic as manager final approval)
	if applyErr := s.applyLeaveToShifts(ctx, leave, teamLeaderID); applyErr != nil {
		fmt.Printf("[LEAVE] Failed to apply leave shifts: %v\n", applyErr)
	}

	// Notify the employee that the leave is fully approved
	if err := s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       leave.EmployeeID,
		SenderID:          &teamLeaderID,
		Type:              "approval",
		Title:             "Leave Approved",
		Message:           strPtr(fmt.Sprintf("%s has approved your leave request. Your leave is now fully approved!", tlName)),
		RelatedEntityType: strPtr("leave"),
		RelatedEntityID:   &leaveID,
		Priority:          "high",
	}); err != nil {
		fmt.Printf("Failed to send employee approval notification: %v\n", err)
	}

	// Send email to employee
	emp, _ := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
	if emp != nil && emp.Email != "" {
		s.emailService.SendEmailAsync(
			[]string{emp.Email},
			"Leave Request Approved",
			fmt.Sprintf("Hello %s,\n\n%s has approved your leave request. Your leave is now fully approved!", emp.FirstName, tlName),
		)
	}

	if s.pushService != nil {
		go func(eID uuid.UUID) {
			_ = s.pushService.SendToEmployee(context.Background(), eID, "Leave Approved", "Your leave request has been fully approved by "+tlName, "/leaves")
		}(emp.ID)
	}

	// Send email to department managers about the approval
	if emp != nil {
		var managers []models.Employee
		if emp.DepartmentID != nil {
			deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *emp.DepartmentID)
			for _, e := range deptEmps {
				if e.Role == "manager" {
					managers = append(managers, e)
				}
			}
		}
		if len(managers) == 0 {
			managers, _ = s.employeeRepo.GetByRole(ctx, "manager")
		}

		var detailMsg string
		if leave.StartTime != nil && leave.EndTime != nil {
			timeInfo := fmt.Sprintf(" from %s to %s", *leave.StartTime, *leave.EndTime)
			detailMsg = fmt.Sprintf("hourly leave on %s%s", leave.StartDate.Format("2006-01-02"), timeInfo)
		} else {
			detailMsg = fmt.Sprintf("leave from %s to %s", leave.StartDate.Format("2006-01-02"), leave.EndDate.Format("2006-01-02"))
		}

		for _, mgr := range managers {
			if mgr.Email != "" {
				s.emailService.SendEmailAsync(
					[]string{mgr.Email},
					"Leave Request Approved by Team Leader",
					fmt.Sprintf("Hello %s,\n\n%s %s's %s has been approved by team leader %s.", mgr.FirstName, emp.FirstName, emp.LastName, detailMsg, tlName),
				)
			}
		}
	}

	return nil
}

// ApproveByManager gives final approval to a leave request.
// Used for team_leader-submitted leaves that go directly to the manager.
func (s *LeaveService) ApproveByManager(ctx context.Context, leaveID uuid.UUID, managerID uuid.UUID) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}
	// Accept both "pending" (TL-submitted leave routed directly to manager)
	// and "approved_by_team_leader" (legacy path, kept for safety).
	if leave.Status != "pending" && leave.Status != "approved_by_team_leader" {
		return fmt.Errorf("leave cannot be approved in its current status: %s", leave.Status)
	}

	// Record the manager approval
	if err := s.leaveRepo.RecordApproval(ctx, leaveID, managerID, "manager", "approved", nil); err != nil {
		return err
	}

	if err := s.leaveRepo.UpdateStatus(ctx, leaveID, "approved_by_manager", managerID, "manager"); err != nil {
		return err
	}

	// Apply leave to employee_shifts
	if applyErr := s.applyLeaveToShifts(ctx, leave, managerID); applyErr != nil {
		fmt.Printf("[LEAVE] Failed to apply leave shifts: %v\n", applyErr)
	}

	// Notify employee about final approval
	if err := s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       leave.EmployeeID,
		SenderID:          &managerID,
		Type:              "approval",
		Title:             "Leave Approved",
		Message:           strPtr("Your leave request has been fully approved!"),
		RelatedEntityType: strPtr("leave"),
		RelatedEntityID:   &leaveID,
		Priority:          "high",
	}); err != nil {
		fmt.Printf("Failed to send manager approval notification: %v\n", err)
	}

	// Send email to employee
	emp, _ := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
	if emp != nil && emp.Email != "" {
		s.emailService.SendEmailAsync(
			[]string{emp.Email},
			"Leave Request Approved",
			fmt.Sprintf("Hello %s,\n\nYour leave request has been fully approved by a manager!", emp.FirstName),
		)
	}

	if s.pushService != nil {
		go func(eID uuid.UUID) {
			_ = s.pushService.SendToEmployee(context.Background(), eID, "Leave Approved", "Your leave request has been fully approved by a manager!", "/leaves")
		}(emp.ID)
	}

	return nil
}

// CancelApprovedLeave cancels an already approved leave and reverts its effects.
func (s *LeaveService) CancelApprovedLeave(ctx context.Context, leaveID uuid.UUID, cancelledBy uuid.UUID, role string) error {
	// Only Managers can cancel manager-approved leaves. Team leaders can cancel TL-approved leaves (which are technically still pending final approval but have their shifts applied or wait).
	// Actually, wait, leave status: "approved_by_team_leader", "approved_by_manager", "approved".
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}

	if leave.Status != "approved_by_manager" && leave.Status != "approved_by_team_leader" && leave.Status != "approved" {
		return fmt.Errorf("leave is not in an approved state: %s", leave.Status)
	}

	if leave.Status == "approved_by_manager" || leave.Status == "approved" {
		if role != "manager" && role != "admin" && role != "team_leader" {
			return fmt.Errorf("only managers and team leaders can cancel a fully approved leave")
		}
	}

	// Change status to cancelled
	if err := s.leaveRepo.UpdateStatus(ctx, leaveID, "cancelled", cancelledBy, role); err != nil {
		return fmt.Errorf("failed to update leave status to cancelled: %w", err)
	}

	// Revert employee_shifts back to template defaults
	start := leave.StartDate.UTC().Truncate(24 * time.Hour)
	end := leave.EndDate.UTC().Truncate(24 * time.Hour)

	templates, err := s.scheduleRepo.GetTemplatesByEmployee(ctx, leave.EmployeeID)
	if err == nil {
		tmplMap := make(map[int]*models.ScheduleTemplate)
		for i := range templates {
			tmplMap[templates[i].DayOfWeek] = &templates[i]
		}

		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			es, err := s.scheduleRepo.GetEmployeeShift(ctx, leave.EmployeeID, d)
			if err == nil && es != nil {
				// Revert to template
				tmpl := tmplMap[int(d.Weekday())]
				if tmpl != nil {
					status := "working"
					if tmpl.IsOff {
						status = "off"
					}
					es.ShiftStatus = status
					es.ShiftID = tmpl.ShiftID
					es.LeaveReason = nil
					_ = s.scheduleRepo.UpdateEmployeeShift(ctx, es)
				} else {
					// Fallback to delete if no template
					_ = s.scheduleRepo.DeleteEmployeeShift(ctx, es.ID)
				}
			}
		}
	}

	// Decrement used amount in leave balances
	if leave.LeaveTypeID != uuid.Nil {
		leaveType, err := s.leaveTypeRepo.GetByID(ctx, leave.LeaveTypeID)
		if err == nil && leaveType != nil {
			amountToRevert := 0.0
			isHourly := leaveType.Unit == "hours"
			if leave.StartTime != nil && leave.EndTime != nil && isHourly {
				st, err1 := time.Parse("15:04", *leave.StartTime)
				en, err2 := time.Parse("15:04", *leave.EndTime)
				if err1 == nil && err2 == nil {
					amountToRevert = en.Sub(st).Hours()
				}
			} else {
				for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
					amountToRevert += 1.0
				}
			}
			
			year := start.Year()
			month := 0
			if leaveType.ResetCycle == "monthly" {
				month = int(start.Month())
			}
			// decrement by amountToRevert
			_ = s.leaveBalanceRepo.IncrementUsedDays(ctx, leave.EmployeeID, leave.LeaveTypeID, year, month, -amountToRevert)
		}
	}

	// Notify employee
	_ = s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       leave.EmployeeID,
		SenderID:          &cancelledBy,
		Type:              "shift_change",
		Title:             "Leave Cancelled",
		Message:           strPtr("Your approved leave request has been cancelled by management."),
		RelatedEntityType: strPtr("leave"),
		RelatedEntityID:   &leaveID,
		Priority:          "high",
	})

	// Send Email
	emp, _ := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
	if emp != nil && emp.Email != "" {
		s.emailService.SendEmailAsync(
			[]string{emp.Email},
			"Leave Cancelled",
			fmt.Sprintf("Hello %s,\n\nYour approved leave request (from %s to %s) has been cancelled by management. Please check your schedule for updates.", emp.FirstName, start.Format("2006-01-02"), end.Format("2006-01-02")),
		)
	}

	return nil
}

// CancelPendingLeave cancels a leave request that has not yet been approved.
func (s *LeaveService) CancelPendingLeave(ctx context.Context, leaveID uuid.UUID, employeeID uuid.UUID) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}

	if leave.EmployeeID != employeeID {
		return fmt.Errorf("unauthorized to cancel this leave")
	}

	if leave.Status != "pending" {
		return fmt.Errorf("only pending leaves can be cancelled")
	}

	if err := s.leaveRepo.UpdateStatus(ctx, leaveID, "cancelled", employeeID, "employee"); err != nil {
		return fmt.Errorf("failed to cancel leave: %w", err)
	}

	return nil
}

// applyLeaveToShifts upserts employee_shifts rows to shift_status='leave' for every
// day in the leave date range. This is shared between TL and manager approval paths.
func (s *LeaveService) applyLeaveToShifts(ctx context.Context, leave *models.Leave, approverID uuid.UUID) error {
	leaveReason := "leave"
	if leave.Reason != nil && *leave.Reason != "" {
		leaveReason = *leave.Reason
	}

	emp, _ := s.employeeRepo.GetByID(ctx, leave.EmployeeID)

	start := leave.StartDate.UTC().Truncate(24 * time.Hour)
	end := leave.EndDate.UTC().Truncate(24 * time.Hour)

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		weekStart := d
		for weekStart.Weekday() != time.Sunday {
			weekStart = weekStart.AddDate(0, 0, -1)
		}
		weekEnd := weekStart.AddDate(0, 0, 6)

		ws, wsErr := s.scheduleRepo.GetWeeklySchedule(ctx, weekStart)
		if wsErr != nil {
			ws = &models.WeeklySchedule{
				WeekStartDate: weekStart,
				WeekEndDate:   weekEnd,
				Status:        "draft",
			}
			if createErr := s.scheduleRepo.CreateWeeklySchedule(ctx, ws); createErr != nil {
				fmt.Printf("[LEAVE] Failed to create weekly schedule for %s: %v\n", weekStart.Format("2006-01-02"), createErr)
				continue
			}
		}

		var shiftID *uuid.UUID
		existing, existErr := s.scheduleRepo.GetEmployeeShift(ctx, leave.EmployeeID, d)
		if existErr == nil && existing != nil {
			shiftID = existing.ShiftID
		} else if emp != nil {
			shiftID = emp.DefaultShiftID
		}

		leaveReasonPtr := &leaveReason
		es := &models.EmployeeShift{
			ScheduleID:  ws.ID,
			EmployeeID:  leave.EmployeeID,
			ShiftID:     shiftID,
			ShiftDate:   d,
			ShiftStatus: "leave",
			LeaveReason: leaveReasonPtr,
			CreatedBy:   &approverID,
		}
		if upsertErr := s.scheduleRepo.UpsertEmployeeShift(ctx, es); upsertErr != nil {
			fmt.Printf("[LEAVE] Failed to upsert employee shift for %s on %s: %v\n", leave.EmployeeID, d.Format("2006-01-02"), upsertErr)
		}
	}

	// Increment used amount in leave balances
	if leave.LeaveTypeID != uuid.Nil {
		leaveType, ltErr := s.leaveTypeRepo.GetByID(ctx, leave.LeaveTypeID)
		if ltErr == nil && leaveType != nil {
			amountTaken := 0.0
			isHourly := leaveType.Unit == "hours"
			if leave.StartTime != nil && leave.EndTime != nil && isHourly {
				st, err1 := time.Parse("15:04", *leave.StartTime)
				en, err2 := time.Parse("15:04", *leave.EndTime)
				if err1 == nil && err2 == nil {
					amountTaken = en.Sub(st).Hours()
				}
			} else {
				for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
					amountTaken += 1.0
				}
			}
			
			year := start.Year()
			month := 0
			if leaveType.ResetCycle == "monthly" {
				month = int(start.Month())
			}
			
			err := s.leaveBalanceRepo.IncrementUsedDays(ctx, leave.EmployeeID, leave.LeaveTypeID, year, month, amountTaken)
			if err != nil {
				// If not found, create it with the allocated amount based on leaveType
				_ = s.leaveBalanceRepo.UpsertBalance(ctx, &models.EmployeeLeaveBalance{
					EmployeeID:      leave.EmployeeID,
					LeaveTypeID:     leave.LeaveTypeID,
					Year:            year,
					Month:           month,
					AllocatedAmount: float64(leaveType.DaysPerYear),
					UsedAmount:      amountTaken,
				})
			}
		}
	}

	return nil
}

// RejectLeave rejects a leave request with a reason.
func (s *LeaveService) RejectLeave(ctx context.Context, leaveID uuid.UUID, rejectedBy uuid.UUID, rejectorRole string, reason string) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}

	// Record the rejection
	notesPtr := &reason
	if err := s.leaveRepo.RecordApproval(ctx, leaveID, rejectedBy, rejectorRole, "rejected", notesPtr); err != nil {
		return err
	}

	if err := s.leaveRepo.Reject(ctx, leaveID, rejectedBy, reason); err != nil {
		return err
	}

	// Notify employee
	if err := s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       leave.EmployeeID,
		SenderID:          &rejectedBy,
		Type:              "approval",
		Title:             "Leave Request Rejected",
		Message:           strPtr(fmt.Sprintf("Your leave request has been rejected. Reason: %s", reason)),
		RelatedEntityType: strPtr("leave"),
		RelatedEntityID:   &leaveID,
		Priority:          "high",
	}); err != nil {
		fmt.Printf("Failed to send rejection notification: %v\n", err)
	}

	// Send email to employee
	emp, _ := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
	if emp != nil && emp.Email != "" {
		s.emailService.SendEmailAsync(
			[]string{emp.Email},
			"Leave Request Rejected",
			fmt.Sprintf("Hello %s,\n\nYour leave request starting on %s has been rejected.\nReason: %s", emp.FirstName, leave.StartDate.Format("2006-01-02"), reason),
		)
	}

	if s.pushService != nil {
		go func(eID uuid.UUID) {
			_ = s.pushService.SendToEmployee(context.Background(), eID, "Leave Rejected", "Your leave request has been rejected.", "/leaves")
		}(emp.ID)
	}

	// Send email to department managers about the rejection
	if emp != nil {
		var managers []models.Employee
		if emp.DepartmentID != nil {
			deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *emp.DepartmentID)
			for _, e := range deptEmps {
				if e.Role == "manager" {
					managers = append(managers, e)
				}
			}
		}
		if len(managers) == 0 {
			managers, _ = s.employeeRepo.GetByRole(ctx, "manager")
		}

		rejectorEmployee, _ := s.employeeRepo.GetByID(ctx, rejectedBy)
		rejectorName := "A team leader/manager"
		if rejectorEmployee != nil {
			rejectorName = rejectorEmployee.FirstName + " " + rejectorEmployee.LastName
		}

		var detailMsg string
		if leave.StartTime != nil && leave.EndTime != nil {
			timeInfo := fmt.Sprintf(" from %s to %s", *leave.StartTime, *leave.EndTime)
			detailMsg = fmt.Sprintf("hourly leave on %s%s", leave.StartDate.Format("2006-01-02"), timeInfo)
		} else {
			detailMsg = fmt.Sprintf("leave from %s to %s", leave.StartDate.Format("2006-01-02"), leave.EndDate.Format("2006-01-02"))
		}

		for _, mgr := range managers {
			if mgr.Email != "" {
				s.emailService.SendEmailAsync(
					[]string{mgr.Email},
					"Leave Request Rejected",
					fmt.Sprintf("Hello %s,\n\n%s %s's %s has been rejected by %s.\nReason: %s", mgr.FirstName, emp.FirstName, emp.LastName, detailMsg, rejectorName, reason),
				)
			}
		}
	}

	return nil
}

// GetEmployeeLeaves returns all leaves for an employee.
func (s *LeaveService) GetEmployeeLeaves(ctx context.Context, employeeID uuid.UUID) ([]models.Leave, error) {
	return s.leaveRepo.GetByEmployee(ctx, employeeID)
}

// GetPendingForApproval returns leaves awaiting approval based on approver's role and department.
func (s *LeaveService) GetPendingForApproval(ctx context.Context, approverID uuid.UUID) ([]models.Leave, error) {
	approver, err := s.employeeRepo.GetByID(ctx, approverID)
	if err != nil {
		return nil, fmt.Errorf("approver not found: %w", err)
	}
	return s.leaveRepo.GetPendingForApproval(ctx, approver.Role, approver.DepartmentID)
}

// GetShiftCoveragePreview helps managers visualize staffing levels before approving a leave.
func (s *LeaveService) GetShiftCoveragePreview(ctx context.Context, shiftID uuid.UUID, date time.Time) (*models.ShiftCoverage, error) {
	return s.scheduleRepo.GetShiftCoveragePreview(ctx, shiftID, date)
}

// GetPendingLeavesRich returns pending leaves with employee details, strictly for the approver's department.
func (s *LeaveService) GetPendingLeavesRich(ctx context.Context, approverID uuid.UUID) ([]models.PendingLeaveRich, error) {
	approver, err := s.employeeRepo.GetByID(ctx, approverID)
	if err != nil {
		return nil, fmt.Errorf("approver not found: %w", err)
	}
	return s.leaveRepo.GetPendingLeavesRich(ctx, approver.Role, approver.DepartmentID)
}

// GetLeaveHistory returns all leaves with their approval details, optionally filtered by department.
func (s *LeaveService) GetLeaveHistory(ctx context.Context, departmentID *uuid.UUID) ([]models.LeaveHistoryRow, error) {
	return s.leaveRepo.GetLeaveHistory(ctx, departmentID)
}

