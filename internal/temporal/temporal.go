// Package temporal is the single authority on shift time semantics.
//
// The schema stores a shift assignment as a DATE (employee_shifts.shift_date)
// plus two bare TIME columns on the shift definition (shifts.start_time,
// shifts.end_time). Nothing in the schema says whether a shift crosses
// midnight; that fact exists only as end <= start. This package is the one
// place that composes those pieces into absolute instants, so that "the
// current shift", "the last hour of my shift" and "how long until the end"
// have exactly one implementation.
//
// The business rule this package encodes: a shift belongs to the calendar day
// it STARTS on (its business date). An employee working 16:30→00:30 on the
// 17th is, at 00:10 on the 18th, still inside the 17th's shift. Every
// consumer — the assistant's tools, hourly-leave windows, team status — must
// go through here rather than re-deriving dates, because ad-hoc date
// arithmetic is precisely what gets the after-midnight case wrong.
package temporal

import (
	"fmt"
	"time"
	// The application's business timezone must resolve identically on every
	// host, including minimal containers with no tzdata package installed. A
	// silent fallback to UTC would shift every boundary by three hours.
	_ "time/tzdata"

	"github.com/google/uuid"
)

// zone is Asia/Baghdad: UTC+3 year-round (Iraq abolished DST in 2007). The
// database stores dates and clock times with no zone and the frontend already
// renders in Asia/Baghdad, so this is the zone in which business dates turn
// over.
var zone = mustLoad()

func mustLoad() *time.Location {
	loc, err := time.LoadLocation("Asia/Baghdad")
	if err != nil {
		// Unreachable with the embedded tzdata import above; a panic at init is
		// preferable to silently computing every shift boundary in the wrong zone.
		panic("temporal: load Asia/Baghdad: " + err.Error())
	}
	return loc
}

// Location returns the business timezone.
func Location() *time.Location { return zone }

