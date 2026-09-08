package assistant

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/temporal"
)

// Supervisor reads. Scope resolution is identical to DepartmentContext:
// team leaders see their own department, managers the departments in
// department_managers (plus home), admins any. The department is validated
// server-side on every call — a department ID in the tool input is a request,
// never a grant.

// resolveDeptScope validates a requested department against the actor's
// authority, defaulting sensibly when the input is empty.
func resolveDeptScope(ctx context.Context, d *Deps, actor *Actor, requested string) (*models.Department, error) {
	var deptID uuid.UUID
	switch {
	case requested != "":
		id, err := uuid.Parse(requested)
		if err != nil {
			return nil, Errf("invalid department_id")
		}
		deptID = id
	case actor.DeptID() != nil:
		deptID = *actor.DeptID()
	case actor.Role() == "manager" && len(actor.ManagedDepartments) > 0:
		deptID = actor.ManagedDepartments[0].ID
	default:
		return nil, Errf("no department to inspect: pass department_id")
	}

	if !actor.CanAccessDepartment(deptID) {
		// Same shape as the REST API: no confirmation that the department exists.
		return nil, Errf("department not found or not within your scope")
	}
	dept, err := d.DepartmentRepo.GetByID(ctx, deptID)
	if err != nil {
		return nil, Errf("department not found or not within your scope")
	}
	return dept, nil
}

type memberStatus struct {
	Name       string  `json:"name"`
	Role       string  `json:"role"`
	Status     string  `json:"status"` // working / hourly / off / leave / ...
	Shift      string  `json:"shift,omitempty"`
	StartsAt   string  `json:"starts_at,omitempty"`
	EndsAt     string  `json:"ends_at,omitempty"`
	OnShiftNow bool    `json:"on_shift_now"`
	CheckedIn  *string `json:"checked_in_at,omitempty"`
	CheckedOut *string `json:"checked_out_at,omitempty"`
	LeaveNote  string  `json:"leave_note,omitempty"`
}

