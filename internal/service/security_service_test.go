package service

import (
	"context"
	"sync"
	"testing"
	"time"
)

// sync.Map keeps the map safe but not the pointer it stores. Before the per-entry
// mutex, concurrent failures from one address lost increments, which let a caller
// issuing requests in parallel stay under the block threshold indefinitely.
// Run with -race to see the original failure.
func TestRecordFailedLoginCountsEveryConcurrentAttempt(t *testing.T) {
	svc := NewSecurityService(nil, 1_000_000, time.Hour)
	ctx := context.Background()

	const attempts = 200
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.RecordFailedLogin(ctx, "203.0.113.7")
		}()
	}
	wg.Wait()

	if got := svc.FailureCount("203.0.113.7"); got != attempts {
		t.Errorf("counter = %d after %d concurrent failures, want %d (increments were lost)", got, attempts, attempts)
	}
}

func TestRecordFailedLoginIsolatesAddresses(t *testing.T) {
	svc := NewSecurityService(nil, 1_000_000, time.Hour)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_ = svc.RecordFailedLogin(ctx, "198.51.100.1")
	}
	_ = svc.RecordFailedLogin(ctx, "198.51.100.2")

	if got := svc.FailureCount("198.51.100.1"); got != 3 {
		t.Errorf("first address counter = %d, want 3", got)
	}
	if got := svc.FailureCount("198.51.100.2"); got != 1 {
		t.Errorf("second address counter = %d, want 1", got)
	}
}

func TestResetFailedLoginClearsCounter(t *testing.T) {
	svc := NewSecurityService(nil, 1_000_000, time.Hour)
	ctx := context.Background()

	_ = svc.RecordFailedLogin(ctx, "198.51.100.3")
	_ = svc.RecordFailedLogin(ctx, "198.51.100.3")
	svc.ResetFailedLogin("198.51.100.3")

	if got := svc.FailureCount("198.51.100.3"); got != 0 {
		t.Errorf("counter = %d after reset, want 0", got)
	}
}

func TestEmptyClientIPIsIgnored(t *testing.T) {
	svc := NewSecurityService(nil, 1_000_000, time.Hour)
	if err := svc.RecordFailedLogin(context.Background(), ""); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got := svc.FailureCount(""); got != 0 {
		t.Errorf("an empty address was tracked (%d)", got)
	}
}

// Stale counters must be evicted, otherwise the map grows once per distinct
// source address and is never reclaimed.
func TestStaleCountersAreSwept(t *testing.T) {
	svc := NewSecurityService(nil, 1_000_000, time.Hour)
	ctx := context.Background()

	_ = svc.RecordFailedLogin(ctx, "198.51.100.4")
	if svc.FailureCount("198.51.100.4") == 0 {
		t.Fatal("counter was not recorded")
	}

	// Age the entry past the window and force the next sweep to run.
	val, _ := svc.failedLogins.Load("198.51.100.4")
	entry := val.(*ipAttempts)
	entry.mu.Lock()
	entry.lastError = time.Now().Add(-2 * time.Hour)
	entry.mu.Unlock()

	svc.sweepMu.Lock()
	svc.lastSweep = time.Time{}
	svc.sweepMu.Unlock()

	_ = svc.RecordFailedLogin(ctx, "198.51.100.5")

	if _, still := svc.failedLogins.Load("198.51.100.4"); still {
		t.Error("a stale counter survived the sweep")
	}
}

// A counter that has gone quiet for a full window starts over rather than
// accumulating across months.
func TestCounterRestartsAfterWindow(t *testing.T) {
	svc := NewSecurityService(nil, 1_000_000, time.Hour)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		_ = svc.RecordFailedLogin(ctx, "198.51.100.6")
	}

	val, _ := svc.failedLogins.Load("198.51.100.6")
	entry := val.(*ipAttempts)
	entry.mu.Lock()
	entry.lastError = time.Now().Add(-2 * time.Hour)
	entry.mu.Unlock()

	_ = svc.RecordFailedLogin(ctx, "198.51.100.6")

	if got := svc.FailureCount("198.51.100.6"); got != 1 {
		t.Errorf("counter = %d after the window elapsed, want 1", got)
	}
}