// BusinessDate reduces an instant to the calendar date it falls on in the
// business timezone, normalised to midnight UTC — the same representation the
// repositories use for DATE columns, so the result can be passed straight into
// queries against shift_date.
func BusinessDate(t time.Time) time.Time {
	y, m, d := t.In(zone).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DateString formats a DATE-column value for humans and tools.
func DateString(date time.Time) string { return date.Format("2006-01-02") }

// ClockString extracts HH:MM from a TIME-column value. pgx anchors TIME at an
// arbitrary date (2000-01-01), which this deliberately discards.
func ClockString(t time.Time) string { return t.Format("15:04") }

// ParseClock accepts "HH:MM" or "HH:MM:SS".
func ParseClock(s string) (hour, minute int, err error) {
	if len(s) > 5 {
		s = s[:5]
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid time %q: expected HH:MM", s)
	}
	return t.Hour(), t.Minute(), nil
}

// Instance is one employee's shift on one business date, materialised to
// absolute instants. EndAt is always strictly after StartAt: when the stored
// end clock is at or before the start clock the shift runs into the next
// calendar day.
type Instance struct {
	EmployeeShiftID uuid.UUID
	EmployeeID      uuid.UUID
	ShiftID         uuid.UUID
	ShiftName       string
	ShiftCode       string

	// BusinessDate is midnight UTC of the shift_date row, matching the DATE
	// column convention.
	BusinessDate time.Time

	StartAt   time.Time // in Location()
	EndAt     time.Time // in Location(); > StartAt
	Overnight bool

	Status     string // employee_shifts.shift_status
	CheckInAt  *time.Time
	CheckOutAt *time.Time
}

// Duration is the scheduled length of the shift.
func (in Instance) Duration() time.Duration { return in.EndAt.Sub(in.StartAt) }

// Contains reports whether now falls inside [StartAt, EndAt).
func (in Instance) Contains(now time.Time) bool {
	return !now.Before(in.StartAt) && now.Before(in.EndAt)
}

// Materialize composes a DATE-column business date with TIME-column clock
// values into concrete instants in the business timezone. An end clock at or
// before the start clock means the shift crosses midnight and ends on the
// following calendar day; there is no other representation of overnight shifts
// in the schema.
func Materialize(businessDate time.Time, startClock, endClock time.Time) (startAt, endAt time.Time, overnight bool) {
	y, m, d := businessDate.Date()
	startAt = time.Date(y, m, d, startClock.Hour(), startClock.Minute(), startClock.Second(), 0, zone)
	endAt = time.Date(y, m, d, endClock.Hour(), endClock.Minute(), endClock.Second(), 0, zone)
	if !endAt.After(startAt) {
		endAt = endAt.Add(24 * time.Hour)
		overnight = true
	}
	return startAt, endAt, overnight
}

// Resolution classifies where "now" stands relative to an employee's recent
// and imminent shift instances.
type Resolution struct {
	// Active is the instance containing now, or nil.
	Active *Instance
	// Upcoming is the next instance starting after now (today or the queried
	// horizon), or nil.
	Upcoming *Instance
	// LastEnded is the most recently finished instance, or nil.
	LastEnded *Instance
}

// Resolve classifies now against a set of candidate instances (typically the
// employee's shifts for yesterday, today and tomorrow in business-date terms).
// Working statuses only; callers filter out days recorded as off or leave
// before building instances.
//
// If overlapping data anomalies produce two simultaneously active instances,
// the one that started most recently wins: it is the shift the employee most
// recently began, and choosing deterministically matters more than which one.
func Resolve(now time.Time, candidates []Instance) Resolution {
	var res Resolution
	for i := range candidates {
		in := candidates[i]
		switch {
		case in.Contains(now):
			if res.Active == nil || in.StartAt.After(res.Active.StartAt) {
				c := in
				res.Active = &c
			}
		case in.StartAt.After(now):
			if res.Upcoming == nil || in.StartAt.Before(res.Upcoming.StartAt) {
				c := in
				res.Upcoming = &c
			}
		default: // ended
			if res.LastEnded == nil || in.EndAt.After(res.LastEnded.EndAt) {
				c := in
				res.LastEnded = &c
			}
		}
	}
	return res
}

// Window is a half-open time span inside a shift, used for hourly leave.
type Window struct {
	Start time.Time
	End   time.Time
}

func (w Window) Duration() time.Duration { return w.End.Sub(w.Start) }

// StartClock and EndClock render the window boundaries as HH:MM in the
// business zone — the representation the leaves table stores. A window whose
// end clock is at or before its start clock crosses midnight; both this
// package's Materialize and the leave duration logic interpret it that way.
func (w Window) StartClock() string { return w.Start.In(zone).Format("15:04") }
func (w Window) EndClock() string   { return w.End.In(zone).Format("15:04") }

// LastWindow returns the final d of the shift: [EndAt-d, EndAt].
func LastWindow(in Instance, d time.Duration) (Window, error) {
	if err := checkWindowDuration(in, d); err != nil {
		return Window{}, err
	}
	return Window{Start: in.EndAt.Add(-d), End: in.EndAt}, nil
}

// FirstWindow returns the opening d of the shift: [StartAt, StartAt+d].
func FirstWindow(in Instance, d time.Duration) (Window, error) {
	if err := checkWindowDuration(in, d); err != nil {
		return Window{}, err
	}
	return Window{Start: in.StartAt, End: in.StartAt.Add(d)}, nil
}

// ExplicitWindow builds a window from clock strings anchored to the shift's
// business date, applying the same crosses-midnight rule as shifts themselves:
// a boundary clock earlier than the shift start belongs to the next calendar
// day. It then verifies the window lies within the shift.
func ExplicitWindow(in Instance, startClock, endClock string) (Window, error) {
	sh, sm, err := ParseClock(startClock)
	if err != nil {
		return Window{}, err
	}
	eh, em, err := ParseClock(endClock)
	if err != nil {
		return Window{}, err
	}

	y, m, d := in.BusinessDate.Date()
	start := time.Date(y, m, d, sh, sm, 0, 0, zone)
	end := time.Date(y, m, d, eh, em, 0, 0, zone)

	// Clocks before the shift's own start clock refer to the portion of an
	// overnight shift that lies past midnight.
	if start.Before(in.StartAt) {
		start = start.Add(24 * time.Hour)
	}
	if !end.After(start) {
		end = end.Add(24 * time.Hour)
	}

	w := Window{Start: start, End: end}
	if w.Start.Before(in.StartAt) || w.End.After(in.EndAt) {
		return Window{}, fmt.Errorf("the requested time %s–%s is outside the shift (%s–%s)",
			w.StartClock(), w.EndClock(), ClockString(in.StartAt), ClockString(in.EndAt))
	}
	return w, nil
}

func checkWindowDuration(in Instance, d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if d > in.Duration() {
		return fmt.Errorf("requested %s but the shift is only %s long",
			FormatDuration(d), FormatDuration(in.Duration()))
	}
	return nil
}

// FormatDuration renders a duration in whole hours and minutes ("1h30m" →
// "1 h 30 min") without seconds noise.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %02dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// HourlyLeaveDuration computes the length of an hourly leave stored as two
// HH:MM clocks under the business-date convention: an end at or before the
// start crosses midnight. This is the same rule Materialize applies to shift
// definitions, extended to the leave rows that shadow them. Equal clocks are
// rejected rather than read as 24 hours — an hourly leave spanning a full day
// is a data error, not a plausible request.
func HourlyLeaveDuration(startClock, endClock string) (time.Duration, error) {
	sh, sm, err := ParseClock(startClock)
	if err != nil {
		return 0, err
	}
	eh, em, err := ParseClock(endClock)
	if err != nil {
		return 0, err
	}
	start := time.Duration(sh)*time.Hour + time.Duration(sm)*time.Minute
	end := time.Duration(eh)*time.Hour + time.Duration(em)*time.Minute
	if end == start {
		return 0, fmt.Errorf("start and end time are the same")
	}
	if end < start {
		end += 24 * time.Hour
	}
	return end - start, nil
}

// Clock abstracts "now" so tests can pin it. The production implementation is
// the system clock; everything downstream must take time from here rather than
// calling time.Now directly, or the overnight tests cannot exist.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production Clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().In(zone) }

// FixedClock pins now for tests.
type FixedClock struct{ T time.Time }

func (f FixedClock) Now() time.Time { return f.T.In(zone) }
