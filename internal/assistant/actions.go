package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/temporal"
)

// The approval boundary.
//
// A propose_* tool VALIDATES a request and freezes it as a pending action:
// exact parameters, human-readable summary, short expiry. Nothing has changed
// in the domain at that point. The human sees the summary and decides through
// a separate authenticated endpoint; approval atomically claims the action
// (compare-and-set on status, single winner across replicas and double
// clicks) and only then runs the executor — which re-validates against live
// state before mutating, because the world may have moved since the preview.
// The model cannot approve: there is no tool for it, and the approval endpoint
// requires the human's own JWT.

// Action type identifiers, stored on the pending action row.
const (
	ActionLeaveRequest  = "leave.request"
	ActionHourlyLeave   = "leave.request_hourly"
	ActionCancelLeave   = "leave.cancel"
	ActionLeaveDecision = "leave.decision"
	ActionTaskStart     = "task.start"
	ActionTaskComplete  = "task.complete"
	ActionCheckIn       = "shift.check_in"
	ActionCheckOut      = "shift.check_out"
	ActionCreateTicket  = "ticket.create"
)

// summaryField is one line of the approval card, pre-localised on the server
// so the card never contains model-authored values.
type summaryField struct {
	LabelAr string `json:"label_ar"`
	LabelEn string `json:"label_en"`
	Value   string `json:"value"`
}

type actionSummary struct {
	TitleAr string         `json:"title_ar"`
	TitleEn string         `json:"title_en"`
	Fields  []summaryField `json:"fields"`
}

// actionDef couples validation with execution for one action type. validate
// runs at propose time and returns the frozen params plus the card summary;
// execute runs after human approval, against the FROZEN params only, and must
// re-validate anything that could have changed.
type actionDef struct {
	validate func(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error)
	execute  func(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error)
}

func actionDefs() map[string]actionDef {
	return map[string]actionDef{
		ActionLeaveRequest:  {validateLeaveRequest, executeLeaveRequest},
		ActionHourlyLeave:   {validateHourlyLeave, executeLeaveRequest},
		ActionCancelLeave:   {validateCancelLeave, executeCancelLeave},
		ActionLeaveDecision: {validateLeaveDecision, executeLeaveDecision},
		ActionTaskStart:     {validateTaskAction(ActionTaskStart), executeTaskStart},
		ActionTaskComplete:  {validateTaskAction(ActionTaskComplete), executeTaskComplete},
		ActionCheckIn:       {validateCheck(true), executeCheckIn},
		ActionCheckOut:      {validateCheck(false), executeCheckOut},
		ActionCreateTicket:  {validateCreateTicket, executeCreateTicket},
	}
}

// propose validates, freezes and stores a pending action, superseding any
// other pending card in the conversation so exactly one decision is live.
func propose(ctx context.Context, d *Deps, actor *Actor, conversationID *uuid.UUID, actionType string, raw json.RawMessage) (any, error) {
	def, ok := actionDefs()[actionType]
	if !ok {
		return nil, fmt.Errorf("unknown action type %s", actionType)
	}
	frozen, summary, err := def.validate(ctx, d, actor, raw)
	if err != nil {
		return nil, err
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return nil, err
	}

	action := &models.AssistantPendingAction{
		EmployeeID:     actor.ID(),
		ConversationID: conversationID,
		ActionType:     actionType,
		Params:         frozen,
		Summary:        summaryJSON,
		ExpiresAt:      time.Now().Add(d.Cfg.PendingActionTTL),
	}
	if err := d.AssistantRepo.CreatePendingAction(ctx, action); err != nil {
		return nil, err
	}
	if conversationID != nil {
		_ = d.AssistantRepo.SupersedePending(ctx, *conversationID, actor.ID(), action.ID)
	}

	return map[string]any{
		"pending_action": map[string]any{
			"action_id":  action.ID.String(),
			"type":       actionType,
			"summary":    json.RawMessage(summaryJSON),
			"expires_at": action.ExpiresAt.In(temporal.Location()).Format("15:04"),
		},
		"note": "staged only — nothing has been submitted; the user must press Approve on the card",
	}, nil
}

// ─── leave.request (whole days) ─────────────────────────────────────────────

