package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// These tests pin the behaviour of D-01: leave semantics must come from explicit
// columns on leave_types, not from matching the type's name. Administrators can
// rename and translate types freely through the leave-type screen, and doing so
// used to change how scheduling and department caps behaved with no error
// anywhere.
//
// Run with SHIFTMASTER_TEST_DB pointing at a migrated database.

// leaveTypeFixture creates a leave type with explicit semantics and removes it
// afterwards.
func (f *fixture) createLeaveType(t *testing.T, nameEn, nameAr, unit string, isHourly, bypassesLimit bool) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	var id uuid.UUID
	err := f.db.QueryRow(ctx,
		`INSERT INTO leave_types (name_ar, name_en, unit, reset_cycle, is_active, is_hourly, bypasses_daily_limit)
		 VALUES ($1,$2,$3,'annual',true,$4,$5) RETURNING id`,
		nameAr, nameEn, unit, isHourly, bypassesLimit,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create leave type %q: %v", nameEn, err)
	}

	t.Cleanup(func() {
		cleanup := context.Background()
		_, _ = f.db.Exec(cleanup, `DELETE FROM leaves WHERE leave_type_id = $1`, id)
		_, _ = f.db.Exec(cleanup, `DELETE FROM leave_types WHERE id = $1`, id)
	})
	return id
}

// createApprovedLeave inserts a fully approved leave for the fixture employee.
//
// The leave_status enum has no 'approved' label: approval is represented by
// 'approved_by_team_leader' / 'approved_by_manager'. Using the wrong literal is
// a mistake the codebase has already made once.
func (f *fixture) createApprovedLeave(t *testing.T, leaveTypeID uuid.UUID, start, end time.Time, reason string) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	var id uuid.UUID
	err := f.db.QueryRow(ctx,
		// total_days is a generated column; the database computes it.
		`INSERT INTO leaves (employee_id, leave_type_id, start_date, end_date, reason, status, applied_date)
		 VALUES ($1,$2,$3,$4,$5,'approved_by_manager',CURRENT_DATE) RETURNING id`,
		f.employeeID, leaveTypeID, start, end, reason,
	).Scan(&id)
	if err != nil {
		t.Fatalf("create leave: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(context.Background(), `DELETE FROM leaves WHERE id = $1`, id)
	})
	return id
}

// renameLeaveType simulates an administrator editing the type through the UI.
func (f *fixture) renameLeaveType(t *testing.T, id uuid.UUID, nameEn, nameAr string) {
	t.Helper()
	if _, err := f.db.Exec(context.Background(),
		`UPDATE leave_types SET name_en = $1, name_ar = $2 WHERE id = $3`, nameEn, nameAr, id); err != nil {
		t.Fatalf("rename leave type: %v", err)
	}
}

// The core D-01 regression: a type flagged hourly is treated as hourly no matter
// what it is called, and renaming it does not change that.
func TestHourlyLeaveFollowsTheFlagNotTheName(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// Deliberately named nothing like "hourly" or "زمنية", the two literals the
	// old code matched on.
	hourlyID := f.createLeaveType(t, "Partial Day Absence", "غياب جزئي", "hours", true, false)

	day := weekStart(1).AddDate(0, 0, 2)
	f.createApprovedLeave(t, hourlyID, day, day, "dentist")

	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(1), &f.deptID); err != nil {
		t.Fatalf("ensure week: %v", err)
	}

	es := f.shiftOn(t, day)
	if es.ShiftStatus != "leave" {
		t.Fatalf("status = %q, want leave", es.ShiftStatus)
	}
	if es.LeaveReason == nil {
		t.Fatal("leave reason was not recorded")
	}
	if !strings.HasPrefix(*es.LeaveReason, models.HourlyLeaveReasonPrefix) {
		t.Errorf("leave reason = %q, want the %q prefix: an hourly type was not recognised",
			*es.LeaveReason, models.HourlyLeaveReasonPrefix)
	}
}

// The mirror case: a type *named* "Hourly" but not flagged is not hourly. Under
// the old code the name alone decided it.
func TestTypeNamedHourlyWithoutTheFlagIsNotHourly(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	notHourlyID := f.createLeaveType(t, "Hourly", "زمنية", "days", false, false)

	day := weekStart(1).AddDate(0, 0, 2)
	f.createApprovedLeave(t, notHourlyID, day, day, "full day off")

	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(1), &f.deptID); err != nil {
		t.Fatalf("ensure week: %v", err)
	}

	es := f.shiftOn(t, day)
	if es.LeaveReason != nil && strings.HasPrefix(*es.LeaveReason, models.HourlyLeaveReasonPrefix) {
		t.Errorf("leave reason = %q: the type's name overrode its explicit flag", *es.LeaveReason)
	}
}

