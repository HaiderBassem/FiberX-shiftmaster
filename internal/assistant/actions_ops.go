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

// Staged writes for the operational families: shift swaps, item requests and
// notification housekeeping.
//
// Each follows the same two-step contract as every other action in this
// package. validate runs when the model proposes: it resolves every reference
// against live data, checks the caller's authority, and freezes an exact
// parameter set plus a card the person can read. execute runs only after that
// person presses Approve, re-validates, and calls the same domain service the
// REST handlers call. Nothing here trusts a value the model produced beyond
// the identifiers it looked up through a tool, and every identifier is
// re-resolved before use.

const (
	ActionSwapRequest = "swap.request"
	ActionSwapRespond = "swap.respond"
	ActionItemRequest = "item.request"
	ActionNotifsRead  = "notification.mark_read"
)

// ─── swap.request ───────────────────────────────────────────────────────────

type swapRequestParams struct {
	TargetEmployeeID string `json:"target_employee_id"`
	Date             string `json:"date"`
	Reason           string `json:"reason,omitempty"`
}

func validateSwapRequest(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	var in swapRequestParams
	if err := decode(raw, &in); err != nil {
		return nil, nil, err
	}
	targetID, err := uuid.Parse(in.TargetEmployeeID)
	if err != nil {
		return nil, nil, Errf("invalid target_employee_id — get it from get_swap_candidates")
	}
	if targetID == actor.ID() {
		return nil, nil, Errf("you cannot swap a shift with yourself")
	}
	date, err := parseDateArg(d, in.Date)
	if err != nil {
		return nil, nil, err
	}
	if date.Before(temporal.BusinessDate(d.Clock.Now())) {
		return nil, nil, Errf("that date has already passed")
	}

	// The colleague must be one the swap service itself considers eligible on
	// that date. Re-deriving eligibility here — rather than trusting the id the
	// model passed — is what stops "swap my shift with the CEO".
	candidates, err := d.SwapService.GetEligibleShiftSwapTargets(ctx, actor.ID(), date)
	if err != nil {
		return nil, nil, err
	}
	var target *models.SwapEligibleEmployee
	for i := range candidates {
		if candidates[i].ID == targetID {
			target = &candidates[i]
			break
		}
	}
	if target == nil {
		return nil, nil, Errf("that colleague cannot take this shift on %s — call get_swap_candidates for who can", temporal.DateString(date))
	}

	// The caller must actually hold a shift that day to give away.
	instance, err := d.instanceForBusinessDate(ctx, actor.Employee, date)
	if err != nil {
		return nil, nil, err
	}
	if instance == nil {
		return nil, nil, Errf("you are not scheduled to work on %s, so there is no shift to swap", temporal.DateString(date))
	}

	frozen, err := json.Marshal(swapRequestParams{
		TargetEmployeeID: targetID.String(),
		Date:             temporal.DateString(date),
		Reason:           clipRunes(strings.TrimSpace(in.Reason), 300),
	})
	if err != nil {
		return nil, nil, err
	}
	name := strings.TrimSpace(target.FirstName + " " + target.LastName)
	summary := &actionSummary{
		TitleAr: "طلب تبديل شفت",
		TitleEn: "Shift swap request",
		Fields: []summaryField{
			{LabelAr: "مع", LabelEn: "With", Value: name},
			{LabelAr: "التاريخ", LabelEn: "Date", Value: temporal.DateString(date)},
			{LabelAr: "الشفت", LabelEn: "Shift", Value: fmt.Sprintf("%s %s–%s", instance.ShiftName, temporal.ClockString(instance.StartAt), temporal.ClockString(instance.EndAt))},
		},
	}
	if in.Reason != "" {
		summary.Fields = append(summary.Fields, summaryField{LabelAr: "السبب", LabelEn: "Reason", Value: clipRunes(in.Reason, 200)})
	}
	return frozen, summary, nil
}

