package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/temporal"
)

// These tests pin the leave-request rules the assistant's approval flow relies
// on: cross-midnight hourly windows are representable (the last hour of a
// 16:30→00:30 shift is 23:30→00:30 on ONE business date), past-date checks run
// on the Baghdad business calendar rather than UTC truncation, and an employee
// cannot double-book overlapping leaves.
//
// Run with SHIFTMASTER_TEST_DB pointing at a migrated database.

// newLeaveService builds a LeaveService over the fixture's database with the
// clock pinned, so the after-midnight cases are reproducible at any wall time.
func (f *fixture) newLeaveService(t *testing.T, now time.Time) *LeaveService {
	t.Helper()
	svc := NewLeaveService(
		repository.NewLeaveRepository(f.db),
		repository.NewEmployeeRepository(f.db),
		repository.NewDepartmentRepository(f.db),
		repository.NewScheduleRepository(f.db),
		repository.NewLeaveBalanceRepository(f.db),
		repository.NewLeaveTypeRepository(f.db),
		NewNotificationService(repository.NewNotificationRepository(f.db)),
		NewEmailService(config.GraphAPIConfig{}),
		nil, // push: nil is a supported configuration and must never panic
	)
	svc.now = func() time.Time { return now }
	return svc
}

func strp(s string) *string { return &s }

// leaveReq builds a request for the fixture employee.
func leaveReq(f *fixture, typeID uuid.UUID, start, end time.Time, startClock, endClock string) *models.Leave {
	l := &models.Leave{
		EmployeeID:  f.employeeID,
		LeaveTypeID: typeID,
		StartDate:   start,
		EndDate:     end,
		Reason:      strp("test"),
	}
	if startClock != "" {
		l.StartTime = strp(startClock)
		l.EndTime = strp(endClock)
	}
	return l
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// baghdad builds the pinned "now".
func baghdad(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, temporal.Location())
}

func TestHourlyLeaveMayCrossMidnight(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	hourlyID := f.createLeaveType(t, "Hourly X "+uuid.NewString()[:4], "زمنية", "hours", true, false)

	// 23:40 during the 17th's 16:30→00:30 shift: request the last hour.
	svc := f.newLeaveService(t, baghdad(2026, 8, 17, 23, 40))
	l := leaveReq(f, hourlyID, date(2026, 8, 17), date(2026, 8, 17), "23:30", "00:30")

	if err := svc.RequestLeave(ctx, l); err != nil {
		t.Fatalf("cross-midnight hourly leave rejected: %v", err)
	}
	if l.Status != "pending" {
		t.Fatalf("status = %q", l.Status)
	}
}

func TestHourlyLeaveAfterMidnightStillOnYesterdaysBusinessDate(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	hourlyID := f.createLeaveType(t, "Hourly Y "+uuid.NewString()[:4], "زمنية", "hours", true, false)

	// 00:10 on the 18th: the 17th's shift is still running, and the last hour
	// (23:30→00:30, business date the 17th) must be accepted even though the
	// 17th is "yesterday" on the calendar.
	svc := f.newLeaveService(t, baghdad(2026, 8, 18, 0, 10))
	l := leaveReq(f, hourlyID, date(2026, 8, 17), date(2026, 8, 17), "23:30", "00:30")

	if err := svc.RequestLeave(ctx, l); err != nil {
		t.Fatalf("after-midnight request for the running shift rejected: %v", err)
	}
}

func TestHourlyLeaveEntirelyInThePastIsRejected(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	hourlyID := f.createLeaveType(t, "Hourly Z "+uuid.NewString()[:4], "زمنية", "hours", true, false)

	svc := f.newLeaveService(t, baghdad(2026, 8, 18, 0, 10))

	// A window that already ended (20:00→21:00 on the 17th).
	l := leaveReq(f, hourlyID, date(2026, 8, 17), date(2026, 8, 17), "20:00", "21:00")
	if err := svc.ValidateLeaveRequest(ctx, l); err == nil {
		t.Fatal("a fully elapsed hourly window must be rejected")
	}

	// Two business days back is past regardless of the window.
	l = leaveReq(f, hourlyID, date(2026, 8, 16), date(2026, 8, 16), "23:30", "00:30")
	if err := svc.ValidateLeaveRequest(ctx, l); err == nil {
		t.Fatal("an hourly window two business days back must be rejected")
	}
}

