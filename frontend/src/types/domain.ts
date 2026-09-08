/**
 * Shared domain types.
 *
 * These mirror the Go structs in internal/models. Before this file existed each
 * feature redeclared whatever subset it happened to need, or typed the value
 * `any`, so the same entity had a different shape in every screen and a renamed
 * field failed silently at runtime instead of loudly at build time.
 *
 * Keep these in step with internal/models when the API changes. Fields are
 * optional here only where the API genuinely omits them; a field that is always
 * present should be required, so that forgetting it is a type error.
 */

// ── Shared primitives ──────────────────────────────────────────────────────

/** RFC3339 timestamp as returned by the API. Render via lib/dateUtils. */
export type Timestamp = string;

/** Calendar date, `YYYY-MM-DD`. */
export type DateOnly = string;

export type UUID = string;

export type EmployeeRole = 'employee' | 'team_leader' | 'manager' | 'admin' | 'hr';

/** Mirrors the employee_status Postgres enum exactly; there is no 'suspended'. */
export type EmployeeStatus = 'active' | 'inactive' | 'on_leave' | 'terminated';

// ── Core entities ──────────────────────────────────────────────────────────

export interface Employee {
  id: UUID;
  employee_code: string;
  first_name: string;
  last_name: string;
  gender: string;
  phone: string | null;
  email: string;
  hire_date: Timestamp;
  role: EmployeeRole;
  department_id: UUID | null;
  position: string | null;
  default_shift_id: UUID | null;
  weekly_off_days: number;
  can_cover_night_shift: boolean;
  status: EmployeeStatus;
  profile_image: string | null;
  last_login: Timestamp | null;
  secondary_phone?: string | null;
  secondary_email?: string | null;

  // Per-employee capability flags, checked alongside the role.
  can_create_tables?: boolean;
  can_manage_help_docs?: boolean;
  can_post_announcements?: boolean;
  can_manage_fiberx_data?: boolean;
  can_manage_services?: boolean;

  ui_preferences?: Record<string, unknown>;
  created_at: Timestamp;
  updated_at: Timestamp;
}

export interface Department {
  id: UUID;
  department_code: string;
  name: string;
  description: string | null;
  fiberx_enabled: boolean;
  max_leaves_per_day: number | null;
  max_hourly_leaves_per_day: number | null;
  manager_ids: UUID[];
  /** Feature modules switched on for this department. */
  active_modules: string[];
  created_at: Timestamp;
  updated_at: Timestamp;
}

export interface Shift {
  id: UUID;
  shift_code: string;
  name: string;
  name_en: string | null;
  /** The API serialises TIME columns as a full timestamp; only the clock part is meaningful. */
  start_time: string;
  end_time: string;
  color_code: string | null;
  requires_vehicle: boolean;
  min_rest_hours: number;
  department_id: UUID | null;
  created_at: Timestamp;
}

// ── Scheduling ─────────────────────────────────────────────────────────────

/**
 * employee_shifts.shift_status is the shift_status_type enum. All eight members
 * are listed: a row can legitimately be sick, training or business_trip, and a
 * union that omits them makes those days unrepresentable rather than impossible.
 */
export type ShiftStatus =
  | 'working'
  | 'off'
  | 'leave'
  | 'sick'
  | 'vacation'
  | 'training'
  | 'business_trip'
  | 'hourly';

/**
 * Who authored a scheduled day. Mirrors models.ShiftSource*.
 *
 * `generated` rows are re-derived from the weekly pattern; `manual` and `leave`
 * rows are never rewritten.
 */
export type ShiftSource = 'generated' | 'manual' | 'leave';

export interface EmployeeShift {
  id: UUID;
  schedule_id: UUID;
  employee_id: UUID;
  shift_id: UUID | null;
  shift_date: DateOnly;
  shift_status: ShiftStatus;
  /**
   * For hourly leave this is prefixed with `[hourly] `. That prefix is a wire
   * contract, matching models.HourlyLeaveReasonPrefix.
   */
  leave_reason: string | null;
  check_in_time: Timestamp | null;
  check_out_time: Timestamp | null;
  source: ShiftSource;
  created_at: Timestamp;
  updated_at: Timestamp;
}

/**
 * A scheduled row joined to the person it belongs to, as the department-wide
 * endpoints return it. Mirrors models.EmployeeShiftExtended.
 */
export interface EmployeeShiftExtended extends EmployeeShift {
  first_name: string;
  last_name: string;
  employee_role: EmployeeRole;
  default_shift_id: UUID | null;
}

/** One weekday of an employee's repeating weekly pattern. */
export interface PatternDay {
  day_of_week: number;
  shift_id: UUID | null;
  is_off: boolean;
}

// ── Leave ──────────────────────────────────────────────────────────────────

export type LeaveStatus =
  | 'pending'
  | 'approved_by_team_leader'
  | 'approved_by_manager'
  | 'rejected'
  | 'cancelled';