type leaveRequestParams struct {
	LeaveTypeID string `json:"leave_type_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	StartTime   string `json:"start_time,omitempty"`
	EndTime     string `json:"end_time,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// buildLeave converts frozen params into the domain object RequestLeave
// expects, always for the acting employee.
func (p leaveRequestParams) buildLeave(actor *Actor) (*models.Leave, error) {
	typeID, err := uuid.Parse(p.LeaveTypeID)
	if err != nil {
		return nil, Errf("invalid leave_type_id")
	}
	start, err := time.Parse("2006-01-02", p.StartDate)
	if err != nil {
		return nil, Errf("invalid start_date")
	}
	end, err := time.Parse("2006-01-02", p.EndDate)
	if err != nil {
		return nil, Errf("invalid end_date")
	}
	leave := &models.Leave{
		EmployeeID:  actor.ID(),
		LeaveTypeID: typeID,
		StartDate:   start,
		EndDate:     end,
	}
	if p.Reason != "" {
		r := clipRunes(p.Reason, 500)
		leave.Reason = &r
	}
	if p.StartTime != "" && p.EndTime != "" {
		st, et := p.StartTime, p.EndTime
		leave.StartTime = &st
		leave.EndTime = &et
	}
	return leave, nil
}

func leaveTypeByID(ctx context.Context, d *Deps, id string) (*models.LeaveType, error) {
	typeID, err := uuid.Parse(id)
	if err != nil {
		return nil, Errf("invalid leave_type_id — call get_leave_types first")
	}
	lt, err := d.LeaveTypeRepo.GetByID(ctx, typeID)
	if err != nil || lt == nil || !lt.IsActive {
		return nil, Errf("leave type not found or inactive — call get_leave_types")
	}
	return lt, nil
}

func validateLeaveRequest(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	var p leaveRequestParams
	if err := decode(raw, &p); err != nil {
		return nil, nil, err
	}
	lt, err := leaveTypeByID(ctx, d, p.LeaveTypeID)
	if err != nil {
		return nil, nil, err
	}
	if lt.Unit == "hours" {
		return nil, nil, Errf("%s is an hourly type — use propose_hourly_leave", lt.NameEn)
	}
	p.StartTime, p.EndTime = "", ""

	leave, err := p.buildLeave(actor)
	if err != nil {
		return nil, nil, err
	}
	if err := d.LeaveService.ValidateLeaveRequest(ctx, leave); err != nil {
		return nil, nil, Errf("%s", err.Error())
	}

	days := int(leave.EndDate.Sub(leave.StartDate).Hours()/24) + 1
	frozen, _ := json.Marshal(p)
	return frozen, &actionSummary{
		TitleAr: "طلب إجازة",
		TitleEn: "Leave request",
		Fields: []summaryField{
			{"النوع", "Type", lt.NameAr + " / " + lt.NameEn},
			{"من", "From", p.StartDate},
			{"إلى", "To", p.EndDate},
			{"المدة", "Duration", fmt.Sprintf("%d day(s)", days)},
			{"السبب", "Reason", orDash(p.Reason)},
		},
	}, nil
}

func executeLeaveRequest(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p leaveRequestParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}

	// For hourly windows staged against a specific shift, verify the shift
	// still holds the window: schedules can change between preview and
	// approval, and an approved-but-now-outside-the-shift window must not be
	// created silently.
	if p.StartTime != "" && p.EndTime != "" {
		date, err := time.Parse("2006-01-02", p.StartDate)
		if err != nil {
			return nil, Errf("invalid start_date")
		}
		inst, err := d.instanceForBusinessDate(ctx, actor.Employee, date)
		if err != nil {
			return nil, err
		}
		if inst == nil {
			return nil, Errf("your shift on %s changed and no longer exists — ask again to rebuild the request", p.StartDate)
		}
		if _, err := temporal.ExplicitWindow(*inst, p.StartTime, p.EndTime); err != nil {
			return nil, Errf("your shift on %s changed (%s–%s) and the approved window no longer fits — ask again to rebuild the request",
				p.StartDate, temporal.ClockString(inst.StartAt), temporal.ClockString(inst.EndAt))
		}
	}

	leave, err := p.buildLeave(actor)
	if err != nil {
		return nil, err
	}
	// RequestLeave re-runs the full validation set (balance, caps, overlap,
	// past-date) against live state before inserting.
	if err := d.LeaveService.RequestLeave(ctx, leave); err != nil {
		return nil, Errf("%s", err.Error())
	}
	return map[string]any{
		"leave_id": leave.ID.String(),
		"status":   leave.Status,
	}, nil
}

