// Package testutil provides in-memory fakes shared by tests across packages.
//
// It is deliberately not guarded by a build tag: Go excludes it from production
// binaries automatically, because nothing outside a _test.go file imports it.
package testutil

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// FakeEmployeeRepo implements just enough of repository.EmployeeRepository to
// exercise authentication. The embedded interface is nil, so any method a test
// does not expect to be called panics rather than silently returning a zero value.
type FakeEmployeeRepo struct {
	repository.EmployeeRepository

	mu       sync.Mutex
	byEmail  map[string]*models.Employee
	attempts map[string]int
	locked   map[string]*time.Time

	// StatusWrites records every UpdateStatus call, so tests can assert that
	// failed logins never touch employment state.
	statusWrites []string
}

// NewFakeEmployeeRepo returns a repository seeded with the given employees.
func NewFakeEmployeeRepo(employees ...*models.Employee) *FakeEmployeeRepo {
	repo := &FakeEmployeeRepo{
		byEmail:  make(map[string]*models.Employee, len(employees)),
		attempts: map[string]int{},
		locked:   map[string]*time.Time{},
	}
	for _, emp := range employees {
		repo.byEmail[emp.Email] = emp
	}
	return repo
}

// TestPassword is the plaintext password on employees built by NewEmployee.
const TestPassword = "correct-horse-battery-staple"

// NewEmployee builds an active employee whose password is TestPassword.
func NewEmployee(email, role string) *models.Employee {
	// MinCost keeps suites fast; the cost factor is irrelevant to auth logic.
	hash, err := bcrypt.GenerateFromPassword([]byte(TestPassword), bcrypt.MinCost)
	if err != nil {
		panic("testutil: hash fixture password: " + err.Error())
	}
	hashStr := string(hash)

	return &models.Employee{
		ID:           uuid.New(),
		Email:        email,
		Role:         role,
		Status:       "active",
		PasswordHash: &hashStr,
	}
}

func (f *FakeEmployeeRepo) GetByEmail(_ context.Context, email string) (*models.Employee, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	emp, ok := f.byEmail[email]
	if !ok {
		return nil, errors.New("employee not found")
	}
	copied := *emp
	return &copied, nil
}

func (f *FakeEmployeeRepo) GetByID(_ context.Context, id uuid.UUID) (*models.Employee, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, emp := range f.byEmail {
		if emp.ID == id {
			copied := *emp
			return &copied, nil
		}
	}
	return nil, errors.New("employee not found")
}

// RegisterFailedLogin mirrors the SQL in employee_repository.go: an expired lock
// resets the counter to 1, and reaching the threshold applies a fresh lock.
func (f *FakeEmployeeRepo) RegisterFailedLogin(_ context.Context, email string, maxAttempts int, d time.Duration) (*time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.byEmail[email]; !ok {
		return nil, nil
	}

	if existing := f.locked[email]; existing != nil && !existing.After(time.Now()) {
		f.attempts[email] = 1
		f.locked[email] = nil
	} else {
		f.attempts[email]++
	}

	if f.attempts[email] >= maxAttempts {
		until := time.Now().Add(d)
		f.locked[email] = &until
		return &until, nil
	}
	return f.locked[email], nil
}

func (f *FakeEmployeeRepo) ClearFailedLogins(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for email, emp := range f.byEmail {
		if emp.ID == id {
			f.attempts[email] = 0
			f.locked[email] = nil
		}
	}
	return nil
}

func (f *FakeEmployeeRepo) Unlock(ctx context.Context, id uuid.UUID) error {
	return f.ClearFailedLogins(ctx, id)
}

func (f *FakeEmployeeRepo) GetLockoutByEmail(_ context.Context, email string) (*time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.locked[email], nil
}

func (f *FakeEmployeeRepo) GetLockoutByID(_ context.Context, id uuid.UUID) (*time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for email, emp := range f.byEmail {
		if emp.ID == id {
			return f.locked[email], nil
		}
	}
	return nil, nil
}

func (f *FakeEmployeeRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statusWrites = append(f.statusWrites, status)
	for _, emp := range f.byEmail {
		if emp.ID == id {
			emp.Status = status
		}
	}
	return nil
}

func (f *FakeEmployeeRepo) UpdateLastLogin(_ context.Context, _ uuid.UUID) error { return nil }

// --- test helpers -----------------------------------------------------------

// ForceLock sets an explicit lock expiry, letting a test simulate the passage of
// time without sleeping.
func (f *FakeEmployeeRepo) ForceLock(email string, until time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.locked[email] = &until
}

// StatusWriteCount reports how many times UpdateStatus was called.
func (f *FakeEmployeeRepo) StatusWriteCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.statusWrites)
}

// SetRole changes an employee's stored role, simulating an administrator edit
// between one request and the next.
func (f *FakeEmployeeRepo) SetRole(email, role string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if emp, ok := f.byEmail[email]; ok {
		emp.Role = role
	}
}

// SetDepartment changes an employee's stored department.
func (f *FakeEmployeeRepo) SetDepartment(email string, deptID *uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if emp, ok := f.byEmail[email]; ok {
		emp.DepartmentID = deptID
	}
}

// Status reports an employee's current employment status.
func (f *FakeEmployeeRepo) Status(email string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if emp, ok := f.byEmail[email]; ok {
		return emp.Status
	}
	return ""
}
