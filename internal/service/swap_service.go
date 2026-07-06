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

// SwapService handles the 2-step shift swap approval flow:
// Step 1: Requester sends swap request → Target employee accepts/rejects
// Step 2: After employee acceptance → Manager/Team Leader approves/rejects
type SwapService struct {
	swapRepo     repository.SwapRepository
	scheduleRepo repository.ScheduleRepository
	employeeRepo repository.EmployeeRepository
	taskRepo     repository.TaskRepository
	notifService *NotificationService
	emailService *EmailService
	db           *database.DB
}

func NewSwapService(
	swapRepo repository.SwapRepository,
	scheduleRepo repository.ScheduleRepository,
	employeeRepo repository.EmployeeRepository,
	taskRepo repository.TaskRepository,
	notifService *NotificationService,
	emailService *EmailService,
	db *database.DB,
) *SwapService {
	return &SwapService{
		swapRepo:     swapRepo,
		scheduleRepo: scheduleRepo,
		employeeRepo: employeeRepo,
		taskRepo:     taskRepo,
		notifService: notifService,
		emailService: emailService,
		db:           db,
	}
}

// RequestSwap creates a swap request and notifies the target employee.
func (s *SwapService) RequestSwap(ctx context.Context, swap *models.ShiftSwap) error {
	// Validate requester and target are different
	if swap.RequesterID == swap.TargetEmployeeID {
		return fmt.Errorf("cannot swap with yourself")
	}

	// Validate both employees exist and are active
	requester, err := s.employeeRepo.GetByID(ctx, swap.RequesterID)
	if err != nil {
		return fmt.Errorf("requester not found: %w", err)
	}
	target, err := s.employeeRepo.GetByID(ctx, swap.TargetEmployeeID)
	if err != nil {
		return fmt.Errorf("target employee not found: %w", err)
	}
	if requester.Status != "active" || target.Status != "active" {
		return fmt.Errorf("both employees must be active")
	}
	// Swaps are allowed for employee<->employee and team_leader<->team_leader only.
	if (requester.Role != "employee" && requester.Role != "team_leader") ||
		(target.Role != "employee" && target.Role != "team_leader") {
		return fmt.Errorf("swaps are only allowed for employees and team leaders")
	}
	if requester.Role != target.Role {
		return fmt.Errorf("swap requester and target must have the same role")
	}
	if requester.DepartmentID == nil || target.DepartmentID == nil || *requester.DepartmentID != *target.DepartmentID {
		return fmt.Errorf("swap is allowed only within the same department")
	}
	if !sameDayOrAfter(swap.ShiftDate, time.Now()) {
		return fmt.Errorf("swap date must be today or in the future")
	}

	// Prevent conflicting open requests for same date among either employee.
	var openConflicts int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM shift_swaps
		WHERE shift_date = $1
		  AND status::text IN ('pending', 'employee_accepted')
		  AND (
			requester_id IN ($2, $3) OR
			target_employee_id IN ($2, $3)
		  )`,
		swap.ShiftDate, swap.RequesterID, swap.TargetEmployeeID,
	).Scan(&openConflicts); err != nil {
		return fmt.Errorf("check swap conflicts: %w", err)
	}
	if openConflicts > 0 {
		return fmt.Errorf("a pending swap already exists for one of these employees on this date")
	}

	// Phase 3 contract: frontend may omit shift_id. If missing, infer it from the requester's shift.
	if swap.ShiftID == uuid.Nil {
		requesterShift, err := s.scheduleRepo.GetEmployeeShift(ctx, swap.RequesterID, swap.ShiftDate)
		if err == nil && requesterShift != nil && requesterShift.ShiftID != nil {
			swap.ShiftID = *requesterShift.ShiftID
		} else {
			// Fallback: allow swap creation even if no daily row exists yet,
			// by using requester's default shift.
			if requester.DefaultShiftID == nil {
				return fmt.Errorf("requester has no shift configured for the requested date")
			}
			swap.ShiftID = *requester.DefaultShiftID
		}
	}

	// Create the swap request
	if err := s.swapRepo.Create(ctx, swap); err != nil {
		return fmt.Errorf("create swap request: %w", err)
	}

	// Notify target employee: "Do you accept this swap?"
	if err := s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       swap.TargetEmployeeID,
		SenderID:          &swap.RequesterID,
		Type:              "shift_change",
		Title:             "Shift Swap Request",
		Message:           strPtr(fmt.Sprintf("%s %s wants to swap shifts with you on %s. Do you accept?", requester.FirstName, requester.LastName, swap.ShiftDate.Format("2006-01-02"))),
		RelatedEntityType: strPtr("swap"),
		RelatedEntityID:   &swap.ID,
		Priority:          "high",
	}); err != nil {
		fmt.Printf("[SWAP] Failed to notify target employee %s about swap request: %v\n", swap.TargetEmployeeID, err)
	}

	return nil
}

// EmployeeRespond handles the target employee's acceptance or rejection.
// If accepted, the request moves to manager approval.
// If rejected, the swap is closed.
func (s *SwapService) EmployeeRespond(ctx context.Context, swapID uuid.UUID, employeeID uuid.UUID, accepted bool) error {
	swap, err := s.swapRepo.GetByID(ctx, swapID)
	if err != nil {
		return fmt.Errorf("swap not found: %w", err)
	}

	// Only the target employee can respond
	if swap.TargetEmployeeID != employeeID {
		return fmt.Errorf("only the target employee can respond to this swap")
	}
	if swap.Status != "pending" {
		return fmt.Errorf("swap is not pending, current status: %s", swap.Status)
	}

	if err := s.swapRepo.EmployeeRespond(ctx, swapID, accepted); err != nil {
		return err
	}

	if !accepted {
		// Notify requester about rejection
		if err := s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       swap.RequesterID,
			SenderID:          &employeeID,
			Type:              "shift_change",
			Title:             "Swap Rejected",
			Message:           strPtr("Your shift swap request has been rejected by the target employee."),
			RelatedEntityType: strPtr("swap"),
			RelatedEntityID:   &swapID,
			Priority:          "medium",
		}); err != nil {
			fmt.Printf("[SWAP] Failed to notify requester %s about employee rejection: %v\n", swap.RequesterID, err)
		}
		return nil
	}

	// Employee accepted.
	// If the requester is a team_leader → notify managers directly for approval.
	// If the requester is a regular employee → notify team leaders as before.
	requester, _ := s.employeeRepo.GetByID(ctx, swap.RequesterID)
	requesterIsTeamLeader := requester != nil && requester.Role == "team_leader"

	target, _ := s.employeeRepo.GetByID(ctx, swap.TargetEmployeeID)

	if requesterIsTeamLeader {
		// TL swap → notify managers
		var managers []models.Employee
		if target != nil && target.DepartmentID != nil {
			deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *target.DepartmentID)
			for _, e := range deptEmps {
				if e.Role == "manager" {
					managers = append(managers, e)
				}
			}
		}
		for _, mgr := range managers {
			if err := s.notifService.SendNotification(ctx, &models.Notification{
				RecipientID:       mgr.ID,
				SenderID:          &employeeID,
				Type:              "approval",
				Title:             "Shift Swap Approval Required (Team Leader)",
				Message:           strPtr("A team leader shift swap has been agreed upon by both employees and needs your approval"),
				RelatedEntityType: strPtr("swap"),
				RelatedEntityID:   &swapID,
				Priority:          "high",
			}); err != nil {
				fmt.Printf("[SWAP] Failed to notify manager %s about TL swap approval: %v\n", mgr.ID, err)
			}
		}
	} else {
		// Employee swap → notify team leaders of the SAME department
		var teamLeaders []models.Employee
		if target != nil && target.DepartmentID != nil {
			deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *target.DepartmentID)
			for _, e := range deptEmps {
				if e.Role == "team_leader" {
					teamLeaders = append(teamLeaders, e)
				}
			}
		}
		for _, tl := range teamLeaders {
			if err := s.notifService.SendNotification(ctx, &models.Notification{
				RecipientID:       tl.ID,
				SenderID:          &employeeID,
				Type:              "approval",
				Title:             "Shift Swap Approval Required",
				Message:           strPtr("A shift swap has been agreed upon by both employees and needs your approval"),
				RelatedEntityType: strPtr("swap"),
				RelatedEntityID:   &swapID,
				Priority:          "high",
			}); err != nil {
				fmt.Printf("[SWAP] Failed to notify team leader %s about swap approval needed: %v\n", tl.ID, err)
			}
		}
	}

	// Notify requester that target accepted
	awaiting := "team leader"
	if requesterIsTeamLeader {
		awaiting = "manager"
	}
	if err := s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       swap.RequesterID,
		SenderID:          &employeeID,
		Type:              "shift_change",
		Title:             "Swap Accepted",
		Message:           strPtr(fmt.Sprintf("Your swap request has been accepted! Awaiting %s approval.", awaiting)),
		RelatedEntityType: strPtr("swap"),
		RelatedEntityID:   &swapID,
		Priority:          "medium",
	}); err != nil {
		fmt.Printf("[SWAP] Failed to notify requester %s about employee acceptance: %v\n", swap.RequesterID, err)
	}

	return nil
}

// ApproveSwap approves the swap and actually swaps the shifts in the schedule.
// - Employee swaps: approved by team_leader only.
// - Team leader swaps: approved by manager only.
func (s *SwapService) ApproveSwap(ctx context.Context, swapID uuid.UUID, approverID uuid.UUID, approverRole string) error {
	if approverRole != "team_leader" && approverRole != "manager" {
		return fmt.Errorf("only team_leader or manager can approve swaps")
	}

	swap, err := s.swapRepo.GetByID(ctx, swapID)
	if err != nil {
		return err
	}
	if swap.Status != "employee_accepted" {
		return fmt.Errorf("swap is not pending approval, current status: %s", swap.Status)
	}

	err = s.db.ExecTx(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		// Approve in shift_swaps table
		if err := s.swapRepo.ManagerApprove(txCtx, swapID, approverID, approverRole); err != nil {
			return err
		}

		// Final approval by team leader -> apply the swap immediately.
		if approverRole == "team_leader" {
			// Ensure both shift rows exist on the swap date.
			requesterShift, err := s.ensureShiftForSwap(txCtx, swap.RequesterID, swap.ShiftDate, approverID)
			if err != nil {
				return fmt.Errorf("requester shift not available: %w", err)
			}
			targetShift, err := s.ensureShiftForSwap(txCtx, swap.TargetEmployeeID, swap.ShiftDate, approverID)
			if err != nil {
				return fmt.Errorf("target shift not available: %w", err)
			}

			// Coverage rule: avoid approvals that would leave a shift with zero working employees.
			// (e.g. the shift currently has only one working employee)
			if err := s.validateSwapCoverage(txCtx, requesterShift, targetShift, swap.ShiftDate); err != nil {
				return err
			}

			// Swap shift identity + status so off-day/working state is actually exchanged.
			tempStatus := requesterShift.ShiftStatus
			requesterShift.ShiftStatus = targetShift.ShiftStatus
			targetShift.ShiftStatus = tempStatus

			tempReason := requesterShift.LeaveReason
			requesterShift.LeaveReason = targetShift.LeaveReason
			targetShift.LeaveReason = tempReason

			// Swap shift_id between the two employees.
			tempShiftID := requesterShift.ShiftID
			requesterShift.ShiftID = targetShift.ShiftID
			targetShift.ShiftID = tempShiftID

			if err := s.scheduleRepo.UpdateEmployeeShift(txCtx, requesterShift); err != nil {
				return fmt.Errorf("update requester shift: %w", err)
			}
			if err := s.scheduleRepo.UpdateEmployeeShift(txCtx, targetShift); err != nil {
				return fmt.Errorf("update target shift: %w", err)
			}

			// Transfer task assignments between the two employees on the swap date
			if err := s.taskRepo.SwapAssignmentsBetweenEmployees(txCtx, swap.RequesterID, swap.TargetEmployeeID, swap.ShiftDate); err != nil {
				return fmt.Errorf("swap task assignments: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	if approverRole == "team_leader" {
		// Notify both employees
		if err := s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       swap.RequesterID,
			SenderID:          &approverID,
			Type:              "shift_change",
			Title:             "Swap Approved!",
			Message:           strPtr(fmt.Sprintf("Your shift swap for %s has been approved and applied!", swap.ShiftDate.Format("2006-01-02"))),
			RelatedEntityType: strPtr("swap"),
			RelatedEntityID:   &swapID,
			Priority:          "high",
		}); err != nil {
			fmt.Printf("[SWAP] Failed to notify requester %s about approval: %v\n", swap.RequesterID, err)
		}
		if err := s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       swap.TargetEmployeeID,
			SenderID:          &approverID,
			Type:              "shift_change",
			Title:             "Swap Approved!",
			Message:           strPtr(fmt.Sprintf("Your shift swap for %s has been approved and applied!", swap.ShiftDate.Format("2006-01-02"))),
			RelatedEntityType: strPtr("swap"),
			RelatedEntityID:   &swapID,
			Priority:          "high",
		}); err != nil {
			fmt.Printf("[SWAP] Failed to notify target %s about approval: %v\n", swap.TargetEmployeeID, err)
		}
	}

	return nil
}

// CancelApprovedSwap cancels an already approved swap and reverts the shifts and tasks back to original.
func (s *SwapService) CancelApprovedSwap(ctx context.Context, swapID uuid.UUID, cancelledBy uuid.UUID, role string) error {
	swap, err := s.swapRepo.GetByID(ctx, swapID)
	if err != nil {
		return err
	}

	if swap.Status != "approved" {
		return fmt.Errorf("swap is not in an approved state: %s", swap.Status)
	}

	if role != "manager" && role != "admin" {
		// If it's a team leader swap, maybe TL shouldn't cancel it?
		// But let's allow TL to cancel if they have access.
		if role != "team_leader" {
			return fmt.Errorf("only managers or team leaders can cancel a swap")
		}
	}

	err = s.db.ExecTx(ctx, func(txCtx context.Context, tx pgx.Tx) error {
		// Change status to cancelled
		// Update status directly
		if _, dbErr := s.db.Exec(txCtx, `UPDATE shift_swaps SET status='cancelled', updated_at=CURRENT_TIMESTAMP WHERE id=$1`, swapID); dbErr != nil {
			return dbErr
		}

		// Revert shifts:
		// Just swap them back again!
		requesterShift, reqErr := s.scheduleRepo.GetEmployeeShift(txCtx, swap.RequesterID, swap.ShiftDate)
		targetShift, trgErr := s.scheduleRepo.GetEmployeeShift(txCtx, swap.TargetEmployeeID, swap.ShiftDate)

		if reqErr == nil && trgErr == nil && requesterShift != nil && targetShift != nil {
			// Swap shift identity + status back
			tempStatus := requesterShift.ShiftStatus
			requesterShift.ShiftStatus = targetShift.ShiftStatus
			targetShift.ShiftStatus = tempStatus

			tempReason := requesterShift.LeaveReason
			requesterShift.LeaveReason = targetShift.LeaveReason
			targetShift.LeaveReason = tempReason

			tempShiftID := requesterShift.ShiftID
			requesterShift.ShiftID = targetShift.ShiftID
			targetShift.ShiftID = tempShiftID

			if updateErr := s.scheduleRepo.UpdateEmployeeShift(txCtx, requesterShift); updateErr != nil {
				return updateErr
			}
			if updateErr := s.scheduleRepo.UpdateEmployeeShift(txCtx, targetShift); updateErr != nil {
				return updateErr
			}

			// Swap task assignments back
			if taskErr := s.taskRepo.SwapAssignmentsBetweenEmployees(txCtx, swap.RequesterID, swap.TargetEmployeeID, swap.ShiftDate); taskErr != nil {
				return taskErr
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Notify both employees
	msg := fmt.Sprintf("Your approved shift swap for %s has been cancelled by management. Please check your schedule.", swap.ShiftDate.Format("2006-01-02"))
	
	_ = s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       swap.RequesterID,
		SenderID:          &cancelledBy,
		Type:              "shift_change",
		Title:             "Swap Cancelled",
		Message:           strPtr(msg),
		RelatedEntityType: strPtr("swap"),
		RelatedEntityID:   &swapID,
		Priority:          "high",
	})
	_ = s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       swap.TargetEmployeeID,
		SenderID:          &cancelledBy,
		Type:              "shift_change",
		Title:             "Swap Cancelled",
		Message:           strPtr(msg),
		RelatedEntityType: strPtr("swap"),
		RelatedEntityID:   &swapID,
		Priority:          "high",
	})

	// Send emails
	reqEmp, _ := s.employeeRepo.GetByID(ctx, swap.RequesterID)
	if reqEmp != nil && reqEmp.Email != "" {
		s.emailService.SendEmailAsync([]string{reqEmp.Email}, "Swap Cancelled", fmt.Sprintf("Hello %s,\n\n%s", reqEmp.FirstName, msg))
	}
	trgEmp, _ := s.employeeRepo.GetByID(ctx, swap.TargetEmployeeID)
	if trgEmp != nil && trgEmp.Email != "" {
		s.emailService.SendEmailAsync([]string{trgEmp.Email}, "Swap Cancelled", fmt.Sprintf("Hello %s,\n\n%s", trgEmp.FirstName, msg))
	}

	return nil
}

// RejectSwap rejects a swap request.
// - Employee swaps: rejected by team_leader.
// - Team leader swaps: rejected by manager.
func (s *SwapService) RejectSwap(ctx context.Context, swapID uuid.UUID, approverID uuid.UUID, approverRole string) error {
	if approverRole != "team_leader" && approverRole != "manager" {
		return fmt.Errorf("only team_leader or manager can reject swaps")
	}

	swap, err := s.swapRepo.GetByID(ctx, swapID)
	if err != nil {
		return fmt.Errorf("swap not found: %w", err)
	}

	if err := s.swapRepo.Reject(ctx, swapID, approverID, approverRole); err != nil {
		return err
	}

	// Notify both employees
	for _, empID := range []uuid.UUID{swap.RequesterID, swap.TargetEmployeeID} {
		if err := s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       empID,
			SenderID:          &approverID,
			Type:              "shift_change",
			Title:             "Swap Rejected",
			Message:           strPtr("The shift swap request has been rejected by the team leader."),
			RelatedEntityType: strPtr("swap"),
			RelatedEntityID:   &swapID,
			Priority:          "medium",
		}); err != nil {
			fmt.Printf("[SWAP] Failed to notify employee %s about rejection: %v\n", empID, err)
		}
	}

	return nil
}

// CancelSwap cancels a swap request (by requester only).
func (s *SwapService) CancelSwap(ctx context.Context, swapID uuid.UUID, requesterID uuid.UUID) error {
	swap, err := s.swapRepo.GetByID(ctx, swapID)
	if err != nil {
		return fmt.Errorf("swap not found: %w", err)
	}
	if swap.RequesterID != requesterID {
		return fmt.Errorf("only the requester can cancel this swap")
	}
	if swap.Status != "pending" && swap.Status != "employee_accepted" {
		return fmt.Errorf("can only cancel pending swaps")
	}
	return s.swapRepo.Cancel(ctx, swapID)
}

// GetMySwapRequests returns swap requests created by an employee.
func (s *SwapService) GetMySwapRequests(ctx context.Context, employeeID uuid.UUID) ([]models.ShiftSwap, error) {
	return s.swapRepo.GetByRequester(ctx, employeeID)
}

// GetPendingSwapsForMe returns swaps waiting for my acceptance.
func (s *SwapService) GetPendingSwapsForMe(ctx context.Context, employeeID uuid.UUID) ([]models.ShiftSwap, error) {
	return s.swapRepo.GetPendingForEmployee(ctx, employeeID)
}

// GetPendingSwapsForManager returns swaps waiting for manager/leader approval.
func (s *SwapService) GetPendingSwapsForManager(ctx context.Context, approverID uuid.UUID) ([]models.ShiftSwap, error) {
	approver, err := s.employeeRepo.GetByID(ctx, approverID)
	if err != nil {
		return nil, fmt.Errorf("approver not found: %w", err)
	}
	return s.swapRepo.GetPendingForManager(ctx, approver.Role, approver.DepartmentID)
}

// GetSwapHistory returns swap history for the manager's department.
func (s *SwapService) GetSwapHistory(ctx context.Context, approverID uuid.UUID) ([]models.ShiftSwap, error) {
	approver, err := s.employeeRepo.GetByID(ctx, approverID)
	if err != nil {
		return nil, fmt.Errorf("approver not found: %w", err)
	}
	return s.swapRepo.GetHistoryForManager(ctx, approver.Role, approver.DepartmentID)
}

// GetEligibleSwapTargets returns eligible employees in the same department across all shifts, indicating their 'off' status.
func (s *SwapService) GetEligibleSwapTargets(ctx context.Context, employeeID uuid.UUID, date time.Time) ([]models.SwapEligibleEmployee, error) {
	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}
	if emp.DepartmentID == nil {
		return []models.SwapEligibleEmployee{}, nil
	}
	all, err := s.scheduleRepo.GetSwapEligibleEmployees(ctx, *emp.DepartmentID, employeeID, date)
	if err != nil {
		return nil, err
	}

	allowedRole := "employee"
	if emp.Role == "team_leader" {
		allowedRole = "team_leader"
	}
	filtered := make([]models.SwapEligibleEmployee, 0, len(all))
	for _, candidate := range all {
		if candidate.Role == allowedRole {
			filtered = append(filtered, candidate)
		}
	}
	return filtered, nil
}

// GetEligibleShiftSwapTargets returns employees in the same department whose shift
// on `date` differs from the requester's effective shift. Used for "swap by shift" mode.
func (s *SwapService) GetEligibleShiftSwapTargets(ctx context.Context, employeeID uuid.UUID, date time.Time) ([]models.SwapEligibleEmployee, error) {
	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}
	if emp.DepartmentID == nil {
		return []models.SwapEligibleEmployee{}, nil
	}

	// Determine the requester's effective shift on the given date.
	var requesterShiftID *uuid.UUID
	dailyShift, err := s.scheduleRepo.GetEmployeeShift(ctx, employeeID, date)
	if err == nil && dailyShift != nil {
		requesterShiftID = dailyShift.ShiftID
	} else {
		// Fallback: use the employee's default shift
		requesterShiftID = emp.DefaultShiftID
	}

	all, err := s.scheduleRepo.GetDeptEmployeesNotInSameShift(ctx, *emp.DepartmentID, employeeID, requesterShiftID, date)
	if err != nil {
		return nil, err
	}

	allowedRole := "employee"
	if emp.Role == "team_leader" {
		allowedRole = "team_leader"
	}
	filtered := make([]models.SwapEligibleEmployee, 0, len(all))
	for _, candidate := range all {
		if candidate.Role == allowedRole {
			filtered = append(filtered, candidate)
		}
	}
	return filtered, nil
}

func sameDayOrAfter(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	adate := time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC)
	bdate := time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	return !adate.Before(bdate)
}

func (s *SwapService) ensureShiftForSwap(ctx context.Context, employeeID uuid.UUID, shiftDate time.Time, createdBy uuid.UUID) (*models.EmployeeShift, error) {
	existing, err := s.scheduleRepo.GetEmployeeShift(ctx, employeeID, shiftDate)
	if err == nil && existing != nil {
		return existing, nil
	}

	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}

	weekStart := shiftDate
	for weekStart.Weekday() != time.Sunday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	weekEnd := weekStart.AddDate(0, 0, 6)

	ws, wsErr := s.scheduleRepo.GetWeeklySchedule(ctx, weekStart)
	if wsErr != nil || ws == nil {
		ws = &models.WeeklySchedule{
			WeekStartDate: weekStart,
			WeekEndDate:   weekEnd,
			Status:        "draft",
		}
		if err := s.scheduleRepo.CreateWeeklySchedule(ctx, ws); err != nil {
			return nil, fmt.Errorf("create weekly schedule: %w", err)
		}
	}

	// If daily shift row is missing, try to infer it from schedule templates first.
	// This is required so team-leader approvals can work even when only one employee
	// has a row for this date.
	dow := int(shiftDate.Weekday()) // Go: Sunday=0..Saturday=6
	var (
		templateFound bool
		shiftStatus   = "working"
		shiftID       *uuid.UUID
	)

	templates, tmplErr := s.scheduleRepo.GetTemplatesByEmployee(ctx, employeeID)
	if tmplErr == nil {
		for _, tmpl := range templates {
			if tmpl.DayOfWeek == dow {
				templateFound = true
				if tmpl.IsOff {
					shiftStatus = "off"
					shiftID = nil
				} else {
					shiftStatus = "working"
					shiftID = tmpl.ShiftID
				}
				break
			}
		}
	}

	// Fallback to default shift (if template didn't exist).
	if !templateFound {
		if emp.DefaultShiftID == nil {
			shiftStatus = "off"
			shiftID = nil
		} else {
			shiftStatus = "working"
			shiftID = emp.DefaultShiftID
		}
	}

	// If template says working but shift_id is missing, fallback to default shift.
	if shiftStatus == "working" && shiftID == nil && emp.DefaultShiftID != nil {
		shiftID = emp.DefaultShiftID
	}

	es := &models.EmployeeShift{
		ScheduleID:  ws.ID,
		EmployeeID:  employeeID,
		ShiftID:     shiftID,
		ShiftDate:   shiftDate,
		ShiftStatus: shiftStatus,
		CreatedBy:   &createdBy,
	}
	if err := s.scheduleRepo.UpsertEmployeeShift(ctx, es); err != nil {
		return nil, fmt.Errorf("upsert employee shift: %w", err)
	}
	return es, nil
}

// validateSwapCoverage prevents approving swaps that would result in a shift
// having zero working employees after the swap/status exchange.
func (s *SwapService) validateSwapCoverage(
	ctx context.Context,
	requesterShift *models.EmployeeShift,
	targetShift *models.EmployeeShift,
	shiftDate time.Time,
) error {
	affectedShiftIDs := make(map[uuid.UUID]struct{})
	if requesterShift.ShiftID != nil {
		affectedShiftIDs[*requesterShift.ShiftID] = struct{}{}
	}
	if targetShift.ShiftID != nil {
		affectedShiftIDs[*targetShift.ShiftID] = struct{}{}
	}

	if len(affectedShiftIDs) == 0 {
		return nil
	}

	// After swap:
	// - requester becomes target's shift_id + shift_status
	// - target becomes requester's shift_id + shift_status
	requesterAfterShiftID := targetShift.ShiftID
	requesterAfterStatus := targetShift.ShiftStatus
	targetAfterShiftID := requesterShift.ShiftID
	targetAfterStatus := requesterShift.ShiftStatus

	for shiftID := range affectedShiftIDs {
		coverage, err := s.scheduleRepo.GetShiftCoveragePreview(ctx, shiftID, shiftDate)
		if err != nil {
			return fmt.Errorf("coverage preview failed: %w", err)
		}

		workingAfter := coverage.TotalWorking

		// Remove current working employees for this shift.
		if requesterShift.ShiftID != nil && *requesterShift.ShiftID == shiftID && requesterShift.ShiftStatus == "working" {
			workingAfter--
		}
		if targetShift.ShiftID != nil && *targetShift.ShiftID == shiftID && targetShift.ShiftStatus == "working" {
			workingAfter--
		}

		// Add working employees after swap.
		if requesterAfterShiftID != nil && *requesterAfterShiftID == shiftID && requesterAfterStatus == "working" {
			workingAfter++
		}
		if targetAfterShiftID != nil && *targetAfterShiftID == shiftID && targetAfterStatus == "working" {
			workingAfter++
		}

		if workingAfter < 1 {
			return fmt.Errorf("cannot approve swap: shift would have no working employees after the swap")
		}
	}

	return nil
}
