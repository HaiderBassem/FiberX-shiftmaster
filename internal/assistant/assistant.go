// Package assistant is the AI workforce assistant: a permission-aware natural
// language layer over ShiftMaster's existing domain services.
//
// Architecture, in one paragraph: the language model plans; the application
// decides. The model receives a system prompt, the conversation, and a tool
// catalogue filtered by the caller's role. Every tool call the model makes is
// executed by this package against the same repositories and services the
// REST handlers use, with the authenticated actor derived server-side from the
// JWT — never from model output. Reads run immediately; anything that would
// change state only ever creates a *pending action*: a validated, frozen
// operation that a human must approve through a separate authenticated
// endpoint before an executor runs it. The model has no tool that approves,
// no tool that executes, and no tool that touches SQL.
package assistant

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/assistant/llm"
	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
	"shiftmaster-backend/internal/temporal"
	"shiftmaster-backend/pkg/database"
)

// Deps carries every domain dependency the assistant's tools and executors
// use. All of them are the application's existing services and repositories —
// the assistant deliberately owns no business logic of its own.
type Deps struct {
	Cfg config.AssistantConfig
	LLM llm.Client
	// Runtime reports the local model's lifecycle state for GET
	// /assistant/status. Nil when the feature is switched off, or in tests
	// that drive a scripted client directly.
	Runtime llm.Prober

	Clock temporal.Clock

	// DB is used for exactly one thing: the assistant's ranked knowledge
	// search, which needs a different query shape from the list endpoints while
	// applying identical access rules. There is no general query path here and
	// the model has no tool that reaches SQL.
	DB *database.DB

	AssistantRepo repository.AssistantRepository

	EmployeeRepo     repository.EmployeeRepository
	DepartmentRepo   repository.DepartmentRepository
	ShiftRepo        repository.ShiftRepository
	ScheduleRepo     repository.ScheduleRepository
	LeaveRepo        repository.LeaveRepository
	LeaveTypeRepo    repository.LeaveTypeRepository
	TaskRepo         repository.TaskRepository
	NotifRepo        repository.NotificationRepository
	AnnouncementRepo repository.AnnouncementRepository
	HandoverRepo     repository.HandoverRepository
	TicketRepo       repository.TicketRepository
	ItemReqRepo      repository.ItemRequestRepository
	HelpDocRepo      *repository.HelpDocumentRepository
	FiberxRepo       *repository.FiberxDataRepository
	InfoTableRepo    *repository.InfoTableRepository
	ServiceRepo      repository.ServiceRepository
	ProvinceRepo     repository.ProvinceRepository
	AuditLogRepo     repository.AuditLogRepository

	AuthService      *service.AuthService
	LeaveService     *service.LeaveService
	ScheduleService  *service.ScheduleService
	TaskService      *service.TaskService
	SwapService      *service.SwapService
	InfoTableService *service.InfoTableService
	HelpDocService   *service.HelpDocumentService
	FiberxService    *service.FiberxDataService
	ItemReqService   *service.ItemRequestService
	ProvinceService  service.ProvinceService
	AuditService     *service.AuditService
}

// Actor is the authenticated caller, loaded fresh from the database at the
// start of every assistant request and every action decision. Loading fresh —
// rather than trusting JWT claims — means a deactivation, lockout, role change
// or department move takes effect on the very next call.
type Actor struct {
	Employee *models.Employee
	// ManagedDepartments is populated for managers from department_managers,
	// the same join table DepartmentContext validates against.
	ManagedDepartments []models.Department
}

func (a *Actor) ID() uuid.UUID      { return a.Employee.ID }
func (a *Actor) Role() string       { return a.Employee.Role }
func (a *Actor) DeptID() *uuid.UUID { return a.Employee.DepartmentID }
func (a *Actor) FirstName() string  { return a.Employee.FirstName }

// IsSupervisor mirrors the router's supervisor group membership.
func (a *Actor) IsSupervisor() bool {
	switch a.Employee.Role {
	case "team_leader", "manager", "admin":
		return true
	}
	return false
}

// CanAccessDepartment answers the same question DepartmentContext answers for
// the REST API: admins anywhere, managers only where department_managers says
// so, everyone else only at home.
func (a *Actor) CanAccessDepartment(dept uuid.UUID) bool {
	switch a.Employee.Role {
	case "admin", "hr":
		return true
	case "manager":
		for _, d := range a.ManagedDepartments {
			if d.ID == dept {
				return true
			}
		}
		// A manager's home department counts even without a join-table row,
		// matching how most services scope managers today.
		return a.Employee.DepartmentID != nil && *a.Employee.DepartmentID == dept
	default:
		return a.Employee.DepartmentID != nil && *a.Employee.DepartmentID == dept
	}
}

// LoadActor authenticates-by-database: the employee must still be active and
// unlocked (the same rule token refresh applies). Conversation context, model
// output and stale JWTs cannot override what this returns.
func (d *Deps) LoadActor(ctx context.Context, employeeID uuid.UUID) (*Actor, error) {
	emp, err := d.AuthService.GetAuthorizedEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("account is not authorized: %w", err)
	}
	actor := &Actor{Employee: emp}
	if emp.Role == "manager" {
		depts, err := d.DepartmentRepo.GetByManagerID(ctx, emp.ID)
		if err != nil {
			return nil, fmt.Errorf("resolve managed departments: %w", err)
		}
		actor.ManagedDepartments = depts
	}
	return actor, nil
}
