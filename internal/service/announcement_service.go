package service

import (
	"context"
	"errors"
	"fmt"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/notification"
	"shiftmaster-backend/internal/repository"

	"github.com/google/uuid"
)

type AnnouncementService interface {
	CreateAnnouncement(ctx context.Context, announcement *models.Announcement, role string) error
}

type announcementService struct {
	announcementRepo repository.AnnouncementRepository
	employeeRepo     repository.EmployeeRepository
	emailService     *EmailService
	pushService      notification.PushService
}

func NewAnnouncementService(ar repository.AnnouncementRepository, er repository.EmployeeRepository, es *EmailService, ps notification.PushService) AnnouncementService {
	return &announcementService{
		announcementRepo: ar,
		employeeRepo:     er,
		emailService:     es,
		pushService:      ps,
	}
}

func (s *announcementService) CreateAnnouncement(ctx context.Context, a *models.Announcement, role string) error {
	canCreate := false
	if role == "admin" || role == "manager" {
		canCreate = true
	} else if a.CreatedBy != uuid.Nil {
		emp, err := s.employeeRepo.GetByID(ctx, a.CreatedBy)
		if err == nil && emp != nil && emp.CanPostAnnouncements {
			canCreate = true
		}
	}

	if !canCreate {
		return errors.New("unauthorized to post announcements")
	}

	// If the new announcement is active, deactivate others
	if a.IsActive {
		err := s.announcementRepo.SetInactiveByDepartment(ctx, a.DepartmentID)
		if err != nil {
			return fmt.Errorf("failed to deactivate old announcements: %w", err)
		}
	}

	// If the new announcement is a ticker, deactivate old tickers
	if a.IsTicker {
		err := s.announcementRepo.SetTickerInactiveByDepartment(ctx, a.DepartmentID)
		if err != nil {
			return fmt.Errorf("failed to deactivate old tickers: %w", err)
		}
	}

	err := s.announcementRepo.Create(ctx, a)
	if err != nil {
		return err
	}

	// Send email to department if active or ticker
	if (a.IsActive || a.IsTicker) && s.emailService != nil {
		emails, err := s.employeeRepo.GetEmailsByDepartment(ctx, a.DepartmentID)
		if err == nil && len(emails) > 0 {
			subject := fmt.Sprintf("[%s] %s", a.Priority, a.Title)
			body := fmt.Sprintf("A new %s announcement has been posted:\n\n%s\n\nPlease check your Dashboard for details.", a.Priority, a.Message)
			s.emailService.SendEmailAsync(emails, subject, body)
		}
	}

	// Send Web Push Notification
	if (a.IsActive || a.IsTicker) && s.pushService != nil {
		go func() {
			bgCtx := context.Background()
			_ = s.pushService.SendToDepartment(bgCtx, a.DepartmentID, "New Announcement: "+a.Title, a.Message, "/")
		}()
	}

	return nil
}
