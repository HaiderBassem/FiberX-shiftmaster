package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// EmployeeService handles employee business logic.
type EmployeeService struct {
	employeeRepo   repository.EmployeeRepository
	departmentRepo repository.DepartmentRepository
	authService    *AuthService
}

func NewEmployeeService(
	employeeRepo repository.EmployeeRepository,
	departmentRepo repository.DepartmentRepository,
	authService *AuthService,
) *EmployeeService {
	return &EmployeeService{
		employeeRepo:   employeeRepo,
		departmentRepo: departmentRepo,
		authService:    authService,
	}
}

func (s *EmployeeService) GetByID(ctx context.Context, id uuid.UUID) (*models.Employee, error) {
	return s.employeeRepo.GetByID(ctx, id)
}

func (s *EmployeeService) GetAll(ctx context.Context) ([]models.Employee, error) {
	return s.employeeRepo.GetAll(ctx)
}

func (s *EmployeeService) GetActive(ctx context.Context) ([]models.Employee, error) {
	return s.employeeRepo.GetActive(ctx)
}

func (s *EmployeeService) GetByDepartment(ctx context.Context, deptID uuid.UUID) ([]models.Employee, error) {
	return s.employeeRepo.GetByDepartment(ctx, deptID)
}

func (s *EmployeeService) GetByRole(ctx context.Context, role string) ([]models.Employee, error) {
	return s.employeeRepo.GetByRole(ctx, role)
}

func (s *EmployeeService) GetByShiftID(ctx context.Context, shiftID uuid.UUID) ([]models.Employee, error) {
	return s.employeeRepo.GetByShiftID(ctx, shiftID)
}

// CreateEmployee creates a new employee with a hashed password.
func (s *EmployeeService) CreateEmployee(ctx context.Context, emp *models.Employee, password string) error {
	// Validate department exists if provided
	if emp.DepartmentID != nil {
		if _, err := s.departmentRepo.GetByID(ctx, *emp.DepartmentID); err != nil {
			return fmt.Errorf("invalid department: %w", err)
		}
	}

	// If creator account was deleted, avoid FK failure on employees.created_by.
	if emp.CreatedBy != nil {
		if _, err := s.employeeRepo.GetByID(ctx, *emp.CreatedBy); err != nil {
			emp.CreatedBy = nil
		}
	}

	// Hash password
	hash, err := s.authService.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	emp.PasswordHash = &hash

	// Set defaults
	if strings.TrimSpace(emp.EmployeeCode) == "" {
		code, err := s.generateEmployeeCode(ctx, emp.Role)
		if err != nil {
			return err
		}
		emp.EmployeeCode = code
	}
	if emp.Status == "" {
		emp.Status = "active"
	}

	return s.employeeRepo.Create(ctx, emp)
}

func (s *EmployeeService) generateEmployeeCode(ctx context.Context, role string) (string, error) {
	prefix := "EMP"
	switch role {
	case "manager":
		prefix = "MGR"
	case "team_leader":
		prefix = "TL"
	case "admin":
		prefix = "ADM"
	}

	for i := 0; i < 8; i++ {
		candidate := fmt.Sprintf("%s-%06d", prefix, time.Now().UnixNano()%1000000)
		if _, err := s.employeeRepo.GetByCode(ctx, candidate); err != nil {
			return candidate, nil
		}
		time.Sleep(2 * time.Millisecond)
	}
	return "", fmt.Errorf("failed to generate unique employee code")
}

func (s *EmployeeService) UpdateEmployee(ctx context.Context, emp *models.Employee) error {
	// Validate employee exists
	if _, err := s.employeeRepo.GetByID(ctx, emp.ID); err != nil {
		return ErrEmployeeNotFound
	}
	return s.employeeRepo.Update(ctx, emp)
}

func (s *EmployeeService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	validStatuses := map[string]bool{"active": true, "inactive": true, "on_leave": true, "terminated": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}
	return s.employeeRepo.UpdateStatus(ctx, id, status)
}

func (s *EmployeeService) ChangePassword(ctx context.Context, id uuid.UUID, oldPassword, newPassword string, isAdmin bool) error {
	// If not admin, verify old password
	if !isAdmin {
		emp, err := s.employeeRepo.GetByID(ctx, id)
		if err != nil {
			return ErrEmployeeNotFound
		}
		if emp.PasswordHash == nil || *emp.PasswordHash == "" {
			return fmt.Errorf("no existing password, contact admin")
		}
		
		err = bcrypt.CompareHashAndPassword([]byte(*emp.PasswordHash), []byte(oldPassword))
		if err != nil {
			return fmt.Errorf("incorrect old password")
		}
	}

	// Hash new password
	hash, err := s.authService.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	return s.employeeRepo.UpdatePassword(ctx, id, hash)
}

func (s *EmployeeService) DeleteEmployee(ctx context.Context, id uuid.UUID) error {
	return s.employeeRepo.ForceDelete(ctx, id)
}
