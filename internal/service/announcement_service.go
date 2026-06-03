package service

import (
	"context"
	"fmt"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

type AnnouncementService interface {
	CreateAnnouncement(ctx context.Context, announcement *models.Announcement) error
}

type announcementService struct {
	announcementRepo repository.AnnouncementRepository
	employeeRepo     repository.EmployeeRepository
	emailService     *EmailService
}

func NewAnnouncementService(ar repository.AnnouncementRepository, er repository.EmployeeRepository, es *EmailService) AnnouncementService {
	return &announcementService{
		announcementRepo: ar,
		employeeRepo:     er,
		emailService:     es,
	}
}

func (s *announcementService) CreateAnnouncement(ctx context.Context, a *models.Announcement) error {
	// If the new announcement is active, deactivate others
	if a.IsActive {
		err := s.announcementRepo.SetInactiveByDepartment(ctx, a.DepartmentID)
		if err != nil {
			return fmt.Errorf("failed to deactivate old announcements: %w", err)
		}
	}

	err := s.announcementRepo.Create(ctx, a)
	if err != nil {
		return err
	}

	// Send email to department if active
	if a.IsActive && s.emailService != nil {
		emails, err := s.employeeRepo.GetEmailsByDepartment(ctx, a.DepartmentID)
		if err == nil && len(emails) > 0 {
			subject := fmt.Sprintf("[%s] %s", a.Priority, a.Title)
			body := fmt.Sprintf("A new %s announcement has been posted:\n\n%s\n\nPlease check your Dashboard for details.", a.Priority, a.Message)
			s.emailService.SendEmailAsync(emails, subject, body)
		}
	}

	return nil
}
