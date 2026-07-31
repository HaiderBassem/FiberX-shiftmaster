package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/pkg/database"
)

// These tests exercise the real materialisation logic against Postgres, because the
// bug they guard against (schedules silently resetting to defaults every week) only
// shows up in the interaction between the pattern table, the `source` column and the
// upsert conflict clauses.
//
// Run with:
//
//	SHIFTMASTER_TEST_DB=shiftmaster_verify go test ./internal/service/ -run Pattern -v
//
// Skipped when SHIFTMASTER_TEST_DB is unset.

func testDB(t *testing.T) *database.DB {
	t.Helper()

	name := os.Getenv("SHIFTMASTER_TEST_DB")
	if name == "" {
		t.Skip("SHIFTMASTER_TEST_DB not set; skipping database-backed test")
	}

	host := os.Getenv("SHIFTMASTER_TEST_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("SHIFTMASTER_TEST_DB_USER")
	if user == "" {
		user = os.Getenv("USER")
	}

	db, err := database.New(config.DatabaseConfig{
		Host:              host,
		Port:              "5432",
		User:              user,
		Password:          os.Getenv("SHIFTMASTER_TEST_DB_PASSWORD"),
		DBName:            name,
		SSLMode:           "disable",
		MaxOpenConns:      4,
		MinConns:          1,
		MaxConnLifetime:   time.Minute,
		MaxConnIdleTime:   time.Minute,
		ConnectTimeout:    5 * time.Second,
		QueryTimeout:      15 * time.Second,
		LongQueryTimeout:  30 * time.Second,
		HealthCheckPeriod: time.Minute,
		MaxRetries:        1,
		RetryInterval:     time.Second,
	})
	if err != nil {
		t.Fatalf("connect to test db %q: %v", name, err)
	}
	t.Cleanup(db.Close)
	return db
}

// fixture is one department, two shifts and one employee, torn down after the test.
type fixture struct {
	db         *database.DB
	svc        *ScheduleService
	repo       repository.ScheduleRepository
	deptID     uuid.UUID
	morningID  uuid.UUID
	eveningID  uuid.UUID
	employeeID uuid.UUID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	db := testDB(t)

	suffix := uuid.NewString()[:8]
	f := &fixture{db: db}

	if err := db.QueryRow(ctx,
		`INSERT INTO departments (name, department_code) VALUES ($1,$2) RETURNING id`,
		"TestDept "+suffix, "TD"+suffix,
	).Scan(&f.deptID); err != nil {
		t.Fatalf("create department: %v", err)
	}

	for _, s := range []struct {
		target     *uuid.UUID
		name, code string
		start, end string
	}{
		{&f.morningID, "Morning " + suffix, "M" + suffix, "08:00", "16:00"},
		{&f.eveningID, "Evening " + suffix, "E" + suffix, "16:00", "00:00"},
	} {
		if err := db.QueryRow(ctx,
			`INSERT INTO shifts (name, shift_code, start_time, end_time, department_id)
			 VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			s.name, s.code, s.start, s.end, f.deptID,
		).Scan(s.target); err != nil {
			t.Fatalf("create shift %s: %v", s.name, err)
		}
	}

	// weekly_off_days = 1 (Monday) and default shift = evening. These are the values
	// the old code collapsed to, so if the pattern is ever lost the test sees evening
	// shifts and a Monday off instead of what was actually configured.
	if err := db.QueryRow(ctx,
		`INSERT INTO employees (employee_code, first_name, last_name, gender, email, password_hash,
			hire_date, role, department_id, default_shift_id, weekly_off_days, status)
		 VALUES ($1,'Test','Employee','male',$2,'x',CURRENT_DATE,'employee',$3,$4,1,'active')
		 RETURNING id`,
		"EMP"+suffix, "emp"+suffix+"@test.local", f.deptID, f.eveningID,
	).Scan(&f.employeeID); err != nil {
		t.Fatalf("create employee: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = db.Exec(cleanupCtx, `DELETE FROM employee_shifts WHERE employee_id = $1`, f.employeeID)
		_, _ = db.Exec(cleanupCtx, `DELETE FROM schedule_templates WHERE employee_id = $1`, f.employeeID)
		_, _ = db.Exec(cleanupCtx, `DELETE FROM employees WHERE id = $1`, f.employeeID)
		_, _ = db.Exec(cleanupCtx, `DELETE FROM shifts WHERE department_id = $1`, f.deptID)
		_, _ = db.Exec(cleanupCtx, `DELETE FROM departments WHERE id = $1`, f.deptID)
	})

	f.repo = repository.NewScheduleRepository(db)
	f.svc = NewScheduleService(
		f.repo,
		repository.NewEmployeeRepository(db),
		repository.NewShiftRepository(db),
		repository.NewLeaveRepository(db),
		nil, nil, db,
	)
	return f
}

// weekStart returns the Sunday n weeks from the current week.
func weekStart(n int) time.Time {
	return normalizeWeekStart(time.Now().UTC()).AddDate(0, 0, 7*n)
}

func (f *fixture) shiftOn(t *testing.T, date time.Time) *models.EmployeeShift {
	t.Helper()
	es, err := f.repo.GetEmployeeShift(context.Background(), f.employeeID, date)
	if err != nil {
		t.Fatalf("no shift row on %s: %v", date.Format("2006-01-02"), err)
	}
	return es
}

func (f *fixture) assertDay(t *testing.T, date time.Time, wantStatus string, wantShift *uuid.UUID, context string) {
	t.Helper()
	es := f.shiftOn(t, date)
	if es.ShiftStatus != wantStatus {
		t.Errorf("%s: %s status = %q, want %q", context, date.Format("Mon 2006-01-02"), es.ShiftStatus, wantStatus)
	}
	switch {
	case wantShift == nil && es.ShiftID != nil:
		t.Errorf("%s: %s shift_id = %s, want nil", context, date.Format("Mon 2006-01-02"), *es.ShiftID)
	case wantShift != nil && es.ShiftID == nil:
		t.Errorf("%s: %s shift_id = nil, want %s", context, date.Format("Mon 2006-01-02"), *wantShift)
	case wantShift != nil && *es.ShiftID != *wantShift:
		t.Errorf("%s: %s shift_id = %s, want %s", context, date.Format("Mon 2006-01-02"), *es.ShiftID, *wantShift)
	}
}

// TestPatternCarriesForwardToFutureWeeks is the regression test for the reported bug:
// a schedule set once must reappear, unchanged, in every following week.
func TestPatternCarriesForwardToFutureWeeks(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// Tuesday of next week: working the MORNING shift (not the employee's default).
	tuesday := weekStart(1).AddDate(0, 0, 2)
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, tuesday, &f.morningID, "working", nil, f.employeeID, "admin", true); err != nil {
		t.Fatalf("set tuesday shift: %v", err)
	}
	// Wednesday of next week: a permanent day off (the employee's default is a working day).
	wednesday := weekStart(1).AddDate(0, 0, 3)
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, wednesday, nil, "off", nil, f.employeeID, "admin", true); err != nil {
		t.Fatalf("set wednesday off: %v", err)
	}

	// Walk forward eight weeks, materialising each one the way a page view would.
	for week := 2; week <= 9; week++ {
		if err := f.svc.EnsureWeekSchedule(ctx, weekStart(week), &f.deptID); err != nil {
			t.Fatalf("ensure week %d: %v", week, err)
		}
		where := fmt.Sprintf("week +%d", week)
		f.assertDay(t, weekStart(week).AddDate(0, 0, 2), "working", &f.morningID, where)
		f.assertDay(t, weekStart(week).AddDate(0, 0, 3), "off", nil, where)
	}
}

// TestFutureWeekMaterialisedEarlyIsCorrected covers the original root cause: opening a
// month view used to write rows for weeks far ahead, which then froze. Those rows must
// now be re-derived once the pattern is known.
func TestFutureWeekMaterialisedEarlyIsCorrected(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// A month view reaches five weeks out before anyone has set a pattern.
	// This writes the fallback: evening shifts, Monday off.
	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(5), &f.deptID); err != nil {
		t.Fatalf("premature ensure: %v", err)
	}
	f.assertDay(t, weekStart(5).AddDate(0, 0, 2), "working", &f.eveningID, "before pattern")

	// The team leader now sets Tuesday to the morning shift, permanently.
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, weekStart(1).AddDate(0, 0, 2), &f.morningID, "working", nil, f.employeeID, "admin", true); err != nil {
		t.Fatalf("set pattern: %v", err)
	}

	// The already-written week five must now reflect it, without any further read.
	f.assertDay(t, weekStart(5).AddDate(0, 0, 2), "working", &f.morningID, "after pattern change, no re-read")

	// And it must still be right after a re-materialisation.
	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(5), &f.deptID); err != nil {
		t.Fatalf("re-ensure: %v", err)
	}
	f.assertDay(t, weekStart(5).AddDate(0, 0, 2), "working", &f.morningID, "after re-ensure")
}

// TestOneOffChangeDoesNotBecomeThePattern guards the other direction: a change marked
// as "this day only" (a swap, a cover) must not leak into following weeks.
func TestOneOffChangeDoesNotBecomeThePattern(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	thursday := weekStart(1).AddDate(0, 0, 4)
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, thursday, &f.morningID, "working", nil, f.employeeID, "admin", false); err != nil {
		t.Fatalf("set one-off shift: %v", err)
	}

	// The day itself keeps the one-off value...
	f.assertDay(t, thursday, "working", &f.morningID, "one-off day")

	// ...but the following week falls back to the employee's normal schedule.
	if err := f.svc.EnsureWeekSchedule(ctx, weekStart(2), &f.deptID); err != nil {
		t.Fatalf("ensure following week: %v", err)
	}
	f.assertDay(t, weekStart(2).AddDate(0, 0, 4), "working", &f.eveningID, "week after a one-off")
}

// TestManualAndLeaveDaysSurviveMaterialisation checks the safety rail: re-materialising
// a week must not overwrite a human decision or an approved leave.
func TestManualAndLeaveDaysSurviveMaterialisation(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	saturday := weekStart(2).AddDate(0, 0, 6)
	if _, err := f.svc.SetEmployeeShift(ctx, f.employeeID, saturday, &f.morningID, "working", nil, f.employeeID, "admin", false); err != nil {
		t.Fatalf("manual set: %v", err)
	}

	// A leave-owned row, written the way leave approval writes it.
	sunday := weekStart(2)
	ws, err := f.svc.getOrCreateWeeklySchedule(ctx, weekStart(2), weekStart(2).AddDate(0, 0, 6))
	if err != nil {
		t.Fatalf("get weekly schedule: %v", err)
	}
	reason := "annual leave"
	if err := f.repo.UpsertEmployeeShift(ctx, &models.EmployeeShift{
		ScheduleID:  ws.ID,
		EmployeeID:  f.employeeID,
		ShiftDate:   sunday,
		ShiftStatus: "leave",
		LeaveReason: &reason,
		Source:      models.ShiftSourceLeave,
	}); err != nil {
		t.Fatalf("write leave row: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := f.svc.EnsureWeekSchedule(ctx, weekStart(2), &f.deptID); err != nil {
			t.Fatalf("ensure pass %d: %v", i, err)
		}
	}

	f.assertDay(t, saturday, "working", &f.morningID, "manual row after re-ensure")
	if got := f.shiftOn(t, sunday); got.ShiftStatus != "leave" {
		t.Errorf("leave row was overwritten: status = %q, want \"leave\"", got.ShiftStatus)
	}
}

// TestSetPatternAppliesToFutureWeeksOnly checks the whole-week pattern editor: it must
// reach every future week at once and leave today and the past alone.
func TestSetPatternAppliesToFutureWeeksOnly(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// Materialise this week and two ahead using the fallback.
	for w := 0; w <= 2; w++ {
		if err := f.svc.EnsureWeekSchedule(ctx, weekStart(w), &f.deptID); err != nil {
			t.Fatalf("ensure week %d: %v", w, err)
		}
	}
	todayRow := f.shiftOn(t, today())
	statusBefore, shiftBefore := todayRow.ShiftStatus, todayRow.ShiftID

	// Friday off, everything else on the morning shift.
	days := make([]models.PatternDay, 0, 7)
	for d := 0; d < 7; d++ {
		if d == 5 {
			days = append(days, models.PatternDay{DayOfWeek: d, IsOff: true})
			continue
		}
		days = append(days, models.PatternDay{DayOfWeek: d, ShiftID: &f.morningID})
	}
	if _, err := f.svc.SetEmployeePattern(ctx, f.employeeID, days, "admin"); err != nil {
		t.Fatalf("set pattern: %v", err)
	}

	for w := 1; w <= 2; w++ {
		f.assertDay(t, weekStart(w).AddDate(0, 0, 5), "off", nil, fmt.Sprintf("friday week +%d", w))
		f.assertDay(t, weekStart(w).AddDate(0, 0, 1), "working", &f.morningID, fmt.Sprintf("monday week +%d", w))
	}

	// Today's roster is already in use and must not have moved.
	after := f.shiftOn(t, today())
	if after.ShiftStatus != statusBefore {
		t.Errorf("today's status changed: %q -> %q", statusBefore, after.ShiftStatus)
	}
	if (after.ShiftID == nil) != (shiftBefore == nil) ||
		(after.ShiftID != nil && shiftBefore != nil && *after.ShiftID != *shiftBefore) {
		t.Errorf("today's shift_id changed unexpectedly")
	}
}
