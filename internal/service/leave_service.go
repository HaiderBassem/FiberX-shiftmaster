package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/notification"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/temporal"
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
	// now is the clock used by date validation. Injectable so tests can pin
	// the after-midnight overnight-shift cases; production uses time.Now.
	now func() time.Time
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
		now:              time.Now,
	}
}

func parseTimeStr(t string) (time.Time, error) {
	if len(t) > 5 {
		t = t[:5]
	}
	return time.Parse("15:04", t)
}

// hourlyLeaveHours computes the length of an hourly leave from its stored
// clocks. An end at or before the start crosses midnight — the same convention
// shifts themselves use (a 16:30→00:30 shift ends the next day), so the last
// hour of an overnight shift (23:30→00:30) is representable as a single row on
// the shift's business date. Plain end-minus-start subtraction used to make
// such a window come out negative and be rejected.
func hourlyLeaveHours(startClock, endClock string) (float64, error) {
	d, err := temporal.HourlyLeaveDuration(startClock, endClock)
	if err != nil {
		return 0, err
	}
	return d.Hours(), nil
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

// leaveAbsoluteWindow materialises a leave into concrete instants in the
// business timezone so windows can be compared for overlap. A full-day leave
// occupies [start 00:00, end+1 00:00); an hourly leave occupies its clock
// window anchored to its (single) business date, crossing midnight when the
// end clock is at or before the start clock.
func leaveAbsoluteWindow(l *models.Leave) (time.Time, time.Time, bool) {
	loc := temporal.Location()
	sy, sm, sd := l.StartDate.UTC().Date()

	if l.StartTime != nil && l.EndTime != nil {
		sh, smin, err1 := temporal.ParseClock(*l.StartTime)
		eh, emin, err2 := temporal.ParseClock(*l.EndTime)
		if err1 != nil || err2 != nil {
			return time.Time{}, time.Time{}, false
		}
		start := time.Date(sy, sm, sd, sh, smin, 0, 0, loc)
		end := time.Date(sy, sm, sd, eh, emin, 0, 0, loc)
		if !end.After(start) {
			end = end.Add(24 * time.Hour)
		}
		return start, end, true
	}

	ey, em, ed := l.EndDate.UTC().Date()
	start := time.Date(sy, sm, sd, 0, 0, 0, 0, loc)
	end := time.Date(ey, em, ed, 0, 0, 0, 0, loc).Add(24 * time.Hour)
	return start, end, true
}

// ValidateLeaveRequest enforces every business rule for a new leave request
// without creating anything. RequestLeave calls it before inserting; the
// assistant's approval flow calls it once to build the preview and relies on
// RequestLeave running it again at execution time, so a request that stopped
// being valid between preview and approval is refused rather than executed.
func (s *LeaveService) ValidateLeaveRequest(ctx context.Context, leave *models.Leave) error {
	if leave.EndDate.Before(leave.StartDate) {
		return fmt.Errorf("end date cannot be before start date")
	}

	// Resolve the leave type. The type decides whether this is an hourly or a
	// day request, so requests without a resolvable type cannot skip the rules.
	var leaveType *models.LeaveType
	if leave.LeaveTypeID != uuid.Nil {
		lt, err := s.leaveTypeRepo.GetByID(ctx, leave.LeaveTypeID)
		if err != nil {
			return fmt.Errorf("leave type not found")
		}
		leaveType = lt
	}

	isHourly := leaveType != nil && leaveType.Unit == "hours"
	hasTimes := leave.StartTime != nil && leave.EndTime != nil && *leave.StartTime != "" && *leave.EndTime != ""

	// Times and type must agree. An hourly request without a window is
	// meaningless, and a window on a day-based type used to be silently
	// counted as whole days while still rendering as an hourly leave.
	if isHourly && !hasTimes {
		return fmt.Errorf("this leave type is hourly: start_time and end_time are required")
	}
	if !isHourly && hasTimes {
		return fmt.Errorf("this leave type covers whole days: remove start_time and end_time")
	}
	// An hourly leave belongs to exactly one business date. The clock window
	// itself may cross midnight — that is still the same business date, the
	// same convention overnight shifts use.
	if isHourly && !leave.StartDate.UTC().Truncate(24*time.Hour).Equal(leave.EndDate.UTC().Truncate(24*time.Hour)) {
		return fmt.Errorf("an hourly leave must start and end on the same date")
	}

	// Requested amount, in the unit the balance is kept in.
	requestedAmount := 0.0
	if isHourly {
		hours, err := hourlyLeaveHours(*leave.StartTime, *leave.EndTime)
		if err != nil {
			return err
		}
		requestedAmount = hours
	} else {
		for d := leave.StartDate.UTC().Truncate(24 * time.Hour); !d.After(leave.EndDate.UTC().Truncate(24 * time.Hour)); d = d.AddDate(0, 0, 1) {
			requestedAmount += 1.0
		}
	}

	// Past-date rule, evaluated on the business calendar (Asia/Baghdad). The
	// old check truncated in UTC, which put the day boundary at 03:00 local.
	// Day leaves must start today or later. An hourly leave is judged by its
	// window's END instant, so the last hour of an overnight shift can still
	// be requested after midnight while the shift is running.
	now := s.now().In(temporal.Location())
	todayBusiness := temporal.BusinessDate(now)
	startDay := leave.StartDate.UTC().Truncate(24 * time.Hour)
	if isHourly {
		if startDay.Before(todayBusiness.AddDate(0, 0, -1)) {
			return fmt.Errorf("cannot request leave for past dates")
		}
		if _, end, ok := leaveAbsoluteWindow(leave); ok && !end.After(now) {
			return fmt.Errorf("cannot request an hourly leave for a time that has already passed")
		}
	} else if startDay.Before(todayBusiness) {
		return fmt.Errorf("cannot request leave for past dates")
	}

	if leaveType != nil && leaveType.DaysPerYear > 0 {
		year := leave.StartDate.Year()
		month := 0
		if leaveType.ResetCycle == "monthly" {
			month = int(leave.StartDate.Month())
		}

		balance, err := s.leaveBalanceRepo.GetByEmployeeLeaveTypeAndYear(ctx, leave.EmployeeID, leave.LeaveTypeID, year, month)
		allocated := float64(leaveType.DaysPerYear)
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
								if hours, hErr := hourlyLeaveHours(*l.StartTime, *l.EndTime); hErr == nil {
									pendingAmount += hours
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

	// Double-booking guard: the same employee cannot hold two live leaves
	// whose absolute windows intersect. The candidate fetch reaches one day
	// each side because an hourly window on the previous date can cross
	// midnight into this one.
	newStart, newEnd, ok := leaveAbsoluteWindow(leave)
	if ok {
		rangeFrom := leave.StartDate.UTC().Truncate(24*time.Hour).AddDate(0, 0, -1)
		rangeTo := leave.EndDate.UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)
		existing, exErr := s.leaveRepo.GetActiveByEmployeeInRange(ctx, leave.EmployeeID, rangeFrom, rangeTo)
		if exErr == nil {
			for i := range existing {
				exStart, exEnd, exOK := leaveAbsoluteWindow(&existing[i])
				if !exOK {
					continue
				}
				if newStart.Before(exEnd) && exStart.Before(newEnd) {
					return fmt.Errorf("you already have a leave request covering %s (status: %s)",
						exStart.Format("2006-01-02 15:04"), existing[i].Status)
				}
			}
		}
	}

	// Department daily caps, unless this leave type is explicitly exempt.
	// The exemption used to be inferred by matching the type's name against
	// "Emergency", so renaming or translating the type silently removed it.
	bypassesDailyLimit := leaveType != nil && leaveType.BypassesDailyLimit

	if !bypassesDailyLimit && emp.DepartmentID != nil {
		dept, err := s.departmentRepo.GetByID(ctx, *emp.DepartmentID)
		if err == nil {
			capIsHourly := leaveType != nil && leaveType.IsHourly
			limit := dept.MaxLeavesPerDay
			if capIsHourly {
				limit = dept.MaxHourlyLeavesPerDay
			}

			if limit != nil {
				for d := leave.StartDate; !d.After(leave.EndDate); d = d.AddDate(0, 0, 1) {
					shiftID := emp.DefaultShiftID
					es, err := s.scheduleRepo.GetEmployeeShift(ctx, emp.ID, d)
					if err == nil && es != nil && es.ShiftID != nil {
						shiftID = es.ShiftID
					}

					if shiftID != nil {
						overlappingCount, err := s.leaveRepo.GetOverlappingLeavesCountByShift(ctx, *emp.DepartmentID, *shiftID, d, capIsHourly)
						if err == nil && overlappingCount >= *limit {
							leaveTypeStr := "leaves"
							if capIsHourly {
								leaveTypeStr = "hourly leaves"
							}
							return fmt.Errorf("the maximum number of allowed %s per shift for your department has been reached on %s", leaveTypeStr, d.Format("2006-01-02"))
						}
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
	if err := s.ValidateLeaveRequest(ctx, leave); err != nil {
		return err
	}

	emp, err := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
	if err != nil {
		return fmt.Errorf("employee not found: %w", err)
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

	// emp comes from a lookup whose error is ignored above; it can be nil.
	if s.pushService != nil && emp != nil {
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

	if s.pushService != nil && emp != nil {
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

	// Release the days back to the employee's fixed weekly pattern. Deleting the
	// leave-owned rows is enough: the next read re-materialises them from the
	// pattern, so there is no second place that has to know what the pattern says.
	start := leave.StartDate.UTC().Truncate(24 * time.Hour)
	end := leave.EndDate.UTC().Truncate(24 * time.Hour)

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		es, esErr := s.scheduleRepo.GetEmployeeShift(ctx, leave.EmployeeID, d)
		if esErr != nil || es == nil {
			continue
		}
		// Only reclaim days this leave owns; a manual edit made during the leave stands.
		if es.Source == models.ShiftSourceLeave {
			_ = s.scheduleRepo.DeleteEmployeeShift(ctx, es.ID)
		}
	}

	// Decrement used amount in leave balances
	if leave.LeaveTypeID != uuid.Nil {
		leaveType, err := s.leaveTypeRepo.GetByID(ctx, leave.LeaveTypeID)
		if err == nil && leaveType != nil {
			amountToRevert := 0.0
			isHourly := leaveType.Unit == "hours"
			if leave.StartTime != nil && leave.EndTime != nil && isHourly {
				if hours, hErr := hourlyLeaveHours(*leave.StartTime, *leave.EndTime); hErr == nil {
					amountToRevert = hours
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

// CancelPendingLeave cancels a leave request that has not yet been fully approved.
func (s *LeaveService) CancelPendingLeave(ctx context.Context, leaveID uuid.UUID, employeeID uuid.UUID) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}

	if leave.EmployeeID != employeeID {
		return fmt.Errorf("unauthorized to cancel this leave")
	}

	if leave.Status != "pending" && leave.Status != "approved_by_team_leader" {
		return fmt.Errorf("only pending or team-leader-approved leaves can be cancelled by the employee")
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

	isHourly := false
	if leave.LeaveTypeID != uuid.Nil {
		if lt, err := s.leaveTypeRepo.GetByID(ctx, leave.LeaveTypeID); err == nil && lt != nil {
			isHourly = lt.Unit == "hours"
		}
	}

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

		var shiftStatus = "leave"
		if isHourly {
			shiftStatus = "hourly"
		}

		leaveReasonPtr := &leaveReason
		es := &models.EmployeeShift{
			ScheduleID:  ws.ID,
			EmployeeID:  leave.EmployeeID,
			ShiftID:     shiftID,
			ShiftDate:   d,
			ShiftStatus: shiftStatus,
			LeaveReason: leaveReasonPtr,
			CreatedBy:   &approverID,
			Source:      models.ShiftSourceLeave,
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
				if hours, hErr := hourlyLeaveHours(*leave.StartTime, *leave.EndTime); hErr == nil {
					amountTaken = hours
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

	// Only requests still awaiting a decision can be rejected. A fully
	// approved leave has shifts and balance applied; the verb for undoing
	// those is CancelApprovedLeave, which reverts them. Reject used to accept
	// any status, silently flipping approved/cancelled leaves to rejected
	// while leaving their side effects in place.
	if leave.Status != "pending" && leave.Status != "approved_by_team_leader" {
		return fmt.Errorf("leave cannot be rejected in its current status: %s", leave.Status)
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

	if s.pushService != nil && emp != nil {
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

// SendUpcomingLeaveReminders sends reminders to team leaders for upcoming leaves that were requested >= 3 days in advance.
func (s *LeaveService) SendUpcomingLeaveReminders(ctx context.Context) error {
	leaves, err := s.leaveRepo.GetLeavesForReminders(ctx)
	if err != nil {
		return fmt.Errorf("failed to get leaves for reminders: %w", err)
	}

	for _, leave := range leaves {
		emp, err := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
		if err != nil {
			continue // Skip if employee not found
		}

		if emp.DepartmentID == nil {
			continue
		}

		// Find team leaders for this department
		teamLeaders, err := s.employeeRepo.GetByRole(ctx, "team_leader")
		if err != nil {
			continue
		}

		// Send notification to team leaders in the same department
		title := "Upcoming Leave Reminder"
		msg := fmt.Sprintf("Reminder: Employee %s %s has a leave starting in 2 days (%s).", emp.FirstName, emp.LastName, leave.StartDate.Format("2006-01-02"))
		entityType := "leave"

		for _, tl := range teamLeaders {
			if tl.DepartmentID != nil && *tl.DepartmentID == *emp.DepartmentID {
				// Send In-App notification
				_ = s.notifService.SendNotification(ctx, &models.Notification{
					RecipientID:       tl.ID,
					Type:              "reminder",
					Title:             title,
					Message:           &msg,
					RelatedEntityType: &entityType,
					RelatedEntityID:   &leave.ID,
					Priority:          "medium",
				})

				// Send Email
				s.emailService.SendEmailAsync([]string{tl.Email}, title, msg)

				// Send Push Notification
				if s.pushService != nil {
					_ = s.pushService.SendToEmployee(ctx, tl.ID, title, msg, "/approvals")
				}
			}
		}

		// Mark reminder as sent
		_ = s.leaveRepo.MarkReminderSent(ctx, leave.ID)
	}

	return nil
}