func executeSwapRequest(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p swapRequestParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	targetID, err := uuid.Parse(p.TargetEmployeeID)
	if err != nil {
		return nil, Errf("invalid target employee")
	}
	date, err := time.Parse("2006-01-02", p.Date)
	if err != nil {
		return nil, Errf("invalid date")
	}
	// Re-check eligibility at execution time: the roster may have changed while
	// the card was on screen.
	candidates, err := d.SwapService.GetEligibleShiftSwapTargets(ctx, actor.ID(), date)
	if err != nil {
		return nil, err
	}
	eligible := false
	for i := range candidates {
		if candidates[i].ID == targetID {
			eligible = true
			break
		}
	}
	if !eligible {
		return nil, Errf("that colleague is no longer available for this swap")
	}

	instance, err := d.instanceForBusinessDate(ctx, actor.Employee, date)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, Errf("you are no longer scheduled to work that day")
	}

	swap := &models.ShiftSwap{
		RequesterID:      actor.ID(),
		TargetEmployeeID: targetID,
		ShiftDate:        date,
		ShiftID:          instance.ShiftID,
	}
	if p.Reason != "" {
		reason := p.Reason
		swap.Reason = &reason
	}
	if err := d.SwapService.RequestSwap(ctx, swap); err != nil {
		return nil, err
	}
	return map[string]any{"swap_id": swap.ID.String(), "status": "pending_colleague_response"}, nil
}

// ─── swap.respond ───────────────────────────────────────────────────────────

type swapRespondParams struct {
	SwapID string `json:"swap_id"`
	Accept bool   `json:"accept"`
}

func validateSwapRespond(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	var in swapRespondParams
	if err := decode(raw, &in); err != nil {
		return nil, nil, err
	}
	swapID, err := uuid.Parse(in.SwapID)
	if err != nil {
		return nil, nil, Errf("invalid swap_id — get it from get_my_swaps")
	}
	// Only a swap addressed to this person, still awaiting their answer.
	pending, err := d.SwapService.GetPendingSwapsForMe(ctx, actor.ID())
	if err != nil {
		return nil, nil, err
	}
	var found *models.ShiftSwap
	for i := range pending {
		if pending[i].ID == swapID {
			found = &pending[i]
			break
		}
	}
	if found == nil {
		return nil, nil, Errf("that swap is not waiting for your answer")
	}

	frozen, err := json.Marshal(swapRespondParams{SwapID: swapID.String(), Accept: in.Accept})
	if err != nil {
		return nil, nil, err
	}
	decisionAr, decisionEn := "رفض", "Decline"
	if in.Accept {
		decisionAr, decisionEn = "قبول", "Accept"
	}
	summary := &actionSummary{
		TitleAr: "الرد على طلب تبديل",
		TitleEn: "Respond to swap request",
		Fields: []summaryField{
			{LabelAr: "من", LabelEn: "From", Value: found.RequesterName},
			{LabelAr: "التاريخ", LabelEn: "Date", Value: temporal.DateString(found.ShiftDate)},
			{LabelAr: "القرار", LabelEn: "Decision", Value: decisionAr + " / " + decisionEn},
		},
	}
	return frozen, summary, nil
}

func executeSwapRespond(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p swapRespondParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	swapID, err := uuid.Parse(p.SwapID)
	if err != nil {
		return nil, Errf("invalid swap")
	}
	// EmployeeRespond enforces target ownership and pending status itself.
	if err := d.SwapService.EmployeeRespond(ctx, swapID, actor.ID(), p.Accept); err != nil {
		return nil, err
	}
	outcome := "declined"
	if p.Accept {
		outcome = "accepted_awaiting_supervisor"
	}
	return map[string]any{"swap_id": p.SwapID, "outcome": outcome}, nil
}

// ─── item.request ───────────────────────────────────────────────────────────

type itemRequestParams struct {
	CategoryID  string `json:"category_id"`
	Description string `json:"description"`
}

func validateItemRequest(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	var in itemRequestParams
	if err := decode(raw, &in); err != nil {
		return nil, nil, err
	}
	categoryID, err := uuid.Parse(in.CategoryID)
	if err != nil {
		return nil, nil, Errf("invalid category_id — get it from get_item_request_categories")
	}
	description := clipRunes(strings.TrimSpace(in.Description), 500)
	if description == "" {
		return nil, nil, Errf("say what is needed — the request must have a description")
	}
	if actor.DeptID() == nil {
		return nil, nil, Errf("you are not assigned to a department, so you cannot raise an item request")
	}
	// The category must belong to the caller's own department.
	cats, err := d.ItemReqService.GetCategoriesByDepartment(ctx, *actor.DeptID())
	if err != nil {
		return nil, nil, err
	}
	var catName string
	for _, c := range cats {
		if c.ID == categoryID {
			catName = c.Name
			break
		}
	}
	if catName == "" {
		return nil, nil, Errf("that category does not exist in your department")
	}

	frozen, err := json.Marshal(itemRequestParams{CategoryID: categoryID.String(), Description: description})
	if err != nil {
		return nil, nil, err
	}
	summary := &actionSummary{
		TitleAr: "طلب مستلزمات",
		TitleEn: "Item request",
		Fields: []summaryField{
			{LabelAr: "النوع", LabelEn: "Category", Value: catName},
			{LabelAr: "التفاصيل", LabelEn: "Details", Value: description},
		},
	}
	return frozen, summary, nil
}

