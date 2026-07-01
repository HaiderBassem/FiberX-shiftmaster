package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

type ItemRequestService struct {
	repo         repository.ItemRequestRepository
	empRepo      repository.EmployeeRepository
	deptRepo     repository.DepartmentRepository
	emailService *EmailService
}

func NewItemRequestService(repo repository.ItemRequestRepository, empRepo repository.EmployeeRepository, deptRepo repository.DepartmentRepository, emailService *EmailService) *ItemRequestService {
	return &ItemRequestService{
		repo:         repo,
		empRepo:      empRepo,
		deptRepo:     deptRepo,
		emailService: emailService,
	}
}

// Categories

func (s *ItemRequestService) CreateCategory(ctx context.Context, cat *models.ItemRequestCategory) error {
	return s.repo.CreateCategory(ctx, cat)
}

func (s *ItemRequestService) GetCategoriesByDepartment(ctx context.Context, departmentID uuid.UUID) ([]models.ItemRequestCategory, error) {
	return s.repo.GetCategoriesByDepartment(ctx, departmentID)
}

func (s *ItemRequestService) UpdateCategory(ctx context.Context, cat *models.ItemRequestCategory) error {
	return s.repo.UpdateCategory(ctx, cat)
}

func (s *ItemRequestService) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCategory(ctx, id)
}

// Requests

func (s *ItemRequestService) SubmitRequest(ctx context.Context, employeeID, categoryID uuid.UUID, description string) (*models.ItemRequest, error) {
	// 1. Fetch category
	cat, err := s.repo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("category not found: %w", err)
	}

	// 2. Fetch employee details
	emp, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}

	deptName := "Unknown Department"
	if emp.DepartmentID != nil {
		dept, err := s.deptRepo.GetByID(ctx, *emp.DepartmentID)
		if err == nil && dept != nil {
			deptName = dept.Name
		}
	}

	// 3. Create request
	req := &models.ItemRequest{
		EmployeeID:  employeeID,
		CategoryID:  categoryID,
		Description: description,
		Status:      "pending",
	}

	if err := s.repo.CreateRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 4. Send email
	if s.emailService != nil {
		toEmails := parseEmails(cat.ToEmails)
		ccEmails := []string{}
		if cat.CCEmails != nil && *cat.CCEmails != "" {
			ccEmails = parseEmails(*cat.CCEmails)
		}

		subject := fmt.Sprintf("New Item Request: %s - %s %s", cat.Name, emp.FirstName, emp.LastName)
		
		body := fmt.Sprintf("Employee: %s %s (%s)\nDepartment: %s\n\nRequested Category: %s\n\nDescription:\n%s",
			emp.FirstName, emp.LastName, emp.EmployeeCode,
			deptName,
			cat.Name,
			description,
		)

		s.emailService.SendEmailWithCCAsync(toEmails, ccEmails, subject, body)
	}

	return req, nil
}

func (s *ItemRequestService) GetRequestsByEmployee(ctx context.Context, employeeID uuid.UUID) ([]models.ItemRequest, error) {
	return s.repo.GetRequestsByEmployee(ctx, employeeID)
}

// parseEmails splits a comma-separated string into a slice of trimmed emails
func parseEmails(input string) []string {
	parts := strings.Split(input, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (s *ItemRequestService) GetPendingRequests(ctx context.Context, departmentID uuid.UUID) ([]models.ItemRequest, error) {
	return s.repo.GetPendingRequests(ctx, departmentID)
}

func (s *ItemRequestService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if status != "processed" && status != "rejected" { return fmt.Errorf("invalid status") }
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *ItemRequestService) CancelRequest(ctx context.Context, id uuid.UUID, employeeID uuid.UUID) error {
	return s.repo.Cancel(ctx, id, employeeID)
}

