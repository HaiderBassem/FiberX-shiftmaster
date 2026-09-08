package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/temporal"
)

// Grounding, identity and operational reads.
//
// The first two tools here exist because of a rule the rest of the system
// depends on: the model interprets language, the backend resolves time. A
// language model asked to work out "the day after tomorrow" will usually get
// it right and occasionally get it wrong, and there is no way to tell which
// from the answer. Giving it resolve_date removes the question entirely — it
// names the expression, the server returns the date.

func toolResolveDate() Tool {
	return Tool{
		Name: "resolve_date",
		Description: "Authoritative clock and calendar. With no arguments it returns the current time, today's business date and weekday. " +
			"With an expression it returns the business date that expression means, in Asia/Baghdad. " +
			"Use this instead of working dates out yourself — for 'باجر', 'بعد بكرة', 'يوم الخميس الجاي', 'last Sunday' and anything else relative. " +
			"Feed the returned date to get_my_schedule, get_team_status or a propose_* tool. " +
			"Note that a shift belongs to the day it STARTS, so after midnight today's business date and the shift you are working can differ.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"expression":{
					"type":"string",
					"enum":["now","today","tomorrow","yesterday","day_after_tomorrow","day_before_yesterday",
					        "next_saturday","next_sunday","next_monday","next_tuesday","next_wednesday","next_thursday","next_friday",
					        "last_saturday","last_sunday","last_monday","last_tuesday","last_wednesday","last_thursday","last_friday",
					        "start_of_this_week","end_of_this_week","start_of_next_week","start_of_this_month"],
					"description":"which date to resolve; omit for the current date and time"
				},
				"offset_days":{"type":"integer","minimum":-90,"maximum":90,"description":"days to add to the resolved date, e.g. expression 'today' with offset 3"}
			},
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				Expression string `json:"expression"`
				OffsetDays int    `json:"offset_days"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			now := d.Clock.Now()
			today := temporal.BusinessDate(now)

			date := today
			switch in.Expression {
			case "", "now", "today":
			case "tomorrow":
				date = today.AddDate(0, 0, 1)
			case "yesterday":
				date = today.AddDate(0, 0, -1)
			case "day_after_tomorrow":
				date = today.AddDate(0, 0, 2)
			case "day_before_yesterday":
				date = today.AddDate(0, 0, -2)
			case "start_of_this_week":
				date = today.AddDate(0, 0, -int(today.Weekday())) // weeks start Sunday here
			case "end_of_this_week":
				date = today.AddDate(0, 0, 6-int(today.Weekday()))
			case "start_of_next_week":
				date = today.AddDate(0, 0, 7-int(today.Weekday()))
			case "start_of_this_month":
				y, m, _ := today.Date()
				date = time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
			default:
				var err error
				date, err = resolveWeekday(today, in.Expression)
				if err != nil {
					return nil, err
				}
			}
			if in.OffsetDays != 0 {
				date = date.AddDate(0, 0, in.OffsetDays)
			}

			return map[string]any{
				"now":                 now.Format("2006-01-02 15:04"),
				"now_time":            now.Format("15:04"),
				"today_business_date": temporal.DateString(today),
				"today_weekday":       today.Weekday().String(),
				"resolved_date":       temporal.DateString(date),
				"resolved_weekday":    date.Weekday().String(),
				"days_from_today":     int(date.Sub(today).Hours() / 24),
				"timezone":            "Asia/Baghdad",
				"business_date_note":  "a shift belongs to the calendar day it starts on; use get_current_shift for 'my shift now'",
			}, nil
		},
	}
}

// resolveWeekday handles next_/last_<weekday>. "Next Thursday" means the
// coming Thursday, and when today IS Thursday it means the one in seven days —
// the reading people intend when they are asking about a schedule.
func resolveWeekday(today time.Time, expr string) (time.Time, error) {
	names := map[string]time.Weekday{
		"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
		"wednesday": time.Wednesday, "thursday": time.Thursday,
		"friday": time.Friday, "saturday": time.Saturday,
	}
	forward := strings.HasPrefix(expr, "next_")
	backward := strings.HasPrefix(expr, "last_")
	if !forward && !backward {
		return time.Time{}, Errf("unknown date expression %q", expr)
	}
	want, ok := names[strings.TrimPrefix(strings.TrimPrefix(expr, "next_"), "last_")]
	if !ok {
		return time.Time{}, Errf("unknown date expression %q", expr)
	}
	diff := int(want-today.Weekday()+7) % 7
	if forward {
		if diff == 0 {
			diff = 7
		}
		return today.AddDate(0, 0, diff), nil
	}
	back := int(today.Weekday()-want+7) % 7
	if back == 0 {
		back = 7
	}
	return today.AddDate(0, 0, -back), nil
}

func toolGetMyProfile() Tool {
	return Tool{
		Name: "get_my_profile",
		Description: "Who the caller is according to the database right now: name, job title, role, department, " +
			"their department manager and team leader, their default shift, and their employment dates. " +
			"Use it when the person asks about themselves, or when you need their department or default shift to interpret a request. " +
			"There is no tool that returns anyone else's profile.",
		InputSchema: schema(`{"type":"object","properties":{},"additionalProperties":false}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			if err := decode(input, &struct{}{}); err != nil {
				return nil, err
			}
			emp := actor.Employee
			out := map[string]any{
				"name":          strings.TrimSpace(emp.FirstName + " " + emp.LastName),
				"employee_code": emp.EmployeeCode,
				"role":          emp.Role,
				"status":        emp.Status,
			}
			if emp.Position != nil {
				out["job_title"] = *emp.Position
			}
			if !emp.HireDate.IsZero() {
				out["hire_date"] = temporal.DateString(temporal.BusinessDate(emp.HireDate))
			}
			if emp.DepartmentID != nil {
				if dept, err := d.DepartmentRepo.GetByID(ctx, *emp.DepartmentID); err == nil && dept != nil {
					out["department"] = dept.Name
					out["department_id"] = dept.ID.String()
					// Who to escalate to is a routine, useful question.
					if members, err := d.EmployeeRepo.GetByDepartment(ctx, dept.ID); err == nil {
						var leaders, managers []string
						for _, m := range members {
							if m.Status != "active" {
								continue
							}
							switch m.Role {
							case "team_leader":
								leaders = append(leaders, strings.TrimSpace(m.FirstName+" "+m.LastName))
							case "manager":
								managers = append(managers, strings.TrimSpace(m.FirstName+" "+m.LastName))
							}
						}
						if len(leaders) > 0 {
							out["team_leaders"] = leaders
						}
						if len(managers) > 0 {
							out["managers"] = managers
						}
					}
				}
			}
			if emp.DefaultShiftID != nil {
				if sh, err := d.ShiftRepo.GetByID(ctx, *emp.DefaultShiftID); err == nil && sh != nil {
					out["default_shift"] = map[string]any{
						"name":       sh.Name,
						"code":       sh.ShiftCode,
						"start_time": temporal.ClockString(sh.StartTime),
						"end_time":   temporal.ClockString(sh.EndTime),
					}
				}
			}
			if len(actor.ManagedDepartments) > 0 {
				var names []string
				for _, m := range actor.ManagedDepartments {
					names = append(names, m.Name)
				}
				out["manages_departments"] = names
			}
			return out, nil
		},
	}
}

