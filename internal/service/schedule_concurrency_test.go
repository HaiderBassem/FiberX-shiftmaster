package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Additional coverage for the scheduling engine (D-04). The existing suite covers
// pattern carry-forward, one-off edits and permanent changes; these fill the gaps
// the remediation plan called out: past immutability, concurrent materialisation,
// and the atomicity of a permanent edit.

// countRowsOn reports how many employee_shifts rows exist for the fixture
// employee on a date. Materialisation must be idempotent, so this is always 0
// or 1 — more would mean the upsert conflict clause is not matching.
func (f *fixture) countRowsOn(t *testing.T, date time.Time) int {
	t.Helper()

	var n int
	if err := f.db.QueryRow(context.Background(),
		`SELECT count(*) FROM employee_shifts WHERE employee_id = $1 AND shift_date = $2`,
		f.employeeID, date,
	).Scan(&n); err != nil {
		t.Fatalf("count rows on %s: %v", date.Format("2006-01-02"), err)
	}
	return n
}

// Rows in the past are history. Materialising a week that has already happened
// must not rewrite what people actually worked, even if the pattern has since
// changed.
func TestPastDaysAreNeverRewritten(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// Materialise last week first so its weekly_schedule row exists, then record
	// what the employee actually worked on one of its days.
	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(-1), &f.deptID); err != nil {
		t.Fatalf("ensure past week: %v", err)
	}

	ws, err := f.repo.GetWeeklySchedule(ctx, weekStart(-1))
	if err != nil || ws == nil {
		t.Fatalf("no weekly schedule for last week: %v", err)
	}

	pastDay := weekStart(-1).AddDate(0, 0, 2)
	if _, err := f.db.Exec(ctx,
		`INSERT INTO employee_shifts (schedule_id, employee_id, shift_date, shift_id, shift_status, source)
		 VALUES ($1,$2,$3,$4,'working','generated')
		 ON CONFLICT (employee_id, shift_date)
		 DO UPDATE SET shift_id = EXCLUDED.shift_id, shift_status = EXCLUDED.shift_status, source = EXCLUDED.source`,
		ws.ID, f.employeeID, pastDay, f.morningID,
	); err != nil {
		t.Fatalf("record historical shift: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.db.Exec(context.Background(),
			`DELETE FROM employee_shifts WHERE employee_id = $1 AND shift_date = $2`, f.employeeID, pastDay)
	})

	// The pattern now says that weekday is a day off.
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, weekStart(1).AddDate(0, 0, 2),
		nil, "off", nil, f.employeeID, "admin", true); err != nil {
		t.Fatalf("set pattern: %v", err)
	}

	// Re-materialise the week that has already passed.
	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(-1), &f.deptID); err != nil {
		t.Fatalf("ensure past week: %v", err)
	}

	es := f.shiftOn(t, pastDay)
	if es.ShiftStatus != "working" {
		t.Errorf("historical status = %q, want working: a past day was rewritten from the current pattern", es.ShiftStatus)
	}
	if es.ShiftID == nil || *es.ShiftID != f.morningID {
		t.Error("historical shift assignment was rewritten")
	}
}

// Several browsers opening the same week at once all trigger materialisation.
// The upserts must converge on one row per day rather than racing into
// duplicates or unique-violation errors.
func TestConcurrentMaterialisationIsIdempotent(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	target := weekStart(2)

	// Give the week something to materialise.
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, weekStart(1).AddDate(0, 0, 2),
		&f.morningID, "working", nil, f.employeeID, "admin", true); err != nil {
		t.Fatalf("set pattern: %v", err)
	}

	const concurrency = 8
	var wg sync.WaitGroup
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = f.svc.EnsureWeekSchedule(context.Background(), target, &f.deptID)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent materialisation %d failed: %v", i, err)
		}
	}

	for offset := 0; offset < 7; offset++ {
		day := target.AddDate(0, 0, offset)
		if n := f.countRowsOn(t, day); n > 1 {
			t.Errorf("%s has %d rows after concurrent materialisation, want at most 1",
				day.Format("Mon 2006-01-02"), n)
		}
	}
}

// Repeated materialisation of the same week must not accumulate rows or drift.
func TestRepeatedMaterialisationIsStable(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	tuesday := weekStart(1).AddDate(0, 0, 2)
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, tuesday, &f.morningID, "working", nil, f.employeeID, "admin", true); err != nil {
		t.Fatalf("set pattern: %v", err)
	}

	target := weekStart(3)
	for i := 0; i < 5; i++ {
		if err := f.svc.EnsureWeekSchedule(ctx, target, &f.deptID); err != nil {
			t.Fatalf("ensure pass %d: %v", i, err)
		}
	}

	day := target.AddDate(0, 0, 2)
	if n := f.countRowsOn(t, day); n != 1 {
		t.Errorf("%s has %d rows after five materialisations, want exactly 1", day.Format("2006-01-02"), n)
	}
	f.assertDay(t, day, "working", &f.morningID, "after repeated materialisation")
}

// A permanent edit writes both the day and the weekly template. Those must land
// together: an earlier bug returned an error to the user after the day had
// already been saved, so the UI reported failure on a change that had taken
// effect.
func TestPermanentEditWritesDayAndTemplateTogether(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	thursday := weekStart(1).AddDate(0, 0, 4)
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, thursday, &f.morningID, "working", nil, f.employeeID, "admin", true); err != nil {
		t.Fatalf("permanent edit: %v", err)
	}

	// The day itself.
	f.assertDay(t, thursday, "working", &f.morningID, "permanent edit, the edited day")

	// And the template row backing it.
	var templateShift *uuid.UUID
	var isOff bool
	if err := f.db.QueryRow(ctx,
		`SELECT shift_id, is_off FROM schedule_templates
		 WHERE employee_id = $1 AND day_of_week = $2`,
		f.employeeID, int(thursday.Weekday()),
	).Scan(&templateShift, &isOff); err != nil {
		t.Fatalf("no template row was written for weekday %d: %v", int(thursday.Weekday()), err)
	}
	if isOff {
		t.Error("template says the day is off, but a working shift was set")
	}
	if templateShift == nil || *templateShift != f.morningID {
		t.Error("template shift does not match the shift that was set")
	}

	// A later week reflects it, proving the two writes agree.
	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(4), &f.deptID); err != nil {
		t.Fatalf("ensure later week: %v", err)
	}
	f.assertDay(t, weekStart(4).AddDate(0, 0, 4), "working", &f.morningID, "four weeks on")
}

// A rejected edit must leave nothing behind.
func TestInvalidShiftStatusIsRejectedWithoutWriting(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	friday := weekStart(1).AddDate(0, 0, 5)
	before := f.countRowsOn(t, friday)

	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, friday, nil, "not-a-real-status", nil, f.employeeID, "admin", true); err == nil {
		t.Fatal("an invalid shift status was accepted")
	}

	if after := f.countRowsOn(t, friday); after != before {
		t.Errorf("row count changed from %d to %d on a rejected edit", before, after)
	}

	var templates int
	if err := f.db.QueryRow(ctx,
		`SELECT count(*) FROM schedule_templates WHERE employee_id = $1 AND day_of_week = $2`,
		f.employeeID, int(friday.Weekday()),
	).Scan(&templates); err != nil {
		t.Fatalf("count templates: %v", err)
	}
	if templates != 0 {
		t.Errorf("a rejected edit wrote %d template row(s)", templates)
	}
}