// ─── leave.request_hourly ───────────────────────────────────────────────────

// hourlyLeaveInput is what the model supplies; the window may be relative
// ("the last N minutes of the shift") and is resolved server-side against the
// materialised shift instance — the model never does midnight arithmetic.
type hourlyLeaveInput struct {
	LeaveTypeID string `json:"leave_type_id"`
	Date        string `json:"date"`
	Anchor      string `json:"anchor"`
	Minutes     int    `json:"minutes"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Reason      string `json:"reason"`
}

func validateHourlyLeave(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	var in hourlyLeaveInput
	if err := decode(raw, &in); err != nil {
		return nil, nil, err
	}

	// Resolve the leave type; when omitted and exactly one active hourly type
	// exists, use it — the common "أريد زمنية" case.
	var lt *models.LeaveType
	if in.LeaveTypeID != "" {
		var err error
		lt, err = leaveTypeByID(ctx, d, in.LeaveTypeID)
		if err != nil {
			return nil, nil, err
		}
	} else {
		types, err := d.LeaveTypeRepo.GetActive(ctx)
		if err != nil {
			return nil, nil, err
		}
		var hourly []models.LeaveType
		for _, t := range types {
			if t.Unit == "hours" {
				hourly = append(hourly, t)
			}
		}
		switch len(hourly) {
		case 0:
			return nil, nil, Errf("no hourly leave type is configured")
		case 1:
			lt = &hourly[0]
		default:
			var names []string
			for _, t := range hourly {
				names = append(names, t.NameAr+" ("+t.ID.String()+")")
			}
			return nil, nil, Errf("multiple hourly types exist — ask the user which one and pass leave_type_id: %s", strings.Join(names, ", "))
		}
	}
	if lt.Unit != "hours" {
		return nil, nil, Errf("%s is not an hourly leave type — use propose_leave_request for whole days", lt.NameEn)
	}

	// Resolve the shift instance the window is anchored to.
	var inst *temporal.Instance
	if in.Date == "" {
		res, err := d.resolveCurrent(ctx, actor.Employee)
		if err != nil {
			return nil, nil, err
		}
		switch {
		case res.Active != nil:
			inst = res.Active
		case res.Upcoming != nil:
			inst = res.Upcoming
		default:
			return nil, nil, Errf("you have no active or upcoming shift — pass a date, or the request cannot be anchored to a shift")
		}
	} else {
		date, err := parseDateArg(d, in.Date)
		if err != nil {
			return nil, nil, err
		}
		inst, err = d.instanceForBusinessDate(ctx, actor.Employee, date)
		if err != nil {
			return nil, nil, err
		}
		if inst == nil {
			return nil, nil, Errf("no working shift on %s (day off or leave) — an hourly leave must sit inside a shift", in.Date)
		}
	}

	// Compute the window.
	var window temporal.Window
	var err error
	switch in.Anchor {
	case "shift_end", "":
		if in.Minutes <= 0 && in.StartTime == "" {
			return nil, nil, Errf("pass minutes (e.g. 60 for the last hour) or an explicit start_time/end_time")
		}
		if in.Minutes > 0 {
			window, err = temporal.LastWindow(*inst, time.Duration(in.Minutes)*time.Minute)
		} else {
			window, err = temporal.ExplicitWindow(*inst, in.StartTime, in.EndTime)
		}
	case "shift_start":
		if in.Minutes <= 0 {
			return nil, nil, Errf("pass minutes for a shift_start window")
		}
		window, err = temporal.FirstWindow(*inst, time.Duration(in.Minutes)*time.Minute)
	case "explicit":
		if in.StartTime == "" || in.EndTime == "" {
			return nil, nil, Errf("explicit anchor requires start_time and end_time")
		}
		window, err = temporal.ExplicitWindow(*inst, in.StartTime, in.EndTime)
	default:
		return nil, nil, Errf("anchor must be shift_end, shift_start or explicit")
	}
	if err != nil {
		return nil, nil, Errf("%s", err.Error())
	}

	p := leaveRequestParams{
		LeaveTypeID: lt.ID.String(),
		StartDate:   temporal.DateString(inst.BusinessDate),
		EndDate:     temporal.DateString(inst.BusinessDate),
		StartTime:   window.StartClock(),
		EndTime:     window.EndClock(),
		Reason:      clipRunes(in.Reason, 500),
	}
	leave, err := p.buildLeave(actor)
	if err != nil {
		return nil, nil, err
	}
	if err := d.LeaveService.ValidateLeaveRequest(ctx, leave); err != nil {
		return nil, nil, Errf("%s", err.Error())
	}

	frozen, _ := json.Marshal(p)
	windowLabel := window.Start.Format("15:04") + " → " + window.End.Format("15:04")
	if window.End.Day() != window.Start.Day() {
		windowLabel = window.Start.Format("15:04") + " → " + window.End.Format("15:04") + " (" + window.End.Format("Jan 2") + ")"
	}
	return frozen, &actionSummary{
		TitleAr: "طلب زمنية",
		TitleEn: "Hourly leave request",
		Fields: []summaryField{
			{"النوع", "Type", lt.NameAr + " / " + lt.NameEn},
			{"الشفت", "Shift", inst.ShiftName + " (" + temporal.ClockString(inst.StartAt) + "–" + temporal.ClockString(inst.EndAt) + ")"},
			{"التاريخ", "Date", temporal.DateString(inst.BusinessDate)},
			{"الوقت", "Time", windowLabel},
			{"المدة", "Duration", temporal.FormatDuration(window.Duration())},
			{"السبب", "Reason", orDash(in.Reason)},
		},
	}, nil
}

// ─── leave.cancel ───────────────────────────────────────────────────────────

type cancelLeaveParams struct {
	LeaveID string `json:"leave_id"`
}

func validateCancelLeave(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	var p cancelLeaveParams
	if err := decode(raw, &p); err != nil {
		return nil, nil, err
	}
	leaveID, err := uuid.Parse(p.LeaveID)
	if err != nil {
		return nil, nil, Errf("invalid leave_id — use get_my_leave_requests to find it")
	}
	leave, err := d.LeaveRepo.GetByID(ctx, leaveID)
	if err != nil || leave == nil || leave.EmployeeID != actor.ID() {
		// Not distinguishing "someone else's" from "nonexistent".
		return nil, nil, Errf("leave request not found among your requests")
	}
	if leave.Status != "pending" && leave.Status != "approved_by_team_leader" {
		return nil, nil, Errf("this request is %s and can no longer be cancelled by you", leave.Status)
	}

	frozen, _ := json.Marshal(p)
	fields := []summaryField{
		{"النوع", "Type", strOrEmpty(leave.LeaveTypeNameAr) + " / " + strOrEmpty(leave.LeaveTypeNameEn)},
		{"من", "From", temporal.DateString(leave.StartDate)},
		{"إلى", "To", temporal.DateString(leave.EndDate)},
		{"الحالة", "Status", leave.Status},
	}
	if leave.StartTime != nil && leave.EndTime != nil {
		fields = append(fields, summaryField{"الوقت", "Time", clip5(*leave.StartTime) + " → " + clip5(*leave.EndTime)})
	}
	return frozen, &actionSummary{
		TitleAr: "إلغاء طلب إجازة",
		TitleEn: "Cancel leave request",
		Fields:  fields,
	}, nil
}

func executeCancelLeave(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p cancelLeaveParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	leaveID, err := uuid.Parse(p.LeaveID)
	if err != nil {
		return nil, err
	}
	// CancelPendingLeave re-checks ownership and status against live state.
	if err := d.LeaveService.CancelPendingLeave(ctx, leaveID, actor.ID()); err != nil {
		return nil, Errf("%s", err.Error())
	}
	return map[string]any{"leave_id": p.LeaveID, "status": "cancelled"}, nil
}

// ─── leave.decision (supervisors) ───────────────────────────────────────────

type leaveDecisionParams struct {
	LeaveID  string `json:"leave_id"`
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

// pendingLeaveInScope confirms the leave sits in the actor's own approval
// queue — the exact scoping the Approval Center uses (role + department in the
// repository query) — rather than re-deriving department rules here.
func pendingLeaveInScope(ctx context.Context, d *Deps, actor *Actor, leaveID uuid.UUID) (*models.PendingLeaveRich, error) {
	rich, err := d.LeaveService.GetPendingLeavesRich(ctx, actor.ID())
	if err != nil {
		return nil, err
	}
	for i := range rich {
		if rich[i].ID == leaveID {
			return &rich[i], nil
		}
	}
	return nil, Errf("this leave request is not in your approval queue")
}

func validateLeaveDecision(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	if !actor.IsSupervisor() {
		return nil, nil, Errf("only team leaders, managers and admins can decide leave requests")
	}
	var p leaveDecisionParams
	if err := decode(raw, &p); err != nil {
		return nil, nil, err
	}
	if p.Decision != "approve" && p.Decision != "reject" {
		return nil, nil, Errf("decision must be approve or reject")
	}
	if p.Decision == "reject" && strings.TrimSpace(p.Reason) == "" {
		return nil, nil, Errf("a rejection requires a reason")
	}
	leaveID, err := uuid.Parse(p.LeaveID)
	if err != nil {
		return nil, nil, Errf("invalid leave_id — use get_pending_approvals")
	}
	leave, err := pendingLeaveInScope(ctx, d, actor, leaveID)
	if err != nil {
		return nil, nil, err
	}

	p.Reason = clipRunes(p.Reason, 500)
	frozen, _ := json.Marshal(p)

	titleAr, titleEn := "الموافقة على إجازة", "Approve leave"
	if p.Decision == "reject" {
		titleAr, titleEn = "رفض إجازة", "Reject leave"
	}
	fields := []summaryField{
		{"الموظف", "Employee", leave.EmployeeName},
		{"النوع", "Type", strOrEmpty(leave.LeaveTypeNameAr) + " / " + strOrEmpty(leave.LeaveTypeNameEn)},
		{"من", "From", temporal.DateString(leave.StartDate)},
		{"إلى", "To", temporal.DateString(leave.EndDate)},
	}
	if leave.StartTime != nil && leave.EndTime != nil {
		fields = append(fields, summaryField{"الوقت", "Time", clip5(*leave.StartTime) + " → " + clip5(*leave.EndTime)})
	}
	if leave.Reason != nil {
		fields = append(fields, summaryField{"سبب الطلب", "Request reason", clipRunes(*leave.Reason, 200)})
	}
	if p.Decision == "reject" {
		fields = append(fields, summaryField{"سبب الرفض", "Rejection reason", p.Reason})
	}
	return frozen, &actionSummary{TitleAr: titleAr, TitleEn: titleEn, Fields: fields}, nil
}

func executeLeaveDecision(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p leaveDecisionParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	leaveID, err := uuid.Parse(p.LeaveID)
	if err != nil {
		return nil, err
	}
	// Re-validate scope: role changes or the leave being decided elsewhere
	// between preview and approval must fail here, not execute.
	if !actor.IsSupervisor() {
		return nil, Errf("your role no longer allows deciding leave requests")
	}
	if _, err := pendingLeaveInScope(ctx, d, actor, leaveID); err != nil {
		return nil, Errf("this request is no longer pending in your queue (it may have been decided already)")
	}

	if p.Decision == "reject" {
		if err := d.LeaveService.RejectLeave(ctx, leaveID, actor.ID(), actor.Role(), p.Reason); err != nil {
			return nil, Errf("%s", err.Error())
		}
		return map[string]any{"leave_id": p.LeaveID, "decision": "rejected"}, nil
	}

	// The service picks the terminal state; team leaders and managers/admins
	// enter through their respective paths exactly as the REST API routes them.
	if actor.Role() == "team_leader" {
		err = d.LeaveService.ApproveByTeamLeader(ctx, leaveID, actor.ID())
	} else {
		err = d.LeaveService.ApproveByManager(ctx, leaveID, actor.ID())
	}
	if err != nil {
		return nil, Errf("%s", err.Error())
	}
	return map[string]any{"leave_id": p.LeaveID, "decision": "approved"}, nil
}

// ─── task.start / task.complete ─────────────────────────────────────────────

type taskActionParams struct {
	ExecutionID    string `json:"execution_id"`
	CompletionType string `json:"completion_type,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

func validateTaskAction(actionType string) func(context.Context, *Deps, *Actor, json.RawMessage) (json.RawMessage, *actionSummary, error) {
	return func(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
		var p taskActionParams
		if err := decode(raw, &p); err != nil {
			return nil, nil, err
		}
		execID, err := uuid.Parse(p.ExecutionID)
		if err != nil {
			return nil, nil, Errf("invalid execution_id — use get_my_tasks")
		}
		ownerID, _, err := d.TaskRepo.GetExecutionOwner(ctx, execID)
		if err != nil || ownerID != actor.ID() {
			return nil, nil, Errf("task not found among your tasks")
		}

		if actionType == ActionTaskComplete {
			if p.CompletionType == "" {
				p.CompletionType = "without_issue"
			}
			if p.CompletionType != "without_issue" && p.CompletionType != "with_issue" {
				return nil, nil, Errf("completion_type must be without_issue or with_issue")
			}
		} else {
			p.CompletionType = ""
		}
		p.Notes = clipRunes(p.Notes, 500)

		title := map[string][2]string{
			ActionTaskStart:    {"بدء مهمة", "Start task"},
			ActionTaskComplete: {"إكمال مهمة", "Complete task"},
		}[actionType]

		fields := []summaryField{}
		if date, err := d.TaskRepo.GetAssignmentDateByExecution(ctx, execID); err == nil {
			fields = append(fields, summaryField{"التاريخ", "Date", temporal.DateString(date)})
		}
		if actionType == ActionTaskComplete {
			label := "بدون مشاكل / without issue"
			if p.CompletionType == "with_issue" {
				label = "مع مشاكل / with issue"
			}
			fields = append(fields, summaryField{"النتيجة", "Outcome", label})
			if p.Notes != "" {
				fields = append(fields, summaryField{"ملاحظات", "Notes", p.Notes})
			}
		}
		frozen, _ := json.Marshal(p)
		return frozen, &actionSummary{TitleAr: title[0], TitleEn: title[1], Fields: fields}, nil
	}
}

func executeTaskStart(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p taskActionParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	execID, err := uuid.Parse(p.ExecutionID)
	if err != nil {
		return nil, err
	}
	if err := d.TaskService.StartTask(ctx, execID, actor.ID(), actor.Role()); err != nil {
		return nil, Errf("%s", err.Error())
	}
	return map[string]any{"execution_id": p.ExecutionID, "status": "in_progress"}, nil
}

func executeTaskComplete(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p taskActionParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	execID, err := uuid.Parse(p.ExecutionID)
	if err != nil {
		return nil, err
	}
	var notes *string
	if p.Notes != "" {
		notes = &p.Notes
	}
	if err := d.TaskService.CompleteTask(ctx, execID, actor.ID(), actor.Role(), p.CompletionType, notes); err != nil {
		return nil, Errf("%s", err.Error())
	}
	return map[string]any{"execution_id": p.ExecutionID, "status": "completed"}, nil
}

// ─── shift.check_in / shift.check_out ───────────────────────────────────────

type checkParams struct {
	ShiftRowID string `json:"shift_row_id"`
}

func validateCheck(isCheckIn bool) func(context.Context, *Deps, *Actor, json.RawMessage) (json.RawMessage, *actionSummary, error) {
	return func(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
		res, err := d.resolveCurrent(ctx, actor.Employee)
		if err != nil {
			return nil, nil, err
		}
		inst := res.Active
		if inst == nil && isCheckIn {
			// Arriving early: allow check-in for a shift starting soon today.
			if res.Upcoming != nil && res.Upcoming.StartAt.Sub(d.Clock.Now()) <= 2*time.Hour {
				inst = res.Upcoming
			}
		}
		if inst == nil {
			if isCheckIn {
				return nil, nil, Errf("no active shift now, and no shift starting within 2 hours — check-in is anchored to a shift")
			}
			return nil, nil, Errf("no active shift to check out from")
		}
		if isCheckIn && inst.CheckInAt != nil {
			return nil, nil, Errf("you already checked in at %s", inst.CheckInAt.Format("15:04"))
		}
		if !isCheckIn {
			if inst.CheckInAt == nil {
				return nil, nil, Errf("you have not checked in for this shift")
			}
			if inst.CheckOutAt != nil {
				return nil, nil, Errf("you already checked out at %s", inst.CheckOutAt.Format("15:04"))
			}
		}

		p := checkParams{ShiftRowID: inst.EmployeeShiftID.String()}
		frozen, _ := json.Marshal(p)
		title := [2]string{"تسجيل حضور", "Check in"}
		if !isCheckIn {
			title = [2]string{"تسجيل انصراف", "Check out"}
		}
		return frozen, &actionSummary{
			TitleAr: title[0],
			TitleEn: title[1],
			Fields: []summaryField{
				{"الشفت", "Shift", inst.ShiftName + " (" + temporal.ClockString(inst.StartAt) + "–" + temporal.ClockString(inst.EndAt) + ")"},
				{"التاريخ", "Date", temporal.DateString(inst.BusinessDate)},
				{"الوقت الآن", "Time now", d.Clock.Now().Format("15:04")},
			},
		}, nil
	}
}

func executeCheckIn(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p checkParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	rowID, err := uuid.Parse(p.ShiftRowID)
	if err != nil {
		return nil, err
	}
	if err := d.ScheduleService.CheckIn(ctx, rowID, actor.ID()); err != nil {
		return nil, Errf("%s", err.Error())
	}
	return map[string]any{"checked_in": true, "at": d.Clock.Now().Format("15:04")}, nil
}

func executeCheckOut(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p checkParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	rowID, err := uuid.Parse(p.ShiftRowID)
	if err != nil {
		return nil, err
	}
	if err := d.ScheduleService.CheckOut(ctx, rowID, actor.ID()); err != nil {
		return nil, Errf("%s", err.Error())
	}
	return map[string]any{"checked_out": true, "at": d.Clock.Now().Format("15:04")}, nil
}

// ─── ticket.create ──────────────────────────────────────────────────────────

type createTicketParams struct {
	TargetDepartmentID string `json:"target_department_id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
}

func validateCreateTicket(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	var p createTicketParams
	if err := decode(raw, &p); err != nil {
		return nil, nil, err
	}
	if actor.DeptID() == nil {
		return nil, nil, Errf("you are not assigned to a department")
	}
	targetID, err := uuid.Parse(p.TargetDepartmentID)
	if err != nil {
		return nil, nil, Errf("invalid target_department_id — use list_departments")
	}
	if targetID == *actor.DeptID() {
		return nil, nil, Errf("cannot open a ticket to your own department")
	}
	target, err := d.DepartmentRepo.GetByID(ctx, targetID)
	if err != nil || target == nil {
		return nil, nil, Errf("target department not found")
	}
	p.Title = strings.TrimSpace(clipRunes(p.Title, 150))
	p.Description = strings.TrimSpace(clipRunes(p.Description, 2000))
	if p.Title == "" || p.Description == "" {
		return nil, nil, Errf("title and description are required")
	}

	frozen, _ := json.Marshal(p)
	return frozen, &actionSummary{
		TitleAr: "فتح تذكرة",
		TitleEn: "Open ticket",
		Fields: []summaryField{
			{"إلى قسم", "To department", target.Name},
			{"العنوان", "Title", p.Title},
			{"الوصف", "Description", clipRunes(p.Description, 300)},
		},
	}, nil
}

func executeCreateTicket(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p createTicketParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	if actor.DeptID() == nil {
		return nil, Errf("you are not assigned to a department")
	}
	targetID, err := uuid.Parse(p.TargetDepartmentID)
	if err != nil {
		return nil, err
	}
	if targetID == *actor.DeptID() {
		return nil, Errf("cannot open a ticket to your own department")
	}
	ticket := &models.Ticket{
		SourceDepartmentID: *actor.DeptID(),
		TargetDepartmentID: targetID,
		CreatorID:          actor.ID(),
		Title:              p.Title,
		Description:        p.Description,
	}
	if err := d.TicketRepo.Create(ctx, ticket); err != nil {
		return nil, err
	}
	return map[string]any{"ticket_id": ticket.ID.String(), "status": "open"}, nil
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