func toolGetTeamMembers() Tool {
	return Tool{
		Name: "get_team_members",
		Description: "The active people in one department: name, job title, role and their default shift. " +
			"Everyone may look up their OWN department's roster — this is the internal directory, not personal data. " +
			"Managers may pass a department they manage. Use it to answer 'who is in my team', to find a colleague's name, " +
			"or to pick a colleague for a shift swap. For who is actually working right now, use get_team_status instead.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{"department_id":{"type":"string","description":"defaults to your own department"}},
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				DepartmentID string `json:"department_id"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			dept, err := resolveDeptScope(ctx, d, actor, in.DepartmentID)
			if err != nil {
				return nil, err
			}
			members, err := d.EmployeeRepo.GetByDepartment(ctx, dept.ID)
			if err != nil {
				return nil, err
			}
			shiftName := map[uuid.UUID]string{}
			out := []map[string]any{}
			for i := range members {
				m := members[i]
				if m.Status != "active" {
					continue
				}
				entry := map[string]any{
					"employee_id": m.ID.String(),
					"name":        strings.TrimSpace(m.FirstName + " " + m.LastName),
					"role":        m.Role,
				}
				if m.Position != nil {
					entry["job_title"] = *m.Position
				}
				if m.DefaultShiftID != nil {
					name, ok := shiftName[*m.DefaultShiftID]
					if !ok {
						if sh, err := d.ShiftRepo.GetByID(ctx, *m.DefaultShiftID); err == nil && sh != nil {
							name = sh.Name
							shiftName[*m.DefaultShiftID] = name
						}
					}
					if name != "" {
						entry["default_shift"] = name
					}
				}
				out = append(out, entry)
				if len(out) >= 60 {
					break
				}
			}
			return map[string]any{"department": dept.Name, "members": out, "count": len(out)}, nil
		},
	}
}

func toolGetHandovers() Tool {
	return Tool{
		Name: "get_handovers",
		Description: "Shift handovers for the caller's department, newest first: what the outgoing shift summarised, " +
			"what issues they left pending, and whether someone has claimed or completed it. " +
			"Statuses are open (nobody picked it up), claimed (someone is on it) and completed. " +
			"Use it for 'شنو صار بالشفت اللي قبلي', 'اكو شي معلق', or before starting a shift.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"status":{"type":"string","enum":["open","claimed","completed"],"description":"omit for all"},
				"limit":{"type":"integer","minimum":1,"maximum":15}
			},
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				Status string `json:"status"`
				Limit  int    `json:"limit"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			if actor.DeptID() == nil {
				return nil, Errf("you are not assigned to a department, so there are no handovers to show")
			}
			rows, err := d.HandoverRepo.GetByDepartment(ctx, *actor.DeptID())
			if err != nil {
				return nil, err
			}
			limit := in.Limit
			if limit <= 0 {
				limit = 6
			}
			out := []map[string]any{}
			for i := range rows {
				h := rows[i]
				if in.Status != "" && h.Status != in.Status {
					continue
				}
				entry := map[string]any{
					"handover_id":    h.ID.String(),
					"created_at":     h.CreatedAt.In(temporal.Location()).Format("2006-01-02 15:04"),
					"status":         h.Status,
					"shift_summary":  clipRunes(h.ShiftSummary, 600),
					"pending_issues": clipRunes(h.PendingIssues, 600),
				}
				if h.CreatorName != nil {
					entry["from"] = *h.CreatorName
				}
				if h.ClaimerName != nil {
					entry["claimed_by"] = *h.ClaimerName
				}
				if h.DoneByName != nil {
					entry["completed_by"] = *h.DoneByName
				}
				out = append(out, entry)
				if len(out) >= limit {
					break
				}
			}
			return map[string]any{"handovers": out, "count": len(out)}, nil
		},
	}
}