func TestDayLeavePastDateUsesBaghdadCalendar(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	dayID := f.createLeaveType(t, "Annual "+uuid.NewString()[:4], "سنوية", "days", false, false)

	// At 00:10 Baghdad on the 18th (21:10 UTC on the 17th), the 17th is in the
	// past. The old UTC truncation would still have accepted it.
	svc := f.newLeaveService(t, baghdad(2026, 8, 18, 0, 10))
	l := leaveReq(f, dayID, date(2026, 8, 17), date(2026, 8, 17), "", "")
	if err := svc.ValidateLeaveRequest(ctx, l); err == nil {
		t.Fatal("yesterday (Baghdad) must be rejected for a day leave")
	}

	// Today (Baghdad, the 18th) is fine.
	l = leaveReq(f, dayID, date(2026, 8, 18), date(2026, 8, 18), "", "")
	if err := svc.ValidateLeaveRequest(ctx, l); err != nil {
		t.Fatalf("today must be accepted: %v", err)
	}
}

func TestTimesMustMatchLeaveTypeUnit(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	hourlyID := f.createLeaveType(t, "Hourly W "+uuid.NewString()[:4], "زمنية", "hours", true, false)
	dayID := f.createLeaveType(t, "Annual W "+uuid.NewString()[:4], "سنوية", "days", false, false)

	svc := f.newLeaveService(t, baghdad(2026, 8, 17, 12, 0))

	if err := svc.ValidateLeaveRequest(ctx,
		leaveReq(f, hourlyID, date(2026, 8, 17), date(2026, 8, 17), "", "")); err == nil {
		t.Fatal("hourly type without a time window must be rejected")
	}
	if err := svc.ValidateLeaveRequest(ctx,
		leaveReq(f, dayID, date(2026, 8, 17), date(2026, 8, 17), "10:00", "11:00")); err == nil {
		t.Fatal("day type with a time window must be rejected")
	}
	if err := svc.ValidateLeaveRequest(ctx,
		leaveReq(f, hourlyID, date(2026, 8, 17), date(2026, 8, 18), "10:00", "11:00")); err == nil {
		t.Fatal("hourly leave spanning two dates must be rejected")
	}
	if err := svc.ValidateLeaveRequest(ctx,
		leaveReq(f, hourlyID, date(2026, 8, 17), date(2026, 8, 17), "10:00", "10:00")); err == nil {
		t.Fatal("zero-length window must be rejected")
	}
}