export interface LeaveType {
  id: UUID;
  name_ar: string;
  name_en: string;
  is_paid: boolean;
  color_code: string;
  is_active: boolean;
  requires_approval: boolean;
  days_per_year: number;
  unit: 'days' | 'hours';
  reset_cycle: 'annual' | 'monthly';
  carries_forward: boolean;
  /** Occupies part of a day. Never infer this from the type's name. */
  is_hourly: boolean;
  /** Exempt from the department's daily leave caps. */
  bypasses_daily_limit: boolean;
}

export interface Leave {
  id: UUID;
  employee_id: UUID;
  leave_type_id: UUID;
  leave_type_name_ar: string | null;
  leave_type_name_en: string | null;
  leave_type_is_hourly: boolean;
  start_date: DateOnly;
  end_date: DateOnly;
  total_days: number;
  /** Present only for hourly leave. */
  start_time: string | null;
  end_time: string | null;
  reason: string | null;
  status: LeaveStatus;
  applied_date: DateOnly;
  rejection_reason: string | null;
  created_at: Timestamp;
  updated_at: Timestamp;
}

/** A single approval action with the approver's name. Mirrors models.LeaveApprovalDetail. */
export interface LeaveApprovalDetail {
  approver_name: string;
  approver_role: string;
  action: string;
  notes: string | null;
  created_at: Timestamp;
}

/**
 * A row of the leave history list. Mirrors models.LeaveHistoryRow.
 *
 * Note this is NOT a Leave: the id is `leave_id`, the employee is identified by
 * name and code rather than by id, and it carries the approval trail.
 */
export interface LeaveHistoryRow {
  leave_id: UUID;
  employee_name: string;
  employee_code: string;
  employee_profile_image: string | null;
  leave_type_id: UUID;
  leave_type_name_ar: string | null;
  leave_type_name_en: string | null;
  start_date: DateOnly;
  end_date: DateOnly;
  total_days: number;
  reason: string | null;
  status: LeaveStatus;
  applied_date: DateOnly | null;
  rejection_reason: string | null;
  approvals: LeaveApprovalDetail[];
}

export interface LeaveBalance {
  id: UUID;
  employee_id: UUID;
  leave_type_id: UUID;
  leave_type_name_ar?: string;
  leave_type_name_en?: string;
  color_code?: string;
  year: number;
  month: number;
  allocated_amount: number;
  used_amount: number;
  pending_amount: number;
  unit: 'days' | 'hours';
  reset_cycle: 'annual' | 'monthly';
}

/** A swap candidate: an employee plus whether they are off on the target date. */
export interface SwapEligibleEmployee extends Employee {
  is_off: boolean;
}

/** The /employees/me/profile-stats payload. */
export interface ProfileStats {
  leave_balances: LeaveBalance[];
  completed_tasks: number;
  active_tasks: number;
  total_leaves_taken: number;
  total_hourly_leaves_taken: number;
  worked_hours: number;
}

// ── Swaps ──────────────────────────────────────────────────────────────────

export type SwapStatus =
  | 'pending'
  | 'employee_accepted'
  | 'approved'
  | 'rejected'
  | 'cancelled';

export interface ShiftSwap {
  id: UUID;
  requester_id: UUID;
  target_employee_id: UUID;
  shift_date: DateOnly;
  shift_id: UUID | null;
  reason: string | null;
  status: SwapStatus;
  requester_name?: string;
  /** The wire name is target_employee_name; models.ShiftSwap serialises it under that key. */
  target_employee_name?: string;
  requester_profile_image?: string | null;
  target_profile_image?: string | null;
  approved_by_team_leader?: UUID | null;
  approved_by_manager?: UUID | null;
  approval_date?: Timestamp | null;
  created_at: Timestamp;
  updated_at?: Timestamp;
}

// ── Tasks ──────────────────────────────────────────────────────────────────

/** task_executions.status is the task_status enum; there is no 'skipped'. */
export type TaskExecutionStatus = 'pending' | 'in_progress' | 'completed' | 'cancelled' | 'overdue';

export interface TaskBoard {
  id: UUID;
  name: string;
  description: string | null;
  department_id: UUID | null;
  created_at: Timestamp;
}

/** Aggregate completion stats for one board. Mirrors models.TaskBoardStats. */
export interface TaskBoardStats {
  board_id: UUID;
  board_name: string;
  total_assigned: number;
  total_pending: number;
  total_in_progress: number;
  total_completed: number;
  completion_pct: number;
}

export interface TaskSchedule {
  id: UUID;
  board_id: UUID | null;
  title: string;
  description: string | null;
  recurrence: string | null;
  /** Day-of-week indices, present when recurrence is 'weekly'. */
  recurrence_days: number[] | null;
  max_assignees: number;
  shift_id: UUID | null;
  is_active: boolean;
  created_at: Timestamp;
}

export interface TaskExecution {
  id: UUID;
  assignment_id: UUID;
  status: TaskExecutionStatus;
  completion_type: string | null;
  notes: string | null;
  started_at: Timestamp | null;
  completed_at: Timestamp | null;
}

