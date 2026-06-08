package models

import (
	"time"

	"github.com/google/uuid"
)

// Internal models
type Department struct {
	ID             uuid.UUID   `json:"id"`
	DepartmentCode string      `json:"department_code"`
	Name           string      `json:"name"`
	Description    *string     `json:"description"`
	ManagerIDs     []uuid.UUID `json:"manager_ids"` // populated via department_managers join table
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type Employee struct {
	ID                 uuid.UUID `json:"id"`
	EmployeeCode       string    `json:"employee_code"`
	FirstName          string    `json:"first_name"`
	LastName           string    `json:"last_name"`
	Gender             string    `json:"gender"`
	Phone              *string   `json:"phone"`
	Email              string    `json:"email"`
	PasswordHash       *string   `json:"-"` // Omit from JSON
	HireDate           time.Time `json:"hire_date"`
	Role               string    `json:"role"`
	DepartmentID       *uuid.UUID `json:"department_id"`
	Position           *string   `json:"position"`
	DefaultShiftID     *uuid.UUID `json:"default_shift_id"`
	WeeklyOffDays      int       `json:"weekly_off_days"`
	CanCoverNightShift bool      `json:"can_cover_night_shift"`
	Status             string    `json:"status"`
	ProfileImage       *string   `json:"profile_image"`
	RememberToken      *string   `json:"-"`
	LastLogin          *time.Time `json:"last_login"`
	SecondaryPhone     *string   `json:"secondary_phone"`
	SecondaryEmail     *string   `json:"secondary_email"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CreatedBy          *uuid.UUID `json:"created_by"`
	CanCreateTables    bool      `json:"can_create_tables"`
	CanManageHelpDocs  bool      `json:"can_manage_help_docs"`
	CanPostAnnouncements bool    `json:"can_post_announcements"`
	CanManageFiberxData  bool      `json:"can_manage_fiberx_data"`
	UIPreferences      map[string]interface{} `json:"ui_preferences"`
}

type Shift struct {
	ID              uuid.UUID  `json:"id"`
	ShiftCode       string     `json:"shift_code"`
	Name            string     `json:"name"`
	NameEn          *string    `json:"name_en"`
	StartTime       time.Time  `json:"start_time"` // Storing TIME as time.Time
	EndTime         time.Time  `json:"end_time"`
	ColorCode       *string    `json:"color_code"`
	RequiresVehicle bool       `json:"requires_vehicle"`
	MinRestHours    int        `json:"min_rest_hours"`
	DepartmentID    *uuid.UUID `json:"department_id"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ScheduleTemplate struct {
	ID          uuid.UUID  `json:"id"`
	EmployeeID  uuid.UUID  `json:"employee_id"`
	DayOfWeek   int        `json:"day_of_week"`
	ShiftID     *uuid.UUID `json:"shift_id"`
	IsOff       bool       `json:"is_off"`
	ValidFrom   *time.Time `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type WeeklySchedule struct {
	ID            uuid.UUID  `json:"id"`
	WeekStartDate time.Time  `json:"week_start_date"`
	WeekEndDate   time.Time  `json:"week_end_date"`
	TemplateID    *uuid.UUID `json:"template_id"`
	Status        string     `json:"status"`
	PublishedBy   *uuid.UUID `json:"published_by"`
	PublishedAt   *time.Time `json:"published_at"`
	Notes         *string    `json:"notes"`
	DepartmentID  *uuid.UUID `json:"department_id"`
	CreatedAt     time.Time  `json:"created_at"`
}

type EmployeeShift struct {
	ID                    uuid.UUID  `json:"id"`
	ScheduleID            uuid.UUID  `json:"schedule_id"`
	EmployeeID            uuid.UUID  `json:"employee_id"`
	ShiftID               *uuid.UUID `json:"shift_id"`
	ShiftDate             time.Time  `json:"shift_date"`
	ShiftStatus           string     `json:"shift_status"`
	LeaveReason           *string    `json:"leave_reason"`
	IsReplacement         bool       `json:"is_replacement"`
	ReplacedEmployeeID    *uuid.UUID `json:"replaced_employee_id"`
	ReplacementApprovedBy *uuid.UUID `json:"replacement_approved_by"`
	CheckInTime           *time.Time `json:"check_in_time"`
	CheckOutTime          *time.Time `json:"check_out_time"`
	ActualWorkedHours     *float64   `json:"actual_worked_hours"`
	OvertimeHours         *float64   `json:"overtime_hours"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	CreatedBy             *uuid.UUID `json:"created_by"`
}

type TaskBoard struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description"`
	RecurrenceType string     `json:"recurrence_type"` // "daily" or "weekly"
	IsActive       bool       `json:"is_active"`
	DepartmentID   *uuid.UUID `json:"department_id"`
	CreatedBy      *uuid.UUID `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type TaskSchedule struct {
	ID             uuid.UUID  `json:"id"`
	Title          string     `json:"title"`
	Description    *string    `json:"description"`
	ScheduleType   string     `json:"schedule_type"`
	BoardID        *uuid.UUID `json:"board_id"`
	ShiftID        *uuid.UUID `json:"shift_id"`
	Recurrence     string     `json:"recurrence"`
	RecurrenceDays []int      `json:"recurrence_days"` // Array of integers mapping to days
	MaxAssignees   int        `json:"max_assignees"`
	IsActive       bool       `json:"is_active"`
	CreatedBy      *uuid.UUID `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type TaskAssignment struct {
	ID           uuid.UUID  `json:"id"`
	ScheduleID   uuid.UUID  `json:"schedule_id"`
	EmployeeID   uuid.UUID  `json:"employee_id"`
	AssignedDate time.Time  `json:"assigned_date"`
	AssignedBy   *uuid.UUID `json:"assigned_by"`
	CreatedAt    time.Time  `json:"created_at"`
}

type TaskRecurringAssignment struct {
	ID         uuid.UUID  `json:"id"`
	ScheduleID uuid.UUID  `json:"schedule_id"`
	EmployeeID uuid.UUID  `json:"employee_id"`
	DayOfWeek  int        `json:"day_of_week"` // 0=Sun..6=Sat
	AssignedBy *uuid.UUID `json:"assigned_by"`
	CreatedAt  time.Time  `json:"created_at"`
}

type TaskExecution struct {
	ID             uuid.UUID  `json:"id"`
	AssignmentID   uuid.UUID  `json:"assignment_id"`
	Status         string     `json:"status"` // pending, in_progress, completed, cancelled
	CompletionType *string    `json:"completion_type"` // without_issue, with_issue
	StartedAt      *time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	Notes          *string    `json:"notes"`
	Attachments    *string    `json:"attachments"` // JSONB mapping to string usually, or custom type
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TaskHistoryRow — a completed task row for the supervisor history view.
type TaskHistoryRow struct {
	AssignmentID   uuid.UUID  `json:"assignment_id"`
	ExecutionID    uuid.UUID  `json:"execution_id"`
	AssignedDate   time.Time  `json:"assigned_date"`
	TaskTitle      string     `json:"task_title"`
	TaskDesc       *string    `json:"task_description"`
	BoardName      *string    `json:"board_name"`
	EmployeeID     uuid.UUID  `json:"employee_id"`
	EmployeeName   string     `json:"employee_name"`
	EmployeeCode   string     `json:"employee_code"`
	EmployeeProfileImage *string `json:"employee_profile_image"`
	Status         string     `json:"status"`
	CompletionType *string    `json:"completion_type"`
	StartedAt      *time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	Notes          *string    `json:"notes"`
}

type Leave struct {
	ID                   uuid.UUID  `json:"id"`
	EmployeeID           uuid.UUID  `json:"employee_id"`
	LeaveTypeID          uuid.UUID  `json:"leave_type_id"`
	LeaveTypeNameAr      *string    `json:"leave_type_name_ar"`
	LeaveTypeNameEn      *string    `json:"leave_type_name_en"`
	StartDate            time.Time  `json:"start_date"`
	EndDate              time.Time  `json:"end_date"`
	TotalDays            int        `json:"total_days"`
	Reason               *string    `json:"reason"`
	Status               string     `json:"status"`
	AppliedDate          *time.Time `json:"applied_date"` // usually CURRENT_DATE
	ApprovedByTeamLeader *uuid.UUID `json:"approved_by_team_leader"`
	ApprovedByManager    *uuid.UUID `json:"approved_by_manager"`
	RejectionReason      *string    `json:"rejection_reason"`
	Attachments          *string    `json:"attachments"` // JSONB
	StartTime            *string    `json:"start_time"`  // For hourly leaves (HH:MM)
	EndTime              *string    `json:"end_time"`    // For hourly leaves (HH:MM)
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// LeaveApproval tracks individual approval/rejection actions on a leave request.
type LeaveApproval struct {
	ID           uuid.UUID  `json:"id"`
	LeaveID      uuid.UUID  `json:"leave_id"`
	ApproverID   uuid.UUID  `json:"approver_id"`
	ApproverRole string     `json:"approver_role"`
	Action       string     `json:"action"` // approved, rejected
	Notes        *string    `json:"notes"`
	CreatedAt    time.Time  `json:"created_at"`
}

// LeaveHistoryRow is a rich view for the leave history page.
type LeaveHistoryRow struct {
	LeaveID       uuid.UUID  `json:"leave_id"`
	EmployeeName  string     `json:"employee_name"`
	EmployeeCode  string     `json:"employee_code"`
	EmployeeProfileImage *string `json:"employee_profile_image"`
	LeaveTypeID   uuid.UUID  `json:"leave_type_id"`
	LeaveTypeNameAr *string  `json:"leave_type_name_ar"`
	LeaveTypeNameEn *string  `json:"leave_type_name_en"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       time.Time  `json:"end_date"`
	TotalDays     int        `json:"total_days"`
	Reason        *string    `json:"reason"`
	Status        string     `json:"status"`
	AppliedDate   *time.Time `json:"applied_date"`
	RejectionReason *string  `json:"rejection_reason"`
	Approvals     []LeaveApprovalDetail `json:"approvals"`
}

// LeaveApprovalDetail is a single approval action with approver name.
type LeaveApprovalDetail struct {
	ApproverName string    `json:"approver_name"`
	ApproverRole string    `json:"approver_role"`
	Action       string    `json:"action"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
}

// PendingLeaveRich is a pending leave with employee details for the approval dashboard.
type PendingLeaveRich struct {
	ID             uuid.UUID  `json:"id"`
	EmployeeID     uuid.UUID  `json:"employee_id"`
	LeaveTypeID    uuid.UUID  `json:"leave_type_id"`
	LeaveTypeNameAr *string   `json:"leave_type_name_ar"`
	LeaveTypeNameEn *string   `json:"leave_type_name_en"`
	StartDate      time.Time  `json:"start_date"`
	EndDate        time.Time  `json:"end_date"`
	TotalDays      int        `json:"total_days"`
	Reason         *string    `json:"reason"`
	Status         string     `json:"status"`
	AppliedDate    *time.Time `json:"applied_date"`
	EmployeeName   string     `json:"employee_name"`
	EmployeeCode   string     `json:"employee_code"`
	EmployeeProfileImage *string `json:"employee_profile_image"`
	DefaultShiftID string     `json:"default_shift_id"`
	ShiftName      string     `json:"shift_name"`
	ShiftCode      string     `json:"shift_code"`
	DepartmentName string     `json:"department_name"`
	TLApprovals    int        `json:"tl_approvals"`
	TotalTLs       int        `json:"total_tls"`
	StartTime      *string    `json:"start_time"`
	EndTime        *string    `json:"end_time"`
}

type ShiftSwap struct {
	ID                   uuid.UUID  `json:"id"`
	RequesterID          uuid.UUID  `json:"requester_id"`
	RequesterName        string     `json:"requester_name"`
	RequesterProfileImage *string   `json:"requester_profile_image"`
	TargetEmployeeID     uuid.UUID  `json:"target_employee_id"`
	TargetEmployeeName   string     `json:"target_employee_name"`
	TargetProfileImage   *string    `json:"target_profile_image"`
	ShiftDate            time.Time  `json:"shift_date"`
	ShiftID              uuid.UUID  `json:"shift_id"`
	Reason               *string    `json:"reason"`
	Status               string     `json:"status"`
	ApprovedByTeamLeader *uuid.UUID `json:"approved_by_team_leader"`
	ApprovedByManager    *uuid.UUID `json:"approved_by_manager"`
	ApprovalDate         *time.Time `json:"approval_date"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type Permission struct {
	ID                   uuid.UUID `json:"id"`
	Role                 string    `json:"role"`
	PermissionName       string    `json:"permission_name"`
	Resource             string    `json:"resource"`
	CanView              bool      `json:"can_view"`
	CanCreate            bool      `json:"can_create"`
	CanEdit              bool      `json:"can_edit"`
	CanDelete            bool      `json:"can_delete"`
	CanApprove           bool      `json:"can_approve"`
	DepartmentRestricted bool      `json:"department_restricted"`
}

type Notification struct {
	ID                uuid.UUID  `json:"id"`
	RecipientID       uuid.UUID  `json:"recipient_id"`
	SenderID          *uuid.UUID `json:"sender_id"`
	Type              string     `json:"type"`
	Title             string     `json:"title"`
	Message           *string    `json:"message"`
	RelatedEntityType *string    `json:"related_entity_type"`
	RelatedEntityID   *uuid.UUID `json:"related_entity_id"`
	Priority          string     `json:"priority"`
	IsRead            bool       `json:"is_read"`
	ReadAt            *time.Time `json:"read_at"`
	ActionUrl         *string    `json:"action_url"`
	CreatedAt         time.Time  `json:"created_at"`
}

type AuditLog struct {
	ID         uuid.UUID  `json:"id"`
	EmployeeID *uuid.UUID `json:"employee_id"`
	Action     string     `json:"action"`
	TableName  string     `json:"table_name"`
	RecordID   *uuid.UUID `json:"record_id"`
	OldData    *string    `json:"old_data"` // JSONB
	NewData    *string    `json:"new_data"` // JSONB
	IPAddress  *string    `json:"ip_address"` // INET maps to string
	UserAgent  *string    `json:"user_agent"`
	CreatedAt  time.Time  `json:"created_at"`
}

// SwapEligibleEmployee represents an employee available for a shift swap
type SwapEligibleEmployee struct {
	Employee
	IsOff bool `json:"is_off"`
}

// ShiftCoverage provides staffing metrics for a specific shift on a specific date
type ShiftCoverage struct {
	ShiftID      uuid.UUID `json:"shift_id"`
	ShiftDate    time.Time `json:"shift_date"`
	TotalAssigned int      `json:"total_assigned"`
	TotalWorking  int      `json:"total_working"`
	TotalOff      int      `json:"total_off"`
	TotalOnLeave  int      `json:"total_on_leave"`
}

// ═══════════════════════════════════════════════════════════
// Task View Models (Rich Responses)
// ═══════════════════════════════════════════════════════════

// MyTaskRow — one task for one day for the employee's weekly view.
type MyTaskRow struct {
	AssignmentID   uuid.UUID  `json:"assignment_id"`
	AssignedDate   time.Time  `json:"assigned_date"`
	TaskTitle      string     `json:"task_title"`
	TaskDesc       *string    `json:"task_description"`
	BoardName      *string    `json:"board_name"`
	ShiftName      *string    `json:"shift_name"`
	ShiftCode      *string    `json:"shift_code"`
	ShiftColor     *string    `json:"shift_color"`
	ExecutionID    *uuid.UUID `json:"execution_id"`
	Status         string     `json:"status"`          // pending, in_progress, completed
	CompletionType *string    `json:"completion_type"` // without_issue, with_issue
	StartedAt      *time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	Notes          *string    `json:"notes"`
}

// BoardViewRow — one cell in the board tracker grid (employee × day × task).
type BoardViewRow struct {
	EmployeeID   uuid.UUID  `json:"employee_id"`
	EmployeeName string     `json:"employee_name"`
	EmployeeCode string     `json:"employee_code"`
	DayOfWeek    int        `json:"day_of_week"` // 0=Sun..6=Sat
	AssignedDate *time.Time `json:"assigned_date"`
	TaskID       *uuid.UUID `json:"task_id"`
	TaskTitle    *string    `json:"task_title"`
	AssignmentID *uuid.UUID `json:"assignment_id"`
	ExecutionID  *uuid.UUID `json:"execution_id"`
	Status       *string    `json:"status"`       // pending, in_progress, completed
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
}

// TaskBoardStats — aggregate completion stats for a board.
type TaskBoardStats struct {
	BoardID         uuid.UUID `json:"board_id"`
	BoardName       string    `json:"board_name"`
	TotalAssigned   int       `json:"total_assigned"`
	TotalPending    int       `json:"total_pending"`
	TotalInProgress int       `json:"total_in_progress"`
	TotalCompleted  int       `json:"total_completed"`
	CompletionPct   float64   `json:"completion_pct"`
}

type HelpDocument struct {
	ID           uuid.UUID  `json:"id"`
	DepartmentID uuid.UUID  `json:"department_id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	CreatedBy    *uuid.UUID `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	
	// Virtual field for frontend
	AccessLevel *string `json:"access_level,omitempty"`
}

type HelpDocumentAccess struct {
	ID          uuid.UUID  `json:"id"`
	DocumentID  uuid.UUID  `json:"document_id"`
	EmployeeID  uuid.UUID  `json:"employee_id"`
	AccessLevel string     `json:"access_level"`
	GrantedBy   *uuid.UUID `json:"granted_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

type PushSubscription struct {
	ID         uuid.UUID `json:"id"`
	EmployeeID uuid.UUID `json:"employee_id"`
	Endpoint   string    `json:"endpoint"`
	P256dh     string    `json:"p256dh"`
	Auth       string    `json:"auth"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
