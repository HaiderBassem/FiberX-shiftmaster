package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountLocked      = errors.New("account is locked or inactive")
	ErrEmployeeNotFound   = errors.New("employee not found")
)

// AuthService handles authentication and password management.
type AuthService struct {
	employeeRepo repository.EmployeeRepository
	bcryptCost   int
}

func NewAuthService(employeeRepo repository.EmployeeRepository, bcryptCost int) *AuthService {
	return &AuthService{
		employeeRepo: employeeRepo,
		bcryptCost:   bcryptCost,
	}
}

// Authenticate verifies email and password, returns the employee if valid.
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*models.Employee, error) {
	emp, err := s.employeeRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if emp.Status != "active" {
		return nil, ErrAccountLocked
	}

	if emp.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*emp.PasswordHash), []byte(password)); err != nil {
		attempts, _ := s.employeeRepo.IncrementFailedLogin(ctx, email)
		if attempts >= 10 {
			_ = s.employeeRepo.UpdateStatus(ctx, emp.ID, "inactive")
			return nil, ErrAccountLocked
		}
		return nil, ErrInvalidCredentials
	}

	// Reset failed attempts on success
	_ = s.employeeRepo.ResetFailedLogin(ctx, emp.ID)

	// Update last login
	_ = s.employeeRepo.UpdateLastLogin(ctx, emp.ID)
	emp.LastLogin = timePtr(time.Now())

	return emp, nil
}

// HashPassword creates a bcrypt hash from a plain text password.
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// ChangePassword updates an employee's password.
func (s *AuthService) ChangePassword(ctx context.Context, employeeID uuid.UUID, oldPassword, newPassword string) error {
	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return ErrEmployeeNotFound
	}

	if emp.PasswordHash == nil {
		return ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*emp.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.employeeRepo.UpdatePassword(ctx, employeeID, hash)
}

// ResetPassword sets a new password without requiring the old one (admin only).
func (s *AuthService) ResetPassword(ctx context.Context, employeeID uuid.UUID, newPassword string) error {
	hash, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.employeeRepo.UpdatePassword(ctx, employeeID, hash)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