func toolGetTeamStatus() Tool {
	return Tool{
		Name: "get_team_status",
		Description: "Operational status of one department for one business date: every active member with their " +
			"shift, whether they are inside their shift window RIGHT NOW (overnight-aware), check-in/out stamps, and who is off or on leave. " +
			"Use it for منو عندي هسه / منو بالشفت / منو موجود / كم واحد موجود / منو بإجازة / وضع التيم, and for 'who is on shift', " +
			"'how many are in', 'who is off today' and 'team status'. " +
			"'Present' means checked in; the system has no other attendance signal. Yesterday's overnight shifts that are still running are included.",
		Roles: []string{"team_leader", "manager", "admin"},
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"department_id":{"type":"string","description":"defaults to your own department; managers may pass a managed department"},
				"date":{"type":"string","description":"business date YYYY-MM-DD; defaults to today"}
			},
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				DepartmentID string `json:"department_id"`
				Date         string `json:"date"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			dept, err := resolveDeptScope(ctx, d, actor, in.DepartmentID)
			if err != nil {
				return nil, err
			}
			now := d.Clock.Now()
			date := temporal.BusinessDate(now)
			if in.Date != "" {
				date, err = parseDateArg(d, in.Date)
				if err != nil {
					return nil, err
				}
			}

			// The date's rows plus the previous day's, because an overnight
			// shift from yesterday is still this morning's staffing.
			rows, err := d.ScheduleService.GetDepartmentShiftsInRange(ctx, date.AddDate(0, 0, -1), date, dept.ID)
			if err != nil {
				return nil, err
			}

			clocks := newShiftClocks(d)
			members := map[uuid.UUID]*memberStatus{}
			counts := map[string]int{}
			onShiftNow := 0
			checkedIn := 0

			for i := range rows {
				row := rows[i]
				isTarget := row.ShiftDate.Equal(date)

				var inst *temporal.Instance
				shiftID := row.ShiftID
				if shiftID == nil {
					shiftID = row.DefaultShiftID
				}
				if onDutyStatus(row.ShiftStatus, row.LeaveReason) && shiftID != nil {
					if sh, err := clocks.get(ctx, *shiftID); err == nil && sh != nil {
						start, end, overnight := temporal.Materialize(row.ShiftDate, sh.StartTime, sh.EndTime)
						inst = &temporal.Instance{
							ShiftName: sh.Name, StartAt: start, EndAt: end, Overnight: overnight,
						}
					}
				}

				activeNow := inst != nil && inst.Contains(now)

				// Yesterday's rows only matter when still running now.
				if !isTarget && !activeNow {
					continue
				}

				// A member may appear twice (yesterday's overnight row and
				// today's row); the target date's row wins for status fields,
				// and yesterday's still-running shift only upgrades the "now"
				// flags below.
				m, exists := members[row.EmployeeID]
				if !exists {
					m = &memberStatus{
						Name: row.FirstName + " " + row.LastName,
						Role: row.EmployeeRole,
					}
					members[row.EmployeeID] = m
				}

				if isTarget || !exists {
					m.Status = row.ShiftStatus
					// Normalise the wire-contract form so the model reads a
					// partial-day زمنية as 'hourly', not as a day on leave.
					if row.ShiftStatus == "leave" && row.LeaveReason != nil && strings.HasPrefix(*row.LeaveReason, models.HourlyLeaveReasonPrefix) {
						m.Status = "hourly"
					}
					if inst != nil {
						m.Shift = inst.ShiftName
						m.StartsAt = inst.StartAt.Format("2006-01-02 15:04")
						m.EndsAt = inst.EndAt.Format("2006-01-02 15:04")
					}
					if row.LeaveReason != nil {
						m.LeaveNote = clipRunes(*row.LeaveReason, 120)
					}
				}
				if activeNow {
					m.OnShiftNow = true
				}
				if row.CheckInTime != nil {
					s := row.CheckInTime.Format("15:04")
					m.CheckedIn = &s
				}
				if row.CheckOutTime != nil {
					s := row.CheckOutTime.Format("15:04")
					m.CheckedOut = &s
				}
			}

			list := make([]memberStatus, 0, len(members))
			for _, m := range members {
				counts[m.Status]++
				if m.OnShiftNow {
					onShiftNow++
					if m.CheckedIn != nil && m.CheckedOut == nil {
						checkedIn++
					}
				}
				list = append(list, *m)
			}
			sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })

			return map[string]any{
				"department":      dept.Name,
				"date":            temporal.DateString(date),
				"now":             now.Format("2006-01-02 15:04"),
				"members":         list,
				"status_counts":   counts,
				"on_shift_now":    onShiftNow,
				"checked_in_now":  checkedIn,
				"attendance_note": "checked_in_at is the only attendance signal this system records; absence of a check-in does not prove absence of the person",
			}, nil
		},
	}
}

func toolGetDepartmentOverview() Tool {
	return Tool{
		Name: "get_department_overview",
		Description: "For managers and admins: a one-line staffing summary of each department in scope for today — " +
			"active headcount, scheduled, off, on leave, and how many are inside a shift window right now.",
		Roles:       []string{"manager", "admin"},
		InputSchema: schema(`{"type":"object","properties":{},"additionalProperties":false}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var scope []models.Department
			if actor.Role() == "admin" {
				all, err := d.DepartmentRepo.GetAll(ctx)
				if err != nil {
					return nil, err
				}
				scope = all
			} else {
				scope = actor.ManagedDepartments
				if actor.DeptID() != nil {
					seen := false
					for _, dept := range scope {
						if dept.ID == *actor.DeptID() {
							seen = true
						}
					}
					if !seen {
						if home, err := d.DepartmentRepo.GetByID(ctx, *actor.DeptID()); err == nil {
							scope = append(scope, *home)
						}
					}
				}
			}
			if len(scope) == 0 {
				return nil, Errf("no departments in your scope")
			}
			if len(scope) > 12 {
				scope = scope[:12]
			}

			now := d.Clock.Now()
			today := temporal.BusinessDate(now)
			clocks := newShiftClocks(d)

			out := []map[string]any{}
			for _, dept := range scope {
				rows, err := d.ScheduleService.GetDepartmentShiftsInRange(ctx, today.AddDate(0, 0, -1), today, dept.ID)
				if err != nil {
					continue
				}
				var working, off, leave, hourly, activeNow int
				seen := map[uuid.UUID]bool{}
				for i := range rows {
					row := rows[i]
					isToday := row.ShiftDate.Equal(today)

					if onDutyStatus(row.ShiftStatus, row.LeaveReason) {
						shiftID := row.ShiftID
						if shiftID == nil {
							shiftID = row.DefaultShiftID
						}
						if shiftID != nil {
							if sh, err := clocks.get(ctx, *shiftID); err == nil && sh != nil {
								start, end, _ := temporal.Materialize(row.ShiftDate, sh.StartTime, sh.EndTime)
								if !now.Before(start) && now.Before(end) && !seen[row.EmployeeID] {
									activeNow++
									seen[row.EmployeeID] = true
								}
							}
						}
					}
					if !isToday {
						continue
					}
					switch {
					case row.ShiftStatus == "working":
						working++
					case row.ShiftStatus == "hourly",
						row.ShiftStatus == "leave" && row.LeaveReason != nil && strings.HasPrefix(*row.LeaveReason, models.HourlyLeaveReasonPrefix):
						hourly++
					case row.ShiftStatus == "off":
						off++
					default:
						leave++
					}
				}
				out = append(out, map[string]any{
					"department":    dept.Name,
					"department_id": dept.ID.String(),
					"working_today": working,
					"hourly_leave":  hourly,
					"off_today":     off,
					"on_leave":      leave,
					"on_shift_now":  activeNow,
				})
			}
			return map[string]any{"date": temporal.DateString(today), "departments": out}, nil
		},
	}
}

