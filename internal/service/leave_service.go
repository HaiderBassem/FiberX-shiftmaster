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
		_ = s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       tl.ID,
			SenderID:          &leave.EmployeeID,
			Type:              "leave_request",
			Title:             "New Leave Request",
			Message:           strPtr(fmt.Sprintf("%s %s requested %s leave from %s to %s", emp.FirstName, emp.LastName, leave.LeaveType, leave.StartDate.Format("2006-01-02"), leave.EndDate.Format("2006-01-02"))),
			RelatedEntityType: strPtr("leave"),
			RelatedEntityID:   &leave.ID,
			Priority:          "high",
		})
	}

	return nil
}

// ApproveByTeamLeader approves a leave request at the team leader level.
// After approval, the request moves to pending manager approval.
func (s *LeaveService) ApproveByTeamLeader(ctx context.Context, leaveID uuid.UUID, teamLeaderID uuid.UUID) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}
	if leave.Status != "pending" {
		return fmt.Errorf("leave is not pending, current status: %s", leave.Status)
	}

	if err := s.leaveRepo.UpdateStatus(ctx, leaveID, "approved_by_team_leader", teamLeaderID, "team_leader"); err != nil {
		return err
	}

	// Notify managers for final approval
	managers, _ := s.employeeRepo.GetByRole(ctx, "manager")
	for _, mgr := range managers {
		_ = s.notifService.SendNotification(ctx, &models.Notification{
			RecipientID:       mgr.ID,
			SenderID:          &teamLeaderID,
			Type:              "approval",
			Title:             "Leave Request Awaiting Approval",
			Message:           strPtr("A leave request has been approved by team leader and needs your final approval"),
			RelatedEntityType: strPtr("leave"),
			RelatedEntityID:   &leaveID,
			Priority:          "high",
		})
	}

	// Notify employee
	_ = s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       leave.EmployeeID,
		SenderID:          &teamLeaderID,
		Type:              "approval",
		Title:             "Leave Approved by Team Leader",
		Message:           strPtr("Your leave request has been approved by the team leader. Awaiting manager approval."),
		RelatedEntityType: strPtr("leave"),
		RelatedEntityID:   &leaveID,
		Priority:          "medium",
	})

	return nil
}

// ApproveByManager gives final approval to a leave request.
func (s *LeaveService) ApproveByManager(ctx context.Context, leaveID uuid.UUID, managerID uuid.UUID) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}
	if leave.Status != "approved_by_team_leader" {
		return fmt.Errorf("leave must be approved by team leader first, current status: %s", leave.Status)
	}

	if err := s.leaveRepo.UpdateStatus(ctx, leaveID, "approved_by_manager", managerID, "manager"); err != nil {
		return err
	}

	// Notify employee about final approval
	_ = s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       leave.EmployeeID,
		SenderID:          &managerID,
		Type:              "approval",
		Title:             "Leave Approved",
		Message:           strPtr("Your leave request has been fully approved!"),
		RelatedEntityType: strPtr("leave"),
		RelatedEntityID:   &leaveID,
		Priority:          "high",
	})

	return nil
}

// RejectLeave rejects a leave request with a reason.
func (s *LeaveService) RejectLeave(ctx context.Context, leaveID uuid.UUID, rejectedBy uuid.UUID, reason string) error {
	leave, err := s.leaveRepo.GetByID(ctx, leaveID)
	if err != nil {
		return fmt.Errorf("leave not found: %w", err)
	}

	if err := s.leaveRepo.Reject(ctx, leaveID, rejectedBy, reason); err != nil {
		return err
	}

	// Notify employee
	_ = s.notifService.SendNotification(ctx, &models.Notification{
		RecipientID:       leave.EmployeeID,
		SenderID:          &rejectedBy,
		Type:              "approval",
		Title:             "Leave Request Rejected",
		Message:           strPtr(fmt.Sprintf("Your leave request has been rejected. Reason: %s", reason)),
		RelatedEntityType: strPtr("leave"),
		RelatedEntityID:   &leaveID,
		Priority:          "high",
	})

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
