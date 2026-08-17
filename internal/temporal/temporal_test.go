package temporal

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// bg builds an instant in the business zone.
func bg(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, Location())
}

// dateUTC builds a DATE-column value (midnight UTC), the repository convention.
func dateUTC(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// clock builds a TIME-column value the way pgx returns it (anchored 2000-01-01).
func clock(hh, mm int) time.Time {
	return time.Date(2000, 1, 1, hh, mm, 0, 0, time.UTC)
}

func overnightShift(t *testing.T) Instance {
	t.Helper()
	start, end, overnight := Materialize(dateUTC(2026, 8, 17), clock(16, 30), clock(0, 30))
	if !overnight {
		t.Fatalf("16:30→00:30 must be detected as overnight")
	}
	return Instance{
		EmployeeShiftID: uuid.New(),
		BusinessDate:    dateUTC(2026, 8, 17),
		StartAt:         start,
		EndAt:           end,
		Overnight:       true,
		Status:          "working",
	}
}

func TestMaterializeSameDay(t *testing.T) {
	start, end, overnight := Materialize(dateUTC(2026, 8, 17), clock(8, 0), clock(16, 0))
	if overnight {
		t.Fatalf("08:00→16:00 misclassified as overnight")
	}
	if !start.Equal(bg(2026, 8, 17, 8, 0)) || !end.Equal(bg(2026, 8, 17, 16, 0)) {
		t.Fatalf("got %v→%v", start, end)
	}
	if d := end.Sub(start); d != 8*time.Hour {
		t.Fatalf("duration = %v", d)
	}
}

func TestMaterializeOvernight(t *testing.T) {
	in := overnightShift(t)
	if !in.StartAt.Equal(bg(2026, 8, 17, 16, 30)) {
		t.Fatalf("start = %v", in.StartAt)
	}
	// Ends the NEXT calendar day.
	if !in.EndAt.Equal(bg(2026, 8, 18, 0, 30)) {
		t.Fatalf("end = %v", in.EndAt)
	}
	if in.Duration() != 8*time.Hour {
		t.Fatalf("duration = %v", in.Duration())
	}
}

func TestMaterializeEndsExactlyAtMidnight(t *testing.T) {
	start, end, overnight := Materialize(dateUTC(2026, 8, 17), clock(16, 0), clock(0, 0))
	if !overnight {
		t.Fatalf("a shift ending at 00:00 crosses into the next day")
	}
	if !end.Equal(bg(2026, 8, 18, 0, 0)) {
		t.Fatalf("end = %v", end)
	}
	_ = start
}

// The two regression cases from the specification, verbatim.

func TestLastHourBeforeMidnight(t *testing.T) {
	// Shift 2026-08-17 16:30 → 2026-08-18 00:30. At 23:40 the last hour is
	// 23:30 → 00:30.
	in := overnightShift(t)
	now := bg(2026, 8, 17, 23, 40)
	if !in.Contains(now) {
		t.Fatalf("shift must be active at 23:40")
	}
	w, err := LastWindow(in, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Start.Equal(bg(2026, 8, 17, 23, 30)) || !w.End.Equal(bg(2026, 8, 18, 0, 30)) {
		t.Fatalf("last hour = %v → %v", w.Start, w.End)
	}
	if w.StartClock() != "23:30" || w.EndClock() != "00:30" {
		t.Fatalf("clocks = %s → %s", w.StartClock(), w.EndClock())
	}
}

func TestLastHourAfterMidnight(t *testing.T) {
	// Same shift, but asked at 00:10 on the 18th: still the 17th's shift, and
	// the last hour is still 23:30 (17th) → 00:30 (18th) — NOT 23:30 on the
	// 18th.
	in := overnightShift(t)
	now := bg(2026, 8, 18, 0, 10)
	if !in.Contains(now) {
		t.Fatalf("the 17th's shift must still be active at 00:10 on the 18th")
	}
	w, err := LastWindow(in, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Start.Equal(bg(2026, 8, 17, 23, 30)) || !w.End.Equal(bg(2026, 8, 18, 0, 30)) {
		t.Fatalf("last hour = %v → %v", w.Start, w.End)
	}
}

func TestActivenessBoundaries(t *testing.T) {
	in := overnightShift(t)
	cases := []struct {
		now    time.Time
		active bool
	}{
		{bg(2026, 8, 17, 16, 29), false}, // before start
		{bg(2026, 8, 17, 16, 30), true},  // exact start is inside
		{bg(2026, 8, 17, 23, 59), true},
		{bg(2026, 8, 18, 0, 0), true},  // midnight itself
		{bg(2026, 8, 18, 0, 29), true}, // final minute
		{bg(2026, 8, 18, 0, 30), false},
		{bg(2026, 8, 18, 23, 40), false}, // the trap: same clock time, next day
	}
	for _, c := range cases {
		if got := in.Contains(c.now); got != c.active {
			t.Errorf("Contains(%v) = %v, want %v", c.now, got, c.active)
		}
	}
}

func TestLastAndFirstWindows(t *testing.T) {
	in := overnightShift(t)

	w, err := LastWindow(in, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// Last 30 minutes of a shift ending 00:30 = 00:00 → 00:30, both on the 18th.
	if !w.Start.Equal(bg(2026, 8, 18, 0, 0)) || !w.End.Equal(bg(2026, 8, 18, 0, 30)) {
		t.Fatalf("last 30m = %v → %v", w.Start, w.End)
	}

	w, err = FirstWindow(in, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !w.Start.Equal(bg(2026, 8, 17, 16, 30)) || !w.End.Equal(bg(2026, 8, 17, 17, 30)) {
		t.Fatalf("first hour = %v → %v", w.Start, w.End)
	}
}

func TestWindowLongerThanShift(t *testing.T) {
	in := overnightShift(t)
	if _, err := LastWindow(in, 9*time.Hour); err == nil {
		t.Fatalf("9h window inside an 8h shift must be rejected")
	}
	if _, err := LastWindow(in, 0); err == nil {
		t.Fatalf("zero duration must be rejected")
	}
	if _, err := LastWindow(in, -time.Hour); err == nil {
		t.Fatalf("negative duration must be rejected")
	}
}

func TestExplicitWindow(t *testing.T) {
	in := overnightShift(t)

	// A window given as 23:30→00:30 wraps midnight and sits inside the shift.
	w, err := ExplicitWindow(in, "23:30", "00:30")
	if err != nil {
		t.Fatal(err)
	}
	if !w.Start.Equal(bg(2026, 8, 17, 23, 30)) || !w.End.Equal(bg(2026, 8, 18, 0, 30)) {
		t.Fatalf("window = %v → %v", w.Start, w.End)
	}

	// Clocks earlier than the shift start refer to the after-midnight leg.
	w, err = ExplicitWindow(in, "00:00", "00:30")
	if err != nil {
		t.Fatal(err)
	}
	if !w.Start.Equal(bg(2026, 8, 18, 0, 0)) {
		t.Fatalf("00:00 must resolve to the 18th, got %v", w.Start)
	}

	// Outside the shift in either direction.
	if _, err := ExplicitWindow(in, "15:00", "16:00"); err == nil {
		t.Fatalf("window before shift start must be rejected")
	}
	if _, err := ExplicitWindow(in, "00:15", "01:00"); err == nil {
		t.Fatalf("window past shift end must be rejected")
	}
	if _, err := ExplicitWindow(in, "25:00", "26:00"); err == nil {
		t.Fatalf("nonsense clock must be rejected")
	}
}

func TestResolvePrefersActiveOvernightOverUpcomingToday(t *testing.T) {
	// At 00:10 on the 18th the employee holds yesterday's still-running shift
	// AND has a shift today at 16:30. Resolve must pick yesterday's as active
	// and today's as upcoming.
	yesterday := overnightShift(t)

	s, e, _ := Materialize(dateUTC(2026, 8, 18), clock(16, 30), clock(0, 30))
	today := Instance{BusinessDate: dateUTC(2026, 8, 18), StartAt: s, EndAt: e, Overnight: true, Status: "working"}

	res := Resolve(bg(2026, 8, 18, 0, 10), []Instance{yesterday, today})
	if res.Active == nil || !res.Active.BusinessDate.Equal(dateUTC(2026, 8, 17)) {
		t.Fatalf("active = %+v, want the 17th's shift", res.Active)
	}
	if res.Upcoming == nil || !res.Upcoming.BusinessDate.Equal(dateUTC(2026, 8, 18)) {
		t.Fatalf("upcoming = %+v, want the 18th's shift", res.Upcoming)
	}
}

func TestResolveNoShift(t *testing.T) {
	res := Resolve(bg(2026, 8, 18, 12, 0), nil)
	if res.Active != nil || res.Upcoming != nil || res.LastEnded != nil {
		t.Fatalf("empty candidates must resolve to nothing: %+v", res)
	}
}

func TestResolveAfterShiftEnded(t *testing.T) {
	in := overnightShift(t)
	res := Resolve(bg(2026, 8, 18, 1, 0), []Instance{in})
	if res.Active != nil {
		t.Fatalf("shift ended 00:30 must not be active at 01:00")
	}
	if res.LastEnded == nil || !res.LastEnded.EndAt.Equal(in.EndAt) {
		t.Fatalf("LastEnded = %+v", res.LastEnded)
	}
}

func TestResolveBeforeShiftStarts(t *testing.T) {
	in := overnightShift(t)
	res := Resolve(bg(2026, 8, 17, 12, 0), []Instance{in})
	if res.Active != nil {
		t.Fatalf("not active before start")
	}
	if res.Upcoming == nil || !res.Upcoming.StartAt.Equal(in.StartAt) {
		t.Fatalf("Upcoming = %+v", res.Upcoming)
	}
}

func TestBusinessDateAroundMidnight(t *testing.T) {
	// 00:10 Baghdad on the 18th is 21:10 UTC on the 17th; the business date
	// must follow Baghdad, not UTC.
	utcInstant := time.Date(2026, 8, 17, 21, 10, 0, 0, time.UTC)
	if got := BusinessDate(utcInstant); !got.Equal(dateUTC(2026, 8, 18)) {
		t.Fatalf("BusinessDate = %v, want 2026-08-18", got)
	}
	// And 23:10 Baghdad on the 17th stays the 17th.
	if got := BusinessDate(bg(2026, 8, 17, 23, 10)); !got.Equal(dateUTC(2026, 8, 17)) {
		t.Fatalf("BusinessDate = %v, want 2026-08-17", got)
	}
	// Year boundary: 00:30 Baghdad on Jan 1 belongs to the new year.
	if got := BusinessDate(bg(2027, 1, 1, 0, 30)); !got.Equal(dateUTC(2027, 1, 1)) {
		t.Fatalf("BusinessDate = %v, want 2027-01-01", got)
	}
}

func TestHourlyLeaveDuration(t *testing.T) {
	cases := []struct {
		start, end string
		want       time.Duration
		wantErr    bool
	}{
		{"09:00", "10:00", time.Hour, false},
		{"23:30", "00:30", time.Hour, false},       // wraps midnight
		{"23:30:00", "00:30:00", time.Hour, false}, // HH:MM:SS from the DB
		{"00:00", "00:30", 30 * time.Minute, false},
		{"22:00", "02:00", 4 * time.Hour, false},
		{"10:00", "10:00", 0, true}, // equal clocks are a data error
		{"aa:bb", "10:00", 0, true},
		{"10:00", "", 0, true},
	}
	for _, c := range cases {
		got, err := HourlyLeaveDuration(c.start, c.end)
		if c.wantErr {
			if err == nil {
				t.Errorf("HourlyLeaveDuration(%q,%q): expected error", c.start, c.end)
			}
			continue
		}
		if err != nil {
			t.Errorf("HourlyLeaveDuration(%q,%q): %v", c.start, c.end, err)
			continue
		}
		if got != c.want {
			t.Errorf("HourlyLeaveDuration(%q,%q) = %v, want %v", c.start, c.end, got, c.want)
		}
	}
}

func TestParseClockAndFormat(t *testing.T) {
	h, m, err := ParseClock("16:30:00")
	if err != nil || h != 16 || m != 30 {
		t.Fatalf("ParseClock = %d:%d, %v", h, m, err)
	}
	if _, _, err := ParseClock("24:00"); err == nil {
		t.Fatalf("24:00 must be rejected")
	}
	if s := FormatDuration(90 * time.Minute); s != "1h 30m" {
		t.Fatalf("FormatDuration = %q", s)
	}
	if s := FormatDuration(2 * time.Hour); s != "2h" {
		t.Fatalf("FormatDuration = %q", s)
	}
	if s := FormatDuration(45 * time.Minute); s != "45m" {
		t.Fatalf("FormatDuration = %q", s)
	}
}

func TestClockStringDiscardsPgxAnchor(t *testing.T) {
	if got := ClockString(clock(16, 30)); got != "16:30" {
		t.Fatalf("ClockString = %q", got)
	}
}

func TestWindowErrorsMentionBothSpans(t *testing.T) {
	in := overnightShift(t)
	_, err := ExplicitWindow(in, "01:00", "02:00")
	if err == nil || !strings.Contains(err.Error(), "outside the shift") {
		t.Fatalf("err = %v", err)
	}
}

func TestFixedClockPinsZone(t *testing.T) {
	c := FixedClock{T: time.Date(2026, 8, 17, 21, 10, 0, 0, time.UTC)}
	now := c.Now()
	if now.Location() != Location() {
		t.Fatalf("clock must yield business-zone instants")
	}
	if now.Hour() != 0 || now.Day() != 18 {
		t.Fatalf("21:10 UTC must be 00:10 Baghdad on the 18th, got %v", now)
	}
}