/**
 * One row of "my tasks this week" — a task joined to its assignment, board and
 * shift. Mirrors models.MyTaskRow.
 *
 * The title field is `task_title`, not `title`: this is a joined row, not a
 * task record, and the board and shift names it carries are already resolved.
 */
export interface MyTaskRow {
  assignment_id: UUID;
  assigned_date: Timestamp;
  task_title: string;
  task_description: string | null;
  board_name: string | null;
  shift_name: string | null;
  shift_code: string | null;
  shift_color: string | null;
  execution_id: UUID | null;
  status: TaskExecutionStatus;
  completion_type: string | null;
  started_at: Timestamp | null;
  completed_at: Timestamp | null;
  notes: string | null;
}

// ── Cross-department ───────────────────────────────────────────────────────

export type TicketStatus = 'open' | 'closed';

/** One comment on a ticket. Mirrors models.TicketComment. */
export interface TicketComment {
  id: UUID;
  ticket_id: UUID;
  employee_id: UUID;
  author_name: string;
  author_image?: string | null;
  comment: string;
  /** A JSON-encoded array of image URLs, or null. */
  attachments: string | null;
  created_at: Timestamp;
}

export interface Ticket {
  id: UUID;
  title: string;
  description: string;
  source_department_id: UUID;
  target_department_id: UUID;
  /** The wire name is creator_id; models.Ticket serialises it under that key. */
  creator_id: UUID;
  status: TicketStatus;
  closed_by: UUID | null;
  /** A JSON-encoded array of image URLs, or null. */
  attachments: string | null;
  created_at: Timestamp;
  updated_at: Timestamp;

  // Joined for the UI; absent unless the handler filled them in.
  creator_name?: string | null;
  creator_profile_image?: string | null;
  source_department?: string | null;
  target_department?: string | null;
  closed_by_name?: string | null;
  comments?: TicketComment[];
}

export type HandoverStatus = 'open' | 'claimed' | 'completed';

/** One comment on a handover. Mirrors models.HandoverComment. */
export interface HandoverComment {
  id: UUID;
  employee_id: UUID;
  author_name: string;
  comment: string;
  created_at: Timestamp;
}

export interface Handover {
  id: UUID;
  department_id: UUID;
  /** The wire name is creator_id; models.Handover serialises it under that key. */
  creator_id: UUID;
  shift_summary: string;
  pending_issues: string;
  status: HandoverStatus;
  claimed_by: UUID | null;
  done_by: UUID | null;
  created_at: Timestamp;
  updated_at: Timestamp;

  // Joined for the UI; absent unless the handler filled them in.
  creator_name?: string | null;
  claimer_name?: string | null;
  done_by_name?: string | null;
  comments?: HandoverComment[];
}

// ── Notifications ──────────────────────────────────────────────────────────

/**
 * notifications.priority is the notification_priority enum. Announcements carry
 * their own info/normal/important/critical column and are not typed by this.
 */
export type NotificationPriority = 'low' | 'medium' | 'high';

export interface AppNotification {
  id: UUID;
  recipient_id: UUID;
  type: string;
  title: string;
  message: string | null;
  related_entity_type: string | null;
  related_entity_id: UUID | null;
  priority: NotificationPriority;
  is_read: boolean;
  action_url: string | null;
  created_at: Timestamp;
}

/**
 * A pending leave with the joined detail the approval dashboard needs.
 * Mirrors models.PendingLeaveRich.
 */
export interface PendingLeaveRich {
  id: UUID;
  employee_id: UUID;
  leave_type_id: UUID;
  leave_type_name_ar: string | null;
  leave_type_name_en: string | null;
  start_date: DateOnly;
  end_date: DateOnly;
  total_days: number;
  reason: string | null;
  status: LeaveStatus;
  applied_date: DateOnly | null;
  employee_name: string;
  employee_code: string;
  employee_profile_image: string | null;
  default_shift_id: string;
  shift_name: string;
  shift_code: string;
  department_name: string;
  tl_approvals: number;
  total_tls: number;
  start_time: string | null;
  end_time: string | null;
}

/** An employee request for an item. Mirrors models.ItemRequest. */
export interface ItemRequest {
  id: UUID;
  employee_id: UUID;
  category_id: UUID;
  description: string;
  status: string;
  created_at: Timestamp;
  updated_at: Timestamp;
  category_name?: string | null;
  employee_name?: string | null;
}

// ── Audit ──────────────────────────────────────────────────────────────────

/** One recorded action. Mirrors models.AuditLog; the JSONB columns arrive as strings. */
export interface AuditLog {
  id: UUID;
  employee_id: UUID | null;
  action: string;
  table_name: string;
  record_id: UUID | null;
  old_data: string | null;
  new_data: string | null;
  ip_address: string | null;
  user_agent: string | null;
  created_at: Timestamp;
}

// ── API envelope ───────────────────────────────────────────────────────────

/** Every endpoint responds in this shape. */
export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
  meta?: { count?: number };
}
