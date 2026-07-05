package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

type ModuleAccessService struct {
	repo         repository.ModuleAccessRepository
	employeeRepo repository.EmployeeRepository
}

func NewModuleAccessService(repo repository.ModuleAccessRepository, empRepo repository.EmployeeRepository) *ModuleAccessService {
	return &ModuleAccessService{
		repo:         repo,
		employeeRepo: empRepo,
	}
}

// Link Management
func (s *ModuleAccessService) CreateLink(ctx context.Context, title, url, iconName string, createdBy *uuid.UUID) (*models.ExternalLink, error) {
	link := &models.ExternalLink{
		Title:     title,
		URL:       url,
		IconName:  iconName,
		CreatedBy: createdBy,
	}
	if err := s.repo.CreateLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *ModuleAccessService) UpdateLink(ctx context.Context, id uuid.UUID, title, url, iconName string) error {
	link := &models.ExternalLink{
		ID:       id,
		Title:    title,
		URL:      url,
		IconName: iconName,
	}
	return s.repo.UpdateLink(ctx, link)
}

func (s *ModuleAccessService) DeleteLink(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteLink(ctx, id)
}

func (s *ModuleAccessService) GetAllLinks(ctx context.Context) ([]models.ExternalLink, error) {
	return s.repo.GetAllLinks(ctx)
}

// Access Management
func (s *ModuleAccessService) GetLinkAccess(ctx context.Context, linkID uuid.UUID, departmentID *uuid.UUID) (*models.LinkAccessResponse, error) {
	link, err := s.repo.GetLinkByID(ctx, linkID)
	if err != nil {
		return nil, fmt.Errorf("link not found: %w", err)
	}

	deps, err := s.repo.GetDepartmentsWithAccess(ctx, linkID)
	if err != nil {
		return nil, err
	}

	emps, err := s.repo.GetExcludedEmployees(ctx, linkID, departmentID)
	if err != nil {
		return nil, err
	}

	return &models.LinkAccessResponse{
		LinkID:       link.ID,
		Title:        link.Title,
		Departments:  deps,
		ExcludedEmps: emps,
	}, nil
}

func (s *ModuleAccessService) SetDepartmentAccess(ctx context.Context, linkID uuid.UUID, departmentID uuid.UUID, grant bool, grantedBy *uuid.UUID) error {
	if grant {
		return s.repo.AddDepartmentAccess(ctx, linkID, departmentID, grantedBy)
	}
	return s.repo.RemoveDepartmentAccess(ctx, linkID, departmentID)
}

func (s *ModuleAccessService) SetEmployeeExclusion(ctx context.Context, linkID uuid.UUID, employeeID uuid.UUID, exclude bool, excludedBy *uuid.UUID) error {
	if exclude {
		return s.repo.AddEmployeeExclusion(ctx, linkID, employeeID, excludedBy)
	}
	return s.repo.RemoveEmployeeExclusion(ctx, linkID, employeeID)
}

func (s *ModuleAccessService) GetMyModules(ctx context.Context, employeeID uuid.UUID) ([]models.ExternalLink, error) {
	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}
	if emp.Role == "admin" {
		return s.repo.GetAllLinks(ctx)
	}
	
	if emp.DepartmentID == nil {
		return []models.ExternalLink{}, nil
	}
	return s.repo.GetEmployeeAllowedLinks(ctx, employeeID, emp.DepartmentID)
}
