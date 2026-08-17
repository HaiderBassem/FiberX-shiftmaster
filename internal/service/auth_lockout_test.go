package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/testutil"
)

const testPassword = testutil.TestPassword

func newTestEmployee(t *testing.T) *models.Employee {
	t.Helper()
	return testutil.NewEmployee("worker@fiberx.iq", "employee")
}

func newTestAuthService(repo repository.EmployeeRepository, maxAttempts int, lockout time.Duration) *AuthService {
	security := NewSecurityService(nil, 1_000_000, time.Hour)
	return NewAuthService(repo, security, bcrypt.MinCost, maxAttempts, lockout)
}

// --- tests ------------------------------------------------------------------

func TestFailedAttemptsBelowThresholdDoNotLock(t *testing.T) {
	emp := newTestEmployee(t)
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 5, 15*time.Minute)
	ctx := context.Background()

	for i := 1; i < 5; i++ {
		_, err := svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: error = %v, want ErrInvalidCredentials", i, err)
		}
	}

	// The correct password must still work right up to the threshold.
	if _, err := svc.Authenticate(ctx, emp.Email, testPassword, "10.0.0.1"); err != nil {
		t.Fatalf("correct password rejected below the lockout threshold: %v", err)
	}
}

func TestThresholdAppliesTemporaryLock(t *testing.T) {
	emp := newTestEmployee(t)
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 3, 15*time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1")
	}

	// Once locked, even the correct password is refused.
	_, err := svc.Authenticate(ctx, emp.Email, testPassword, "10.0.0.1")
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("error = %v, want ErrAccountLocked", err)
	}
}

// The central regression for S-07: a lockout must never be written to
// employees.status, because the login identifier is an email address and that
// would let anyone permanently disable a colleague's account.
func TestLockoutNeverMutatesEmploymentStatus(t *testing.T) {
	emp := newTestEmployee(t)
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 3, 15*time.Minute)
	ctx := context.Background()

	// Far more failures than the threshold, as an attacker would send.
	for i := 0; i < 50; i++ {
		_, _ = svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1")
	}

	if n := repo.StatusWriteCount(); n != 0 {
		t.Fatalf("employment status was written %d time(s) by failed logins", n)
	}

	stored, err := repo.GetByEmail(ctx, emp.Email)
	if err != nil {
		t.Fatalf("fetch employee: %v", err)
	}
	if stored.Status != "active" {
		t.Fatalf("status = %q, want %q: a failed login changed employment state", stored.Status, "active")
	}
}

// The lock is self-healing: no administrator is needed once it expires.
func TestLockExpiresAutomatically(t *testing.T) {
	emp := newTestEmployee(t)
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 3, 15*time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1")
	}
	if _, err := svc.Authenticate(ctx, emp.Email, testPassword, "10.0.0.1"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected the account to be locked, got %v", err)
	}

	// Move the expiry into the past, as the passage of time would.
	repo.ForceLock(emp.Email, time.Now().Add(-time.Second))

	if _, err := svc.Authenticate(ctx, emp.Email, testPassword, "10.0.0.1"); err != nil {
		t.Fatalf("account still locked after the lock expired: %v", err)
	}
}

func TestSuccessfulLoginClearsFailureState(t *testing.T) {
	emp := newTestEmployee(t)
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 5, 15*time.Minute)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1")
	}
	if _, err := svc.Authenticate(ctx, emp.Email, testPassword, "10.0.0.1"); err != nil {
		t.Fatalf("correct password rejected: %v", err)
	}

	// The counter is back to zero, so a fresh run of failures is needed to lock.
	for i := 0; i < 4; i++ {
		if _, err := svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d after reset: error = %v, want ErrInvalidCredentials", i+1, err)
		}
	}
}

func TestMaxLoginAttemptsIsConfigurable(t *testing.T) {
	for _, threshold := range []int{2, 3, 7} {
		t.Run(string(rune('0'+threshold)), func(t *testing.T) {
			emp := newTestEmployee(t)
			repo := testutil.NewFakeEmployeeRepo(emp)
			svc := newTestAuthService(repo, threshold, 15*time.Minute)
			ctx := context.Background()

			for i := 1; i < threshold; i++ {
				if _, err := svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1"); !errors.Is(err, ErrInvalidCredentials) {
					t.Fatalf("locked early at attempt %d of %d", i, threshold)
				}
			}

			if _, err := svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1"); !errors.Is(err, ErrAccountLocked) {
				t.Fatalf("not locked at the configured threshold of %d: %v", threshold, err)
			}
		})
	}
}