func toolGetTickets() Tool {
	return Tool{
		Name: "get_tickets",
		Description: "Cross-department tickets involving the caller's department, newest first — both the ones sent TO it " +
			"(direction 'incoming', i.e. work it owes another department) and the ones it raised (direction 'outgoing'). " +
			"Use it for 'اكو تكتات جديدة', 'شنو طالبين منا', or to check what happened to a ticket. " +
			"To open a new one use propose_create_ticket.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"direction":{"type":"string","enum":["incoming","outgoing","all"]},
				"status":{"type":"string","enum":["open","closed"]},
				"limit":{"type":"integer","minimum":1,"maximum":15}
			},
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				Direction string `json:"direction"`
				Status    string `json:"status"`
				Limit     int    `json:"limit"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			if actor.DeptID() == nil {
				return nil, Errf("you are not assigned to a department, so there are no tickets to show")
			}
			mine := *actor.DeptID()
			rows, err := d.TicketRepo.GetTicketsForDepartment(ctx, mine)
			if err != nil {
				return nil, err
			}
			limit := in.Limit
			if limit <= 0 {
				limit = 8
			}
			out := []map[string]any{}
			for i := range rows {
				t := rows[i]
				direction := "outgoing"
				if t.TargetDepartmentID == mine {
					direction = "incoming"
				}
				if in.Direction != "" && in.Direction != "all" && in.Direction != direction {
					continue
				}
				if in.Status != "" && t.Status != in.Status {
					continue
				}
				entry := map[string]any{
					"ticket_id":   t.ID.String(),
					"direction":   direction,
					"title":       clipRunes(t.Title, 200),
					"description": clipRunes(t.Description, 500),
					"status":      t.Status,
					"created_at":  t.CreatedAt.In(temporal.Location()).Format("2006-01-02 15:04"),
				}
				if t.SourceDepartment != nil {
					entry["from_department"] = *t.SourceDepartment
				}
				if t.TargetDepartment != nil {
					entry["to_department"] = *t.TargetDepartment
				}
				if t.CreatorName != nil {
					entry["opened_by"] = *t.CreatorName
				}
				out = append(out, entry)
				if len(out) >= limit {
					break
				}
			}
			return map[string]any{"tickets": out, "count": len(out)}, nil
		},
	}
}

func toolGetItemCategories() Tool {
	return Tool{
		Name: "get_item_request_categories",
		Description: "The kinds of equipment or supplies the caller's department lets its people request (router, SIM, cable, and so on). " +
			"Each has a category_id to pass to propose_item_request. Use it when someone says they need a thing — " +
			"'اريد راوتر', 'I need a new headset' — so the request is filed under a real category.",
		InputSchema: schema(`{"type":"object","properties":{},"additionalProperties":false}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			if err := decode(input, &struct{}{}); err != nil {
				return nil, err
			}
			if actor.DeptID() == nil {
				return nil, Errf("you are not assigned to a department, so there is nothing to request from")
			}
			cats, err := d.ItemReqService.GetCategoriesByDepartment(ctx, *actor.DeptID())
			if err != nil {
				return nil, err
			}
			out := []map[string]any{}
			for _, c := range cats {
				out = append(out, map[string]any{"category_id": c.ID.String(), "name": c.Name})
			}
			return map[string]any{"categories": out}, nil
		},
	}
}

