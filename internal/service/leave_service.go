package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// LeaveService handles leave request business logic with approval chain.
type LeaveService struct {
	leaveRepo    repository.LeaveRepository
	employeeRepo repository.EmployeeRepository
	scheduleRepo repository.ScheduleRepository
	notifService *NotificationService
}

func NewLeaveService(
	leaveRepo repository.LeaveRepository,
	employeeRepo repository.EmployeeRepository,
	scheduleRepo repository.ScheduleRepository,
	notifService *NotificationService,
) *LeaveService {
	return &LeaveService{
		leaveRepo:    leaveRepo,
		employeeRepo: employeeRepo,
		scheduleRepo: scheduleRepo,
		notifService: notifService,
	}
}

// RequestLeave creates a new leave request and notifies team leaders.
func (s *LeaveService) RequestLeave(ctx context.Context, leave *models.Leave) error {
	// Validate dates
	if leave.EndDate.Before(leave.StartDate) {
		return fmt.Errorf("end date cannot be before start date")
	}
	if leave.StartDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return fmt.Errorf("cannot request leave for past dates")
	}

	// Validate employee exists
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

	// Notify team leaders of the SAME department to review
	var teamLeaders []models.Employee
	if emp.DepartmentID != nil {
		deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *emp.DepartmentID)
		for _, e := range deptEmps {
			if e.Role == "team_leader" {
				teamLeaders = append(teamLeaders, e)
			}
		}
	}

	actionUrl := "/approvals"
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
	}

	return nil
}

// ApproveByTeamLeader records one TL's approval. Only when ALL TLs approve
// does the leave move to "approved_by_team_leader" status.
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

	// Record the individual approval
	if err := s.leaveRepo.RecordApproval(ctx, leaveID, teamLeaderID, "team_leader", "approved", nil); err != nil {
		return err
	}

	// Count how many TLs have now approved
	approvedCount, _ := s.leaveRepo.CountTLApprovals(ctx, leaveID)

	// Count total team leaders in the employee's department
	var teamLeaders []models.Employee
	// Removed unused deptErr
	if leave.EmployeeID != uuid.Nil {
		leaveEmp, _ := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
		if leaveEmp != nil && leaveEmp.DepartmentID != nil {
			deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *leaveEmp.DepartmentID)
			for _, e := range deptEmps {
				if e.Role == "team_leader" {
					teamLeaders = append(teamLeaders, e)
				}
			}
		} else {
			teamLeaders, _ = s.employeeRepo.GetByRole(ctx, "team_leader")
		}
	}
	totalTLs := len(teamLeaders)

	// Get the TL's name for the notification
	tlEmployee, _ := s.employeeRepo.GetByID(ctx, teamLeaderID)
	tlName := "A team leader"
	if tlEmployee != nil {
		tlName = tlEmployee.FirstName + " " + tlEmployee.LastName
	}

	if totalTLs > 0 && approvedCount >= totalTLs {
		// ALL team leaders approved → move to next stage
		if err := s.leaveRepo.UpdateStatus(ctx, leaveID, "approved_by_team_leader", teamLeaderID, "team_leader"); err != nil {
			return err
		}

		// Notify managers of the SAME department for final approval
		var managers []models.Employee
		if leave.EmployeeID != uuid.Nil {
			leaveEmp, _ := s.employeeRepo.GetByID(ctx, leave.EmployeeID)
			if leaveEmp != nil && leaveEmp.DepartmentID != nil {
				deptEmps, _ := s.employeeRepo.GetByDepartment(ctx, *leaveEmp.DepartmentID)
				for _, e := range deptEmps {
					if e.Role == "manager" {
						managers = append(managers, e)
					}
				}
			}
		}

		for _, mgr := range managers {
			if err := s.notifService.SendNotification(ctx, &models.Notification{
				RecipientID:       mgr.ID,
				SenderID:          &teamLeaderID,
				Type:              "approval",
				Title:             "Leave Request Awaiting Approval",
				Message:           strPtr("A leave request has been approved by ALL team leaders and needs your final approval"),
				RelatedEntityType: strPtr("leave"),
				RelatedEntityID:   &leaveID,
				Priority:          "high",
			}); err != nil {
				fmt.Printf("Failed to send manager notification: %v\n", err)
			}
		}

		// Notify employee — all TLs approved
		if err := s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       leave.EmployeeID,
			SenderID:          &teamLeaderID,
			Type:              "approval",
			Title:             "Leave Approved by All Team Leaders",
			Message:           strPtr(fmt.Sprintf("%s has approved your leave request. All team leaders have now approved — awaiting manager approval.", tlName)),
			RelatedEntityType: strPtr("leave"),
			RelatedEntityID:   &leaveID,
			Priority:          "medium",
		}); err != nil {
			fmt.Printf("Failed to send employee notification (all TLs): %v\n", err)
		}
	} else {
		// Notify employee — partial approval with TL name
		if err := s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       leave.EmployeeID,
			SenderID:          &teamLeaderID,
			Type:              "approval",
			Title:             fmt.Sprintf("Leave Approved by %s", tlName),
			Message:           strPtr(fmt.Sprintf("%s has approved your leave request (%d of %d team leaders approved).", tlName, approvedCount, totalTLs)),
			RelatedEntityType: strPtr("leave"),
			RelatedEntityID:   &leaveID,
			Priority:          "low",
		}); err != nil {
			fmt.Printf("Failed to send employee notification (partial TL): %v\n", err)
		}
	}

	return nil
}

// ApproveByManager gives final approval to a leave request.
func (s *LeaveService) ApproveByManager(ctx context.Context, leaveID uuid.UUID, managerID uuid.UUID) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}
	if leave.Status != "approved_by_team_leader" {
		return fmt.Errorf("leave must be approved by all team leaders first, current status: %s", leave.Status)
	}

	// Record the manager approval
	if err := s.leaveRepo.RecordApproval(ctx, leaveID, managerID, "manager", "approved", nil); err != nil {
		return err
	}

	if err := s.leaveRepo.UpdateStatus(ctx, leaveID, "approved_by_manager", managerID, "manager"); err != nil {
		return err
	}

	// ── Apply leave to employee_shifts ──────────────────────────────────────
	// For each calendar day in the leave range, upsert the employee's shift row
	// to shift_status='leave'. This makes the daily schedule display correctly.
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
			CreatedBy:   &managerID,
		}
		if upsertErr := s.scheduleRepo.UpsertEmployeeShift(ctx, es); upsertErr != nil {
			fmt.Printf("[LEAVE] Failed to upsert employee shift for %s on %s: %v\n", leave.EmployeeID, d.Format("2006-01-02"), upsertErr)
		}
	}
	// ────────────────────────────────────────────────────────────────────────

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

// GetLeaveHistory returns all leaves with their approval details.
func (s *LeaveService) GetLeaveHistory(ctx context.Context) ([]models.LeaveHistoryRow, error) {
	return s.leaveRepo.GetLeaveHistory(ctx)
}

