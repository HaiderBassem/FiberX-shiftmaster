package service

import (
	"context"
	"errors"
	"fmt"
	"log"
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

type AuthService struct {
	employeeRepo    repository.EmployeeRepository
	securityService *SecurityService
	bcryptCost      int
	maxAttempts     int
	lockoutDuration time.Duration
}

func NewAuthService(
	employeeRepo repository.EmployeeRepository,
	securityService *SecurityService,
	bcryptCost int,
	maxAttempts int,
	lockoutDuration time.Duration,
) *AuthService {
	if maxAttempts < 1 {
		maxAttempts = 5
	}
	if lockoutDuration <= 0 {
		lockoutDuration = 15 * time.Minute
	}
	return &AuthService{
		employeeRepo:    employeeRepo,
		securityService: securityService,
		bcryptCost:      bcryptCost,
		maxAttempts:     maxAttempts,
		lockoutDuration: lockoutDuration,
	}
}

// dummyHash is a valid bcrypt digest of a random value. It is compared against
// when the supplied account does not exist or has no password, so that the
// response time for an unknown account resembles that of a known one. Without it
// the endpoint leaks account existence through timing alone.
var dummyHash = []byte("$2a$12$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

// Authenticate verifies email and password, returns the employee if valid.
//
// A failed password applies a *temporary* lock via employees.locked_until. It
// deliberately does not touch employees.status: that column records employment,
// and because the login identifier is an email address, letting failed attempts
// write to it would let anyone permanently disable a colleague's account.
func (s *AuthService) Authenticate(ctx context.Context, email, password, ip string) (*models.Employee, error) {
	emp, err := s.employeeRepo.GetByEmail(ctx, email)
	if err != nil {
		// Spend comparable time to a real bcrypt comparison before answering.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		_ = s.securityService.RecordFailedLogin(ctx, ip)
		return nil, ErrInvalidCredentials
	}

	if emp.Status != "active" {
		return nil, ErrAccountLocked
	}

	// A live lock short-circuits before the password is checked, so an attacker
	// cannot keep testing candidate passwords against a locked account.
	if lockedUntil, lockErr := s.employeeRepo.GetLockoutByEmail(ctx, email); lockErr == nil {
		if lockedUntil != nil && lockedUntil.After(time.Now()) {
			_ = s.securityService.RecordFailedLogin(ctx, ip)
			return nil, ErrAccountLocked
		}
	}

	if emp.PasswordHash == nil {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		_ = s.securityService.RecordFailedLogin(ctx, ip)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*emp.PasswordHash), []byte(password)); err != nil {
		lockedUntil, regErr := s.employeeRepo.RegisterFailedLogin(ctx, email, s.maxAttempts, s.lockoutDuration)
		if regErr != nil {
			log.Printf("auth: failed to record failed login for %s: %v", emp.ID, regErr)
		}

		// Record IP failure regardless, so distributed guessing across many
		// accounts is still caught by the IP blocker.
		_ = s.securityService.RecordFailedLogin(ctx, ip)

		if lockedUntil != nil && lockedUntil.After(time.Now()) {
			return nil, ErrAccountLocked
		}
		return nil, ErrInvalidCredentials
	}

	// Reset failed attempts and release any expired lock on success.
	_ = s.employeeRepo.ClearFailedLogins(ctx, emp.ID)
	s.securityService.ResetFailedLogin(ip)

	// Update last login
	_ = s.employeeRepo.UpdateLastLogin(ctx, emp.ID)
	emp.LastLogin = timePtr(time.Now())

	return emp, nil
}

// GetAuthorizedEmployee re-reads an employee and enforces that they are still
// permitted to hold a session. It is the single place that decides whether an
// already-authenticated identity remains valid, and is called on every token
// refresh so that deactivation, lockout and role changes take effect promptly
// instead of waiting for the refresh token to expire.
func (s *AuthService) GetAuthorizedEmployee(ctx context.Context, id uuid.UUID) (*models.Employee, error) {
	emp, err := s.employeeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrEmployeeNotFound
	}

	if emp.Status != "active" {
		return nil, ErrAccountLocked
	}

	lockedUntil, err := s.employeeRepo.GetLockoutByID(ctx, id)
	if err == nil && lockedUntil != nil && lockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	return emp, nil
}

// Unlock clears an authentication lock administratively.
func (s *AuthService) Unlock(ctx context.Context, id uuid.UUID) error {
	return s.employeeRepo.Unlock(ctx, id)
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