func executeItemRequest(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p itemRequestParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	categoryID, err := uuid.Parse(p.CategoryID)
	if err != nil {
		return nil, Errf("invalid category")
	}
	req, err := d.ItemReqService.SubmitRequest(ctx, actor.ID(), categoryID, p.Description)
	if err != nil {
		return nil, err
	}
	return map[string]any{"request_id": req.ID.String(), "status": req.Status}, nil
}

// ─── notification.mark_read ─────────────────────────────────────────────────

type notifsReadParams struct {
	NotificationID string `json:"notification_id,omitempty"`
	All            bool   `json:"all,omitempty"`
}

func validateNotifsRead(ctx context.Context, d *Deps, actor *Actor, raw json.RawMessage) (json.RawMessage, *actionSummary, error) {
	var in notifsReadParams
	if err := decode(raw, &in); err != nil {
		return nil, nil, err
	}
	if !in.All && in.NotificationID == "" {
		return nil, nil, Errf("say which notification to mark read, or pass all=true")
	}

	if in.All {
		count, err := d.NotifRepo.GetUnreadCount(ctx, actor.ID())
		if err != nil {
			return nil, nil, err
		}
		if count == 0 {
			return nil, nil, Errf("there is nothing unread to mark")
		}
		frozen, err := json.Marshal(notifsReadParams{All: true})
		if err != nil {
			return nil, nil, err
		}
		return frozen, &actionSummary{
			TitleAr: "تعليم كل الإشعارات كمقروءة",
			TitleEn: "Mark all notifications read",
			Fields: []summaryField{
				{LabelAr: "العدد", LabelEn: "Count", Value: fmt.Sprintf("%d", count)},
			},
		}, nil
	}

	notifID, err := uuid.Parse(in.NotificationID)
	if err != nil {
		return nil, nil, Errf("invalid notification_id — get it from get_my_notifications")
	}
	// Ownership: the notification must be one of the caller's unread ones.
	unread, err := d.NotifRepo.GetUnread(ctx, actor.ID())
	if err != nil {
		return nil, nil, err
	}
	var title string
	for _, n := range unread {
		if n.ID == notifID {
			title = n.Title
			break
		}
	}
	if title == "" {
		return nil, nil, Errf("that notification is not in your unread list")
	}
	frozen, err := json.Marshal(notifsReadParams{NotificationID: notifID.String()})
	if err != nil {
		return nil, nil, err
	}
	return frozen, &actionSummary{
		TitleAr: "تعليم إشعار كمقروء",
		TitleEn: "Mark notification read",
		Fields:  []summaryField{{LabelAr: "الإشعار", LabelEn: "Notification", Value: clipRunes(title, 120)}},
	}, nil
}

func executeNotifsRead(ctx context.Context, d *Deps, actor *Actor, frozen json.RawMessage) (map[string]any, error) {
	var p notifsReadParams
	if err := json.Unmarshal(frozen, &p); err != nil {
		return nil, err
	}
	if p.All {
		if err := d.NotifRepo.MarkAllAsRead(ctx, actor.ID()); err != nil {
			return nil, err
		}
		return map[string]any{"marked": "all"}, nil
	}
	notifID, err := uuid.Parse(p.NotificationID)
	if err != nil {
		return nil, Errf("invalid notification")
	}
	// MarkAsRead is scoped to the recipient in SQL, so a foreign id is a no-op
	// rather than a leak.
	if err := d.NotifRepo.MarkAsRead(ctx, notifID, actor.ID()); err != nil {
		return nil, err
	}
	return map[string]any{"marked": p.NotificationID}, nil
}