func TestOverlappingLeavesAreRejected(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	hourlyID := f.createLeaveType(t, "Hourly O "+uuid.NewString()[:4], "زمنية", "hours", true, false)
	dayID := f.createLeaveType(t, "Annual O "+uuid.NewString()[:4], "سنوية", "days", false, false)

	svc := f.newLeaveService(t, baghdad(2026, 8, 17, 12, 0))

	// Seed a real pending hourly leave: 23:30→00:30 on the 17th.
	first := leaveReq(f, hourlyID, date(2026, 8, 17), date(2026, 8, 17), "23:30", "00:30")
	if err := svc.RequestLeave(ctx, first); err != nil {
		t.Fatalf("seed request: %v", err)
	}

	// The same window again: double-booked.
	dup := leaveReq(f, hourlyID, date(2026, 8, 17), date(2026, 8, 17), "23:30", "00:30")
	if err := svc.ValidateLeaveRequest(ctx, dup); err == nil || !strings.Contains(err.Error(), "already have") {
		t.Fatalf("duplicate window not rejected: %v", err)
	}

	// A window on the NEXT calendar date that intersects the wrapped part
	// (00:00→01:00 on the 18th overlaps 23:30→00:30-of-the-17th at 00:00–00:30).
	crossDay := leaveReq(f, hourlyID, date(2026, 8, 18), date(2026, 8, 18), "00:00", "01:00")
	if err := svc.ValidateLeaveRequest(ctx, crossDay); err == nil || !strings.Contains(err.Error(), "already have") {
		t.Fatalf("cross-date absolute overlap not rejected: %v", err)
	}

	// Clear of the wrapped window: accepted.
	clear := leaveReq(f, hourlyID, date(2026, 8, 18), date(2026, 8, 18), "01:00", "02:00")
	if err := svc.ValidateLeaveRequest(ctx, clear); err != nil {
		t.Fatalf("non-overlapping window rejected: %v", err)
	}

	// A full-day leave covering the hourly leave's business date.
	fullDay := leaveReq(f, dayID, date(2026, 8, 17), date(2026, 8, 18), "", "")
	if err := svc.ValidateLeaveRequest(ctx, fullDay); err == nil || !strings.Contains(err.Error(), "already have") {
		t.Fatalf("full-day overlap with hourly leave not rejected: %v", err)
	}
}

func TestHourlyBalanceCountsWrappedWindowCorrectly(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// A type with a 2-hour allocation.
	var typeID uuid.UUID
	if err := f.db.QueryRow(ctx,
		`INSERT INTO leave_types (name_ar, name_en, unit, reset_cycle, is_active, is_hourly, bypasses_daily_limit, days_per_year)
		 VALUES ('زمنية محدودة','Limited Hourly ' || $1,'hours','annual',true,true,false,2) RETURNING id`,
		uuid.NewString()[:4],
	).Scan(&typeID); err != nil {
		t.Fatalf("create limited type: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(context.Background(), `DELETE FROM leaves WHERE leave_type_id = $1`, typeID)
		_, _ = f.db.Exec(context.Background(), `DELETE FROM employee_leave_balances WHERE leave_type_id = $1`, typeID)
		_, _ = f.db.Exec(context.Background(), `DELETE FROM leave_types WHERE id = $1`, typeID)
	})

	svc := f.newLeaveService(t, baghdad(2026, 8, 17, 12, 0))

	// A wrapped 1-hour window consumes 1 hour, not -23.
	first := leaveReq(f, typeID, date(2026, 8, 17), date(2026, 8, 17), "23:30", "00:30")
	if err := svc.RequestLeave(ctx, first); err != nil {
		t.Fatalf("first hour: %v", err)
	}
	// One more hour fits the 2h allocation…
	second := leaveReq(f, typeID, date(2026, 8, 18), date(2026, 8, 18), "10:00", "11:00")
	if err := svc.RequestLeave(ctx, second); err != nil {
		t.Fatalf("second hour: %v", err)
	}
	// …and a third does not.
	third := leaveReq(f, typeID, date(2026, 8, 19), date(2026, 8, 19), "10:00", "11:00")
	if err := svc.ValidateLeaveRequest(ctx, third); err == nil || !strings.Contains(err.Error(), "insufficient") {
		t.Fatalf("third hour must exhaust the balance, got: %v", err)
	}
}

func TestRejectRequiresAnUndecidedLeave(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	dayID := f.createLeaveType(t, "Annual R "+uuid.NewString()[:4], "سنوية", "days", false, false)

	leaveID := f.createApprovedLeave(t, dayID, date(2026, 9, 1), date(2026, 9, 2), "trip")

	svc := f.newLeaveService(t, baghdad(2026, 8, 17, 12, 0))
	err := svc.RejectLeave(ctx, leaveID, f.employeeID, "manager", "no")
	if err == nil || !strings.Contains(err.Error(), "cannot be rejected") {
		t.Fatalf("rejecting an approved leave must fail, got: %v", err)
	}
}
