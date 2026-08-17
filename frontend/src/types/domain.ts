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

export type EmployeeStatus = 'active' | 'inactive' | 'suspended';

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

export type ShiftStatus = 'working' | 'off' | 'leave' | 'vacation' | 'hourly';

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

// ── Swaps ──────────────────────────────────────────────────────────────────

export type SwapStatus =
  | 'pending'
  | 'employee_accepted'
  | 'employee_rejected'
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
  target_name?: string;
  requester_profile_image?: string | null;
  target_profile_image?: string | null;
  created_at: Timestamp;
}

// ── Tasks ──────────────────────────────────────────────────────────────────

export type TaskExecutionStatus = 'pending' | 'in_progress' | 'completed' | 'skipped';

export interface TaskBoard {
  id: UUID;
  name: string;
  description: string | null;
  department_id: UUID | null;
  created_at: Timestamp;
}

export interface TaskSchedule {
  id: UUID;
  board_id: UUID | null;
  title: string;
  description: string | null;
  recurrence: string | null;
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

// ── Cross-department ───────────────────────────────────────────────────────

export type TicketStatus = 'open' | 'closed';

export interface Ticket {
  id: UUID;
  title: string;
  description: string | null;
  source_department_id: UUID;
  target_department_id: UUID;
  created_by: UUID;
  status: TicketStatus;
  created_at: Timestamp;
  closed_at: Timestamp | null;
}

export type HandoverStatus = 'open' | 'claimed' | 'completed';

export interface Handover {
  id: UUID;
  department_id: UUID;
  shift_id: UUID | null;
  created_by: UUID;
  shift_summary: string | null;
  pending_issues: string | null;
  status: HandoverStatus;
  claimed_by: UUID | null;
  claimer_notes: string | null;
  done_by: UUID | null;
  created_at: Timestamp;
}

// ── Notifications ──────────────────────────────────────────────────────────

export type NotificationPriority = 'low' | 'normal' | 'medium' | 'high';

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

// ── API envelope ───────────────────────────────────────────────────────────

/** Every endpoint responds in this shape. */
export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
  meta?: { count?: number };
}
