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

	// Notify team leaders to review
	teamLeaders, _ := s.employeeRepo.GetByRole(ctx, "team_leader")
	for _, tl := range teamLeaders {
	if err := s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       tl.ID,
		SenderID:          &leave.EmployeeID,
		Type:              "leave_request",
		Title:             "New Leave Request",
		Message:           strPtr(fmt.Sprintf("%s %s requested %s leave from %s to %s", emp.FirstName, emp.LastName, leave.LeaveType, leave.StartDate.Format("2006-01-02"), leave.EndDate.Format("2006-01-02"))),
		RelatedEntityType: strPtr("leave"),
		RelatedEntityID:   &leave.ID,
		Priority:          "high",
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

	// Count total team leaders
	teamLeaders, _ := s.employeeRepo.GetByRole(ctx, "team_leader")
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

		// Notify managers for final approval
		managers, _ := s.employeeRepo.GetByRole(ctx, "manager")
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

// GetPendingForApproval returns leaves awaiting approval based on approver's role.
func (s *LeaveService) GetPendingForApproval(ctx context.Context, approverRole string) ([]models.Leave, error) {
	return s.leaveRepo.GetPendingForApproval(ctx, approverRole)
}

// GetShiftCoveragePreview helps managers visualize staffing levels before approving a leave.
func (s *LeaveService) GetShiftCoveragePreview(ctx context.Context, shiftID uuid.UUID, date time.Time) (*models.ShiftCoverage, error) {
	return s.scheduleRepo.GetShiftCoveragePreview(ctx, shiftID, date)
}

// GetPendingLeavesRich returns pending leaves with employee details.
func (s *LeaveService) GetPendingLeavesRich(ctx context.Context, approverRole string) ([]models.PendingLeaveRich, error) {
	return s.leaveRepo.GetPendingLeavesRich(ctx, approverRole)
}

// GetLeaveHistory returns all leaves with their approval details.
func (s *LeaveService) GetLeaveHistory(ctx context.Context) ([]models.LeaveHistoryRow, error) {
	return s.leaveRepo.GetLeaveHistory(ctx)
}

