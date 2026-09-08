package assistant

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// The propose_* tools are the ONLY write-adjacent tools in the catalogue, and
// none of them mutates domain state: each validates and stages a pending
// action for human approval. There is deliberately no approve tool, no
// execute tool, and no way to skip the card.

type convKeyType struct{}

var convKey convKeyType

// withConversation tags the request context with the current conversation so
// staged actions attach to it (and supersede its earlier cards).
func withConversation(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, convKey, id)
}

func conversationFrom(ctx context.Context) *uuid.UUID {
	if id, ok := ctx.Value(convKey).(uuid.UUID); ok {
		return &id
	}
	return nil
}

func proposeTool(name, description string, roles []string, inputSchema string, actionType string) Tool {
	return Tool{
		Name:        name,
		Description: description,
		Roles:       roles,
		InputSchema: schema(inputSchema),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			return propose(ctx, d, actor, conversationFrom(ctx), actionType, input)
		},
	}
}

func toolProposeLeaveRequest() Tool {
	return proposeTool(
		"propose_leave_request",
		"Stage a WHOLE-DAY leave request for the caller (an approval card is shown; nothing is submitted until the user approves). "+
			"Use get_leave_types first for the leave_type_id; for زمنية / hourly leave use propose_hourly_leave instead.",
		nil,
		`{
			"type":"object",
			"properties":{
				"leave_type_id":{"type":"string"},
				"start_date":{"type":"string","description":"YYYY-MM-DD"},
				"end_date":{"type":"string","description":"YYYY-MM-DD (same as start_date for one day)"},
				"reason":{"type":"string"}
			},
			"required":["leave_type_id","start_date","end_date"],
			"additionalProperties":false
		}`,
		ActionLeaveRequest,
	)
}

func toolProposeHourlyLeave() Tool {
	return proposeTool(
		"propose_hourly_leave",
		"Stage an hourly leave (زمنية) anchored to the caller's actual shift; the server resolves the window, including "+
			"overnight shifts that cross midnight — NEVER compute the window yourself. 'The last hour of my shift' = anchor shift_end, minutes 60; "+
			"'first half hour' = anchor shift_start, minutes 30; a specific window = anchor explicit with start_time/end_time (HH:MM). "+
			"Omit date to use the currently active (or next) shift — this is what makes 'last hour' correct even after midnight. "+
			// The Arabic is here because this is the sentence people actually
			// type, and because the arithmetic they are asking for is the
			// arithmetic the model gets wrong: asked for the last hour of a
			// shift ending 00:30 it will happily narrate 00:30–01:30. Naming
			// the phrases on the tool that computes them correctly keeps the
			// model out of the calculation entirely.
			"أريد آخر ساعة من دوامي = anchor shift_end, minutes 60. آخر نص ساعة = anchor shift_end, minutes 30. "+
			"أول ساعة / أول نص ساعة من الدوام = anchor shift_start. إذا ما عدك إلا نوع زمني واحد لا تسأل عنه — استخدمه.",
		nil,
		`{
			"type":"object",
			"properties":{
				"leave_type_id":{"type":"string","description":"an hourly leave type id; may be omitted when only one hourly type exists"},
				"date":{"type":"string","description":"business date of the shift, YYYY-MM-DD; omit for the current shift"},
				"anchor":{"type":"string","enum":["shift_end","shift_start","explicit"]},
				"minutes":{"type":"integer","minimum":1,"maximum":720},
				"start_time":{"type":"string","description":"HH:MM, for explicit windows"},
				"end_time":{"type":"string","description":"HH:MM, for explicit windows"},
				"reason":{"type":"string"}
			},
			"additionalProperties":false
		}`,
		ActionHourlyLeave,
	)
}

func toolProposeCancelLeave() Tool {
	return proposeTool(
		"propose_cancel_leave",
		"Stage cancelling one of the CALLER'S OWN leave requests (only pending or team-leader-approved ones can be cancelled). "+
			"Get leave_id from get_my_leave_requests.",
		nil,
		`{
			"type":"object",
			"properties":{"leave_id":{"type":"string"}},
			"required":["leave_id"],
			"additionalProperties":false
		}`,
		ActionCancelLeave,
	)
}