func toolGetPendingApprovals() Tool {
	return Tool{
		Name: "get_pending_approvals",
		Description: "Leave requests waiting for the caller's approval (scoped exactly as the Approval Center scopes them), " +
			"plus the count of shift swaps awaiting supervisor action. Use leave_id with propose_leave_decision.",
		Roles:       []string{"team_leader", "manager", "admin"},
		InputSchema: schema(`{"type":"object","properties":{},"additionalProperties":false}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			rich, err := d.LeaveService.GetPendingLeavesRich(ctx, actor.ID())
			if err != nil {
				return nil, err
			}
			leaves := []map[string]any{}
			for i, l := range rich {
				if i >= 15 {
					break
				}
				entry := map[string]any{
					"leave_id":   l.ID.String(),
					"employee":   l.EmployeeName,
					"type_ar":    strOrEmpty(l.LeaveTypeNameAr),
					"type_en":    strOrEmpty(l.LeaveTypeNameEn),
					"start_date": temporal.DateString(l.StartDate),
					"end_date":   temporal.DateString(l.EndDate),
					"days":       l.TotalDays,
					"shift":      l.ShiftName,
				}
				if l.StartTime != nil && l.EndTime != nil {
					entry["start_time"] = clip5(*l.StartTime)
					entry["end_time"] = clip5(*l.EndTime)
				}
				if l.Reason != nil {
					entry["reason"] = clipRunes(*l.Reason, 200)
				}
				leaves = append(leaves, entry)
			}

			swapCount := 0
			if swaps, err := d.SwapService.GetPendingSwapsForManager(ctx, actor.ID()); err == nil {
				swapCount = len(swaps)
			}

			return map[string]any{
				"pending_leaves":      leaves,
				"pending_leave_count": len(rich),
				"pending_swap_count":  swapCount,
			}, nil
		},
	}
}

func toolGetShiftCoverage() Tool {
	return Tool{
		Name: "get_shift_coverage",
		Description: "Staffing counts for one shift type on one date (assigned / working / off / on leave) — " +
			"the same preview supervisors see before approving a leave.",
		Roles: []string{"team_leader", "manager", "admin"},
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"shift_id":{"type":"string"},
				"date":{"type":"string","description":"YYYY-MM-DD"}
			},
			"required":["shift_id","date"],
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				ShiftID string `json:"shift_id"`
				Date    string `json:"date"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			shiftID, err := uuid.Parse(in.ShiftID)
			if err != nil {
				return nil, Errf("invalid shift_id")
			}
			date, err := parseDateArg(d, in.Date)
			if err != nil {
				return nil, err
			}
			// The shift type must belong to a department the actor can access.
			sh, err := d.ShiftRepo.GetByID(ctx, shiftID)
			if err != nil || sh == nil {
				return nil, Errf("shift not found")
			}
			if sh.DepartmentID != nil && !actor.CanAccessDepartment(*sh.DepartmentID) {
				return nil, Errf("shift not found")
			}
			cov, err := d.LeaveService.GetShiftCoveragePreview(ctx, shiftID, date)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"shift":          sh.Name,
				"date":           temporal.DateString(date),
				"total_assigned": cov.TotalAssigned,
				"working":        cov.TotalWorking,
				"off":            cov.TotalOff,
				"on_leave":       cov.TotalOnLeave,
			}, nil
		},
	}
}
