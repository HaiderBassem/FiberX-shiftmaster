package assistant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/temporal"
)

// onDutyStatus reports whether a day row means the employee is expected to
// work that shift. A partial-day (hourly) leave is on duty: it appears either
// as the 'hourly' enum value or — the wire contract every deployed database
// supports — as 'leave' with the "[hourly] " reason prefix
// (models.HourlyLeaveReasonPrefix). Every other non-working status (off,
// leave, vacation, sick, training, business_trip) means the shift is not held
// that day. Without the prefix branch, one approved زمنية made the assistant
// lose the whole day's shift: "what's my shift" skipped it and a second
// hourly-leave request anchored to the wrong (next) shift.
func onDutyStatus(status string, leaveReason *string) bool {
	if status == "working" || status == "hourly" {
		return true
	}
	return status == "leave" && leaveReason != nil && strings.HasPrefix(*leaveReason, models.HourlyLeaveReasonPrefix)
}

// shiftClocks caches shift-type rows for one request.
type shiftClocks struct {
	deps  *Deps
	cache map[uuid.UUID]*models.Shift
}

func newShiftClocks(deps *Deps) *shiftClocks {
	return &shiftClocks{deps: deps, cache: map[uuid.UUID]*models.Shift{}}
}

func (s *shiftClocks) get(ctx context.Context, id uuid.UUID) (*models.Shift, error) {
	if sh, ok := s.cache[id]; ok {
		return sh, nil
	}
	sh, err := s.deps.ShiftRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.cache[id] = sh
	return sh, nil
}

// instancesForRange materialises an employee's shift instances for a range of
// business dates. The read path re-derives generated rows from the weekly
// pattern (the schedule service's EnsureWeekSchedule side effect), so the
// answer reflects the same schedule the UI shows.
//
// The effective shift follows the application's own rule: the day row's shift
// if set, else the employee's default shift.
func (d *Deps) instancesForRange(ctx context.Context, emp *models.Employee, from, to time.Time) ([]temporal.Instance, error) {
	rows, err := d.ScheduleService.GetEmployeeShifts(ctx, emp.ID, from, to)
	if err != nil {
		return nil, fmt.Errorf("read schedule: %w", err)
	}

	clocks := newShiftClocks(d)
	var out []temporal.Instance
	for i := range rows {
		row := rows[i]
		if !onDutyStatus(row.ShiftStatus, row.LeaveReason) {
			continue
		}
		shiftID := row.ShiftID
		if shiftID == nil {
			shiftID = emp.DefaultShiftID
		}
		if shiftID == nil {
			continue // no shift type anywhere: nothing to materialise
		}
		sh, err := clocks.get(ctx, *shiftID)
		if err != nil || sh == nil {
			continue
		}
		start, end, overnight := temporal.Materialize(row.ShiftDate, sh.StartTime, sh.EndTime)
		out = append(out, temporal.Instance{
			EmployeeShiftID: row.ID,
			EmployeeID:      row.EmployeeID,
			ShiftID:         sh.ID,
			ShiftName:       sh.Name,
			ShiftCode:       sh.ShiftCode,
			BusinessDate:    row.ShiftDate,
			StartAt:         start,
			EndAt:           end,
			Overnight:       overnight,
			Status:          row.ShiftStatus,
			CheckInAt:       row.CheckInTime,
			CheckOutAt:      row.CheckOutTime,
		})
	}
	return out, nil
}

// resolveCurrent answers "which shift instance is the employee inside right
// now" — the question no ad-hoc date arithmetic gets right after midnight. It
// considers yesterday (a still-running overnight shift), today, and tomorrow
// (so an off-duty ask can still report the next shift).
func (d *Deps) resolveCurrent(ctx context.Context, emp *models.Employee) (temporal.Resolution, error) {
	now := d.Clock.Now()
	today := temporal.BusinessDate(now)
	instances, err := d.instancesForRange(ctx, emp, today.AddDate(0, 0, -1), today.AddDate(0, 0, 1))
	if err != nil {
		return temporal.Resolution{}, err
	}
	return temporal.Resolve(now, instances), nil
}

// instanceForBusinessDate materialises exactly one date's instance (nil when
// the employee is off / on leave / unscheduled that day).
func (d *Deps) instanceForBusinessDate(ctx context.Context, emp *models.Employee, businessDate time.Time) (*temporal.Instance, error) {
	instances, err := d.instancesForRange(ctx, emp, businessDate, businessDate)
	if err != nil {
		return nil, err
	}
	for i := range instances {
		if instances[i].BusinessDate.Equal(businessDate) {
			return &instances[i], nil
		}
	}
	return nil, nil
}