// Renaming a type must not alter scheduling behaviour. This is what the previous
// implementation could not survive.
func TestRenamingALeaveTypeDoesNotChangeSchedulingSemantics(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	typeID := f.createLeaveType(t, "Hourly", "زمنية", "hours", true, false)

	firstDay := weekStart(1).AddDate(0, 0, 2)
	f.createApprovedLeave(t, typeID, firstDay, firstDay, "before rename")

	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(1), &f.deptID); err != nil {
		t.Fatalf("ensure week 1: %v", err)
	}
	before := f.shiftOn(t, firstDay)
	if before.LeaveReason == nil || !strings.HasPrefix(*before.LeaveReason, models.HourlyLeaveReasonPrefix) {
		t.Fatalf("baseline failed: reason = %v", before.LeaveReason)
	}

	// An administrator renames the type, in both languages.
	f.renameLeaveType(t, typeID, "Medical Appointment", "موعد طبي")

	secondDay := weekStart(2).AddDate(0, 0, 2)
	f.createApprovedLeave(t, typeID, secondDay, secondDay, "after rename")

	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(2), &f.deptID); err != nil {
		t.Fatalf("ensure week 2: %v", err)
	}

	after := f.shiftOn(t, secondDay)
	if after.LeaveReason == nil || !strings.HasPrefix(*after.LeaveReason, models.HourlyLeaveReasonPrefix) {
		t.Errorf("after rename, reason = %v: renaming the type changed scheduling behaviour", after.LeaveReason)
	}
}

// The department daily cap exemption must likewise follow the flag. Previously it
// matched name_en against "Emergency", so translating the type removed the
// exemption silently.
func TestDailyLimitExemptionFollowsTheFlagNotTheName(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	leaveRepo := repository.NewLeaveRepository(f.db)

	day := weekStart(1).AddDate(0, 0, 2)

	// Exempt, but named in Arabic rather than "Emergency".
	exemptID := f.createLeaveType(t, "حالة طارئة", "حالة طارئة", "days", false, true)
	f.createApprovedLeave(t, exemptID, day, day, "urgent")

	count, err := leaveRepo.GetOverlappingLeavesCount(ctx, f.deptID, day, day)
	if err != nil {
		t.Fatalf("count overlapping: %v", err)
	}
	if count != 0 {
		t.Errorf("exempt leave counted %d times toward the department cap, want 0", count)
	}

	// A type literally named "Emergency" but not flagged must count.
	countedID := f.createLeaveType(t, "Emergency", "طارئة", "days", false, false)
	f.createApprovedLeave(t, countedID, day, day, "not actually exempt")

	count, err = leaveRepo.GetOverlappingLeavesCount(ctx, f.deptID, day, day)
	if err != nil {
		t.Fatalf("count overlapping: %v", err)
	}
	if count != 1 {
		t.Errorf("counted %d leaves toward the cap, want 1: the name overrode the flag", count)
	}
}

// The joined flag must survive the repository projection, or every consumer would
// silently see false.
func TestLeaveQueriesCarryTheHourlyFlag(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	leaveRepo := repository.NewLeaveRepository(f.db)

	hourlyID := f.createLeaveType(t, "Short Absence", "غياب قصير", "hours", true, false)
	day := weekStart(1).AddDate(0, 0, 2)
	f.createApprovedLeave(t, hourlyID, day, day, "appointment")

	leaves, err := leaveRepo.GetByEmployee(ctx, f.employeeID)
	if err != nil {
		t.Fatalf("get leaves: %v", err)
	}
	if len(leaves) == 0 {
		t.Fatal("no leaves returned")
	}

	found := false
	for _, l := range leaves {
		if l.LeaveTypeID == hourlyID {
			found = true
			if !l.LeaveTypeIsHourly {
				t.Error("LeaveTypeIsHourly is false on a leave whose type is flagged hourly")
			}
		}
	}
	if !found {
		t.Error("the created leave was not returned by GetByEmployee")
	}
}
