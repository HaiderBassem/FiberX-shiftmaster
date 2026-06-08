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

func (s *ModuleAccessService) GetModuleAccess(ctx context.Context, moduleName string, departmentID *uuid.UUID) (*models.ModuleAccessResponse, error) {
	deps, err := s.repo.GetDepartmentsWithAccess(ctx, moduleName)
	if err != nil {
		return nil, err
	}

	emps, err := s.repo.GetExcludedEmployees(ctx, moduleName, departmentID)
	if err != nil {
		return nil, err
	}

	return &models.ModuleAccessResponse{
		ModuleName:   moduleName,
		Departments:  deps,
		ExcludedEmps: emps,
	}, nil
}

func (s *ModuleAccessService) SetDepartmentAccess(ctx context.Context, moduleName string, departmentID uuid.UUID, grant bool, grantedBy *uuid.UUID) error {
	if grant {
		return s.repo.AddDepartmentAccess(ctx, moduleName, departmentID, grantedBy)
	}
	return s.repo.RemoveDepartmentAccess(ctx, moduleName, departmentID)
}

func (s *ModuleAccessService) SetEmployeeExclusion(ctx context.Context, moduleName string, employeeID uuid.UUID, exclude bool, excludedBy *uuid.UUID) error {
	if exclude {
		return s.repo.AddEmployeeExclusion(ctx, moduleName, employeeID, excludedBy)
	}
	return s.repo.RemoveEmployeeExclusion(ctx, moduleName, employeeID)
}

func (s *ModuleAccessService) GetMyModules(ctx context.Context, employeeID uuid.UUID) ([]string, error) {
	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("employee not found: %w", err)
	}
	if emp.DepartmentID == nil {
		return []string{}, nil
	}
	return s.repo.GetEmployeeAllowedModules(ctx, employeeID, emp.DepartmentID)
}