func toolCheckLeaveEligibility() Tool {
	return Tool{
		Name: "check_leave_eligibility",
		Description: "Dry-run a leave request WITHOUT staging anything: it runs the same balance, overlap, coverage and " +
			"notice-period rules a real request goes through and reports whether it would be accepted, and if not, exactly why. " +
			"Use it when someone asks 'أگدر آخذ إجازة يوم الخميس؟' or 'do I have enough balance', so you can answer before " +
			"proposing anything. It changes nothing; propose_leave_request is still required to actually stage the request.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"leave_type_id":{"type":"string"},
				"start_date":{"type":"string","description":"YYYY-MM-DD"},
				"end_date":{"type":"string","description":"YYYY-MM-DD, same as start_date for one day"}
			},
			"required":["leave_type_id","start_date","end_date"],
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				LeaveTypeID string `json:"leave_type_id"`
				StartDate   string `json:"start_date"`
				EndDate     string `json:"end_date"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			params := leaveRequestParams{LeaveTypeID: in.LeaveTypeID, StartDate: in.StartDate, EndDate: in.EndDate}
			leave, err := params.buildLeave(actor)
			if err != nil {
				return nil, err
			}
			// The same validator the real request path runs. Nothing is written.
			if err := d.LeaveService.ValidateLeaveRequest(ctx, leave); err != nil {
				return map[string]any{
					"eligible": false,
					"reason":   asUserMessage("check_leave_eligibility", err),
				}, nil
			}
			return map[string]any{
				"eligible": true,
				"note":     "the rules pass right now; it still has to be staged with propose_leave_request and approved by the user, and then by their supervisor",
			}, nil
		},
	}
}

func toolGetSwapCandidates() Tool {
	return Tool{
		Name: "get_swap_candidates",
		Description: "Colleagues who could take over the caller's shift on a given date, according to the swap rules " +
			"(same department, qualified, not already committed). Each has an employee_id for propose_swap_request. " +
			"Call this before proposing a swap — the swap will be refused for anyone not on this list.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{"date":{"type":"string","description":"business date of the shift to give away, YYYY-MM-DD"}},
			"required":["date"],
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				Date string `json:"date"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			date, err := parseDateArg(d, in.Date)
			if err != nil {
				return nil, err
			}
			instance, err := d.instanceForBusinessDate(ctx, actor.Employee, date)
			if err != nil {
				return nil, err
			}
			if instance == nil {
				return map[string]any{
					"date":       temporal.DateString(date),
					"candidates": []any{},
					"note":       "you are not scheduled to work that day, so there is no shift to swap",
				}, nil
			}
			candidates, err := d.SwapService.GetEligibleShiftSwapTargets(ctx, actor.ID(), date)
			if err != nil {
				return nil, err
			}
			out := []map[string]any{}
			for i := range candidates {
				c := candidates[i]
				out = append(out, map[string]any{
					"employee_id":     c.ID.String(),
					"name":            strings.TrimSpace(c.FirstName + " " + c.LastName),
					"is_off_that_day": c.IsOff,
				})
				if len(out) >= 30 {
					break
				}
			}
			return map[string]any{
				"date":       temporal.DateString(date),
				"your_shift": fmt.Sprintf("%s %s–%s", instance.ShiftName, temporal.ClockString(instance.StartAt), temporal.ClockString(instance.EndAt)),
				"candidates": out,
			}, nil
		},
	}
}