// Concurrent failed logins must not race past the threshold.
func TestConcurrentFailedLoginsStillLock(t *testing.T) {
	emp := newTestEmployee(t)
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 5, 15*time.Minute)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1")
		}()
	}
	wg.Wait()

	if _, err := svc.Authenticate(ctx, emp.Email, testPassword, "10.0.0.1"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("account not locked after 20 concurrent failures: %v", err)
	}
	if n := repo.StatusWriteCount(); n != 0 {
		t.Errorf("employment status written %d time(s) under concurrency", n)
	}
}

// --- refresh revalidation (S-05) -------------------------------------------

func TestGetAuthorizedEmployeeEnforcesCurrentState(t *testing.T) {
	ctx := context.Background()

	t.Run("active employee is authorized", func(t *testing.T) {
		emp := newTestEmployee(t)
		repo := testutil.NewFakeEmployeeRepo(emp)
		svc := newTestAuthService(repo, 5, 15*time.Minute)

		got, err := svc.GetAuthorizedEmployee(ctx, emp.ID)
		if err != nil {
			t.Fatalf("active employee rejected: %v", err)
		}
		if got.ID != emp.ID {
			t.Errorf("returned the wrong employee")
		}
	})

	t.Run("deactivated employee is rejected", func(t *testing.T) {
		emp := newTestEmployee(t)
		repo := testutil.NewFakeEmployeeRepo(emp)
		svc := newTestAuthService(repo, 5, 15*time.Minute)

		_ = repo.UpdateStatus(ctx, emp.ID, "inactive")

		if _, err := svc.GetAuthorizedEmployee(ctx, emp.ID); !errors.Is(err, ErrAccountLocked) {
			t.Fatalf("a deactivated employee could still refresh: %v", err)
		}
	})

	t.Run("locked employee is rejected", func(t *testing.T) {
		emp := newTestEmployee(t)
		repo := testutil.NewFakeEmployeeRepo(emp)
		svc := newTestAuthService(repo, 5, 15*time.Minute)

		repo.ForceLock(emp.Email, time.Now().Add(10*time.Minute))

		if _, err := svc.GetAuthorizedEmployee(ctx, emp.ID); !errors.Is(err, ErrAccountLocked) {
			t.Fatalf("a locked employee could still refresh: %v", err)
		}
	})

	t.Run("unknown employee is rejected", func(t *testing.T) {
		emp := newTestEmployee(t)
		repo := testutil.NewFakeEmployeeRepo(emp)
		svc := newTestAuthService(repo, 5, 15*time.Minute)

		if _, err := svc.GetAuthorizedEmployee(ctx, uuid.New()); !errors.Is(err, ErrEmployeeNotFound) {
			t.Fatalf("an unknown employee id was authorized: %v", err)
		}
	})

	// A role change must be visible immediately, because the refresh handler signs
	// the new token from whatever this returns.
	t.Run("current role is returned after a demotion", func(t *testing.T) {
		emp := newTestEmployee(t)
		emp.Role = "admin"
		repo := testutil.NewFakeEmployeeRepo(emp)
		svc := newTestAuthService(repo, 5, 15*time.Minute)

		repo.SetRole(emp.Email, "employee")

		got, err := svc.GetAuthorizedEmployee(ctx, emp.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Role != "employee" {
			t.Errorf("role = %q, want %q: a demotion did not take effect", got.Role, "employee")
		}
	})
}

func TestAuthenticateRejectsInactiveEmployee(t *testing.T) {
	emp := newTestEmployee(t)
	emp.Status = "inactive"
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 5, 15*time.Minute)

	if _, err := svc.Authenticate(context.Background(), emp.Email, testPassword, "10.0.0.1"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("an inactive employee authenticated: %v", err)
	}
}

func TestUnknownAccountYieldsGenericError(t *testing.T) {
	emp := newTestEmployee(t)
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 5, 15*time.Minute)

	_, err := svc.Authenticate(context.Background(), "nobody@fiberx.iq", "whatever", "10.0.0.1")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials for an unknown account", err)
	}
}

func TestAdministrativeUnlock(t *testing.T) {
	emp := newTestEmployee(t)
	repo := testutil.NewFakeEmployeeRepo(emp)
	svc := newTestAuthService(repo, 3, time.Hour)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = svc.Authenticate(ctx, emp.Email, "wrong", "10.0.0.1")
	}
	if _, err := svc.Authenticate(ctx, emp.Email, testPassword, "10.0.0.1"); !errors.Is(err, ErrAccountLocked) {
		t.Fatal("expected the account to be locked")
	}

	if err := svc.Unlock(ctx, emp.ID); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}

	if _, err := svc.Authenticate(ctx, emp.Email, testPassword, "10.0.0.1"); err != nil {
		t.Fatalf("account still locked after an administrative unlock: %v", err)
	}
}