func toolProposeLeaveDecision() Tool {
	return proposeTool(
		"propose_leave_decision",
		"Supervisors only: stage approving or rejecting a leave request that is in YOUR approval queue "+
			"(get_pending_approvals). Rejection requires a reason. The decision executes only after you approve the card.",
		[]string{"team_leader", "manager", "admin"},
		`{
			"type":"object",
			"properties":{
				"leave_id":{"type":"string"},
				"decision":{"type":"string","enum":["approve","reject"]},
				"reason":{"type":"string","description":"required when rejecting"}
			},
			"required":["leave_id","decision"],
			"additionalProperties":false
		}`,
		ActionLeaveDecision,
	)
}

func toolProposeTaskAction() Tool {
	return Tool{
		Name: "propose_task_action",
		Description: "Stage starting or completing one of the CALLER'S OWN tasks (execution_id from get_my_tasks). " +
			"Completing accepts completion_type (without_issue | with_issue) and notes.",
		InputSchema: schema(`{
			"type":"object",
			"properties":{
				"execution_id":{"type":"string"},
				"action":{"type":"string","enum":["start","complete"]},
				"completion_type":{"type":"string","enum":["without_issue","with_issue"]},
				"notes":{"type":"string"}
			},
			"required":["execution_id","action"],
			"additionalProperties":false
		}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			var in struct {
				ExecutionID    string `json:"execution_id"`
				Action         string `json:"action"`
				CompletionType string `json:"completion_type"`
				Notes          string `json:"notes"`
			}
			if err := decode(input, &in); err != nil {
				return nil, err
			}
			actionType := ActionTaskStart
			if in.Action == "complete" {
				actionType = ActionTaskComplete
			} else if in.Action != "start" {
				return nil, Errf("action must be start or complete")
			}
			params, _ := json.Marshal(taskActionParams{
				ExecutionID:    in.ExecutionID,
				CompletionType: in.CompletionType,
				Notes:          in.Notes,
			})
			return propose(ctx, d, actor, conversationFrom(ctx), actionType, params)
		},
	}
}

func toolProposeCheckIn() Tool {
	return proposeTool(
		"propose_check_in",
		"Stage checking the caller in to their current shift (or one starting within 2 hours). Idempotency-safe.",
		nil,
		`{"type":"object","properties":{},"additionalProperties":false}`,
		ActionCheckIn,
	)
}

func toolProposeCheckOut() Tool {
	return proposeTool(
		"propose_check_out",
		"Stage checking the caller out of their current shift (requires an earlier check-in).",
		nil,
		`{"type":"object","properties":{},"additionalProperties":false}`,
		ActionCheckOut,
	)
}

func toolProposeCreateTicket() Tool {
	return proposeTool(
		"propose_create_ticket",
		"Stage opening a cross-department ticket from the caller's department to another department "+
			"(list_departments gives target IDs). Not for requests inside your own department.",
		nil,
		`{
			"type":"object",
			"properties":{
				"target_department_id":{"type":"string"},
				"title":{"type":"string"},
				"description":{"type":"string"}
			},
			"required":["target_department_id","title","description"],
			"additionalProperties":false
		}`,
		ActionCreateTicket,
	)
}

func toolListDepartments() Tool {
	return Tool{
		Name:        "list_departments",
		Description: "All department names and IDs — needed as ticket targets. Names are organisational data visible to every employee.",
		InputSchema: schema(`{"type":"object","properties":{},"additionalProperties":false}`),
		Run: func(ctx context.Context, d *Deps, actor *Actor, input json.RawMessage) (any, error) {
			depts, err := d.DepartmentRepo.GetAll(ctx)
			if err != nil {
				return nil, err
			}
			out := []map[string]string{}
			for _, dept := range depts {
				out = append(out, map[string]string{"id": dept.ID.String(), "name": dept.Name, "code": dept.DepartmentCode})
			}
			return map[string]any{"departments": out}, nil
		},
	}
}

func toolProposeSwapRequest() Tool {
	return proposeTool(
		"propose_swap_request",
		"Stage asking a colleague to take one of the CALLER'S OWN shifts. Call get_swap_candidates for that date first — "+
			"only the colleagues it lists can be asked, and the swap still needs their acceptance and then a supervisor's approval "+
			"after this card is approved.",
		nil,
		`{
			"type":"object",
			"properties":{
				"target_employee_id":{"type":"string","description":"from get_swap_candidates"},
				"date":{"type":"string","description":"business date of the shift to give away, YYYY-MM-DD"},
				"reason":{"type":"string"}
			},
			"required":["target_employee_id","date"],
			"additionalProperties":false
		}`,
		ActionSwapRequest,
	)
}

func toolProposeSwapResponse() Tool {
	return proposeTool(
		"propose_swap_response",
		"Stage the caller's answer to a swap request ADDRESSED TO THEM (swap_id from get_my_swaps). "+
			"Accepting sends it on to a supervisor for final approval; declining ends it.",
		nil,
		`{
			"type":"object",
			"properties":{
				"swap_id":{"type":"string"},
				"accept":{"type":"boolean"}
			},
			"required":["swap_id","accept"],
			"additionalProperties":false
		}`,
		ActionSwapRespond,
	)
}

func toolProposeItemRequest() Tool {
	return proposeTool(
		"propose_item_request",
		"Stage a request for equipment or supplies from the caller's own department (category_id from "+
			"get_item_request_categories). Use it when someone says they need a thing — a router, a SIM, a headset.",
		nil,
		`{
			"type":"object",
			"properties":{
				"category_id":{"type":"string"},
				"description":{"type":"string","description":"what exactly is needed, in the user's own words"}
			},
			"required":["category_id","description"],
			"additionalProperties":false
		}`,
		ActionItemRequest,
	)
}

func toolProposeMarkNotificationsRead() Tool {
	return proposeTool(
		"propose_mark_notifications_read",
		"Stage marking the caller's OWN notifications as read: one by notification_id (from get_my_notifications), "+
			"or all of them with all=true. Nothing is marked until the user approves the card.",
		nil,
		`{
			"type":"object",
			"properties":{
				"notification_id":{"type":"string"},
				"all":{"type":"boolean"}
			},
			"additionalProperties":false
		}`,
		ActionNotifsRead,
	)
}

// AllTools is the complete catalogue in presentation order.
func AllTools() []Tool {
	return []Tool{
		// grounding: identity and authoritative time
		toolResolveDate(),
		toolGetMyProfile(),
		// self-service reads
		toolGetCurrentShift(),
		toolGetSchedule(),
		toolGetMyTasks(),
		toolGetLeaveTypes(),
		toolGetMyLeaveBalances(),
		toolGetMyLeaves(),
		toolGetMySwaps(),
		toolGetMyItemRequests(),
		toolGetNotifications(),
		toolGetAnnouncements(),
		toolGetMyDepartment(),
		toolListDepartments(),
		// operations
		toolGetTeamMembers(),
		toolGetHandovers(),
		toolGetTickets(),
		toolGetItemCategories(),
		toolGetSwapCandidates(),
		toolCheckLeaveEligibility(),
		// supervisor reads
		toolGetTeamStatus(),
		toolGetDepartmentOverview(),
		toolGetPendingApprovals(),
		toolGetShiftCoverage(),
		// catalog
		toolSearchPlans(),
		toolGetPlanDetails(),
		// knowledge
		toolSearchKnowledge(),
		toolGetKnowledgeDocument(),
		toolGetInfoTableRows(),
		// staged writes
		toolProposeLeaveRequest(),
		toolProposeHourlyLeave(),
		toolProposeCancelLeave(),
		toolProposeLeaveDecision(),
		toolProposeTaskAction(),
		toolProposeCheckIn(),
		toolProposeCheckOut(),
		toolProposeCreateTicket(),
		toolProposeSwapRequest(),
		toolProposeSwapResponse(),
		toolProposeItemRequest(),
		toolProposeMarkNotificationsRead(),
	}
}
