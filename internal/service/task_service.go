package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// TaskService handles task scheduling, assignment, execution, and boards.
type TaskService struct {
	taskRepo     repository.TaskRepository
	boardRepo    repository.BoardRepository
	employeeRepo repository.EmployeeRepository
	scheduleRepo repository.ScheduleRepository
}

func NewTaskService(taskRepo repository.TaskRepository, boardRepo repository.BoardRepository, employeeRepo repository.EmployeeRepository, scheduleRepo repository.ScheduleRepository) *TaskService {
	return &TaskService{
		taskRepo:     taskRepo,
		boardRepo:    boardRepo,
		employeeRepo: employeeRepo,
		scheduleRepo: scheduleRepo,
	}
}

// ═══════════════════════════════════════════
// Boards
// ═══════════════════════════════════════════

func (s *TaskService) GetAllBoards(ctx context.Context, departmentID *uuid.UUID) ([]models.TaskBoard, error) {
	return s.boardRepo.GetAll(ctx, departmentID)
}

func (s *TaskService) GetBoardByID(ctx context.Context, id uuid.UUID) (*models.TaskBoard, error) {
	return s.boardRepo.GetByID(ctx, id)
}

func (s *TaskService) CreateBoard(ctx context.Context, b *models.TaskBoard) error {
	validTypes := map[string]bool{"daily": true, "weekly": true}
	if !validTypes[b.RecurrenceType] {
		return fmt.Errorf("invalid recurrence type: %s", b.RecurrenceType)
	}
	return s.boardRepo.Create(ctx, b)
}

func (s *TaskService) UpdateBoard(ctx context.Context, b *models.TaskBoard) error {
	validTypes := map[string]bool{"daily": true, "weekly": true}
	if !validTypes[b.RecurrenceType] {
		return fmt.Errorf("invalid recurrence type: %s", b.RecurrenceType)
	}
	return s.boardRepo.Update(ctx, b)
}

func (s *TaskService) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	return s.boardRepo.Delete(ctx, id)
}

func (s *TaskService) GetBoardView(ctx context.Context, boardID uuid.UUID, shiftID *uuid.UUID, fromDate *time.Time, toDate *time.Time) ([]models.BoardViewRow, error) {
	// Auto-materialize recurring assignments for the requested date range
	if fromDate != nil && toDate != nil {
		if err := s.taskRepo.MaterializeRecurringForDateRange(ctx, boardID, *fromDate, *toDate); err != nil {
			// Log but don't fail — the view can still return existing assignments
			fmt.Printf("WARNING: failed to materialize recurring assignments for board %s: %v\n", boardID, err)
		}
	}
	return s.taskRepo.GetBoardView(ctx, boardID, shiftID, fromDate, toDate)
}

func (s *TaskService) GetBoardStats(ctx context.Context, departmentID *uuid.UUID) ([]models.TaskBoardStats, error) {
	return s.taskRepo.GetBoardStats(ctx, departmentID)
}

func (s *TaskService) GetBoardEligibleEmployees(ctx context.Context, shiftID *uuid.UUID, date *time.Time) ([]models.Employee, error) {
	return s.taskRepo.GetBoardEligibleEmployees(ctx, shiftID, date)
}

// ═══════════════════════════════════════════
// Recurring Assignments
// ═══════════════════════════════════════════

func (s *TaskService) CreateRecurringAssignment(ctx context.Context, ra *models.TaskRecurringAssignment) error {
	// Validate employee
	emp, err := s.employeeRepo.GetByID(ctx, ra.EmployeeID)
	if err != nil {
		return fmt.Errorf("employee not found: %w", err)
	}
	if emp.Status != "active" {
		return fmt.Errorf("employee is not active")
	}
	if emp.Role != "employee" {
		return fmt.Errorf("cannot assign tasks to role: %s", emp.Role)
	}
	// Validate schedule
	_, err = s.taskRepo.GetScheduleByID(ctx, ra.ScheduleID)
	if err != nil {
		return fmt.Errorf("task schedule not found: %w", err)
	}
	// Validate day_of_week
	if ra.DayOfWeek < 0 || ra.DayOfWeek > 6 {
		return fmt.Errorf("invalid day_of_week: %d (must be 0-6)", ra.DayOfWeek)
	}
	return s.taskRepo.CreateRecurringAssignment(ctx, ra)
}

func (s *TaskService) DeleteRecurringAssignment(ctx context.Context, id uuid.UUID) error {
	return s.taskRepo.DeleteRecurringAssignment(ctx, id)
}

func (s *TaskService) DeleteRecurringAssignmentByKey(ctx context.Context, scheduleID, employeeID uuid.UUID, dayOfWeek int) error {
	return s.taskRepo.DeleteRecurringAssignmentByKey(ctx, scheduleID, employeeID, dayOfWeek)
}

func (s *TaskService) GetRecurringAssignmentsByBoard(ctx context.Context, boardID uuid.UUID) ([]models.TaskRecurringAssignment, error) {
	return s.taskRepo.GetRecurringAssignmentsByBoard(ctx, boardID)
}

// ═══════════════════════════════════════════
// Employee Weekly View
// ═══════════════════════════════════════════

func (s *TaskService) GetMyWeeklyTasks(ctx context.Context, employeeID uuid.UUID, weekStart, weekEnd time.Time) ([]models.MyTaskRow, error) {
	// Auto-materialize recurring tasks for the employee in the week
	if err := s.taskRepo.MaterializeAllRecurringForEmployee(ctx, employeeID, weekStart, weekEnd); err != nil {
		fmt.Printf("WARNING: failed to materialize recurring assignments for employee %s: %v\n", employeeID, err)
	}
	return s.taskRepo.GetMyWeeklyTasks(ctx, employeeID, weekStart, weekEnd)
}

// ═══════════════════════════════════════════
// Task Schedules
// ═══════════════════════════════════════════

func (s *TaskService) GetSchedulesByType(ctx context.Context, scheduleType string) ([]models.TaskSchedule, error) {
	return s.taskRepo.GetSchedulesByType(ctx, scheduleType)
}

func (s *TaskService) GetSchedulesByShift(ctx context.Context, shiftID uuid.UUID) ([]models.TaskSchedule, error) {
	return s.taskRepo.GetSchedulesByShift(ctx, shiftID)
}

func (s *TaskService) GetSchedulesByBoard(ctx context.Context, boardID uuid.UUID) ([]models.TaskSchedule, error) {
	return s.taskRepo.GetSchedulesByBoard(ctx, boardID)
}

func (s *TaskService) CreateSchedule(ctx context.Context, ts *models.TaskSchedule) error {
	if ts.MaxAssignees < 1 {
		ts.MaxAssignees = 1
	}
	return s.taskRepo.CreateSchedule(ctx, ts)
}

func (s *TaskService) UpdateSchedule(ctx context.Context, ts *models.TaskSchedule) error {
	return s.taskRepo.UpdateSchedule(ctx, ts)
}

func (s *TaskService) ToggleActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	return s.taskRepo.ToggleScheduleActive(ctx, id, isActive)
}

func (s *TaskService) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	return s.taskRepo.DeleteSchedule(ctx, id)
}

func (s *TaskService) GetAllSchedules(ctx context.Context, departmentID *uuid.UUID) ([]models.TaskSchedule, error) {
	return s.taskRepo.GetAllSchedules(ctx, departmentID)
}

// ═══════════════════════════════════════════
// Task Assignments (auto-create execution)
// ═══════════════════════════════════════════

func (s *TaskService) GetEligibleAssignees(ctx context.Context, shiftID uuid.UUID, date time.Time) ([]models.Employee, error) {
	return s.scheduleRepo.GetEligibleAssignees(ctx, shiftID, date)
}

func (s *TaskService) AssignTask(ctx context.Context, ta *models.TaskAssignment) error {
	// Validate employee exists and is eligible for task assignments.
	emp, err := s.employeeRepo.GetByID(ctx, ta.EmployeeID)
	if err != nil {
		return fmt.Errorf("employee not found: %w", err)
	}
	if emp.Status != "active" {
		return fmt.Errorf("employee is not active")
	}
	if emp.Role != "employee" {
		// Task assignments are employee-only.
		return fmt.Errorf("cannot assign tasks to role: %s", emp.Role)
	}
	// Validate schedule exists
	ts, err := s.taskRepo.GetScheduleByID(ctx, ta.ScheduleID)
	if err != nil {
		return fmt.Errorf("task schedule not found: %w", err)
	}

	// If the task schedule is bound to a shift, enforce it matches the employee's default shift.
	// Prefer the employee's actual shift on that date; fallback to default shift.
	if ts.ShiftID != nil {
		dayShift, dayShiftErr := s.scheduleRepo.GetEmployeeShift(ctx, ta.EmployeeID, ta.AssignedDate)
		if dayShiftErr == nil && dayShift != nil && dayShift.ShiftID != nil {
			// If daily shift exists, enforce against it.
			if *dayShift.ShiftID != *ts.ShiftID || dayShift.ShiftStatus != "working" {
				return fmt.Errorf("employee is not scheduled for this task shift on selected date")
			}
		} else {
			// Fallback to default shift for employees without a daily shift row.
			if emp.DefaultShiftID == nil {
				return fmt.Errorf("employee has no shift configured for shift-bound tasks")
			}
			if *emp.DefaultShiftID != *ts.ShiftID {
				return fmt.Errorf("employee shift does not match task schedule shift")
			}
		}
	}

	// Check max assignees per shift on this date.
	// This allows, for example, max_assignees=2 in morning and also 2 in evening.
	var count int
	if emp.DefaultShiftID != nil {
		count, err = s.taskRepo.CountAssignmentsByScheduleDateAndShift(ctx, ta.ScheduleID, ta.AssignedDate, *emp.DefaultShiftID)
	} else {
		count, err = s.taskRepo.CountAssignmentsByScheduleAndDate(ctx, ta.ScheduleID, ta.AssignedDate)
	}
	if err != nil {
		return err
	}
	if count >= ts.MaxAssignees {
		return fmt.Errorf("maximum assignees (%d) already reached for this task in this shift", ts.MaxAssignees)
	}

	// Create assignment
	if err := s.taskRepo.CreateAssignment(ctx, ta); err != nil {
		return err
	}

	// Auto-create execution record with 'pending' status
	te := &models.TaskExecution{
		AssignmentID: ta.ID,
		Status:       "pending",
	}
	if err := s.taskRepo.CreateExecution(ctx, te); err != nil {
		// Not fatal — log it but don't fail the assignment
		fmt.Printf("WARNING: failed to auto-create execution for assignment %s: %v\n", ta.ID, err)
	}

	return nil
}

func (s *TaskService) GetEmployeeTasks(ctx context.Context, employeeID uuid.UUID, date time.Time) ([]models.TaskAssignment, error) {
	// Auto-materialize recurring tasks for the employee on this date
	if err := s.taskRepo.MaterializeAllRecurringForEmployee(ctx, employeeID, date, date); err != nil {
		fmt.Printf("WARNING: failed to materialize recurring assignments for employee %s: %v\n", employeeID, err)
	}
	return s.taskRepo.GetAssignmentsByEmployee(ctx, employeeID, date)
}

func (s *TaskService) GetDailyAssignments(ctx context.Context, date time.Time) ([]models.TaskAssignment, error) {
	return s.taskRepo.GetAssignmentsByDate(ctx, date)
}

func (s *TaskService) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	return s.taskRepo.DeleteAssignment(ctx, id)
}

// ═══════════════════════════════════════════
// Task Executions
// ═══════════════════════════════════════════

// StartTask marks a task execution as in_progress with a timestamp.
func (s *TaskService) StartTask(ctx context.Context, executionID uuid.UUID) error {
	return s.taskRepo.StartExecution(ctx, executionID)
}

// CompleteTask marks a task execution as completed with a timestamp.
func (s *TaskService) CompleteTask(ctx context.Context, executionID uuid.UUID, completionType string, notes *string) error {
	validTypes := map[string]bool{"without_issue": true, "with_issue": true}
	if !validTypes[completionType] {
		return fmt.Errorf("invalid completion_type: %s (must be 'without_issue' or 'with_issue')", completionType)
	}
	return s.taskRepo.CompleteExecution(ctx, executionID, completionType, notes)
}

func (s *TaskService) UpdateTaskStatus(ctx context.Context, executionID uuid.UUID, status string, notes *string) error {
	validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "cancelled": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid task status: %s", status)
	}
	return s.taskRepo.UpdateExecutionStatus(ctx, executionID, status, notes)
}

func (s *TaskService) GetTaskExecution(ctx context.Context, assignmentID uuid.UUID) (*models.TaskExecution, error) {
	return s.taskRepo.GetExecutionByAssignment(ctx, assignmentID)
}

func (s *TaskService) GetTaskHistory(ctx context.Context, date time.Time, boardID *uuid.UUID, departmentID *uuid.UUID) ([]models.TaskHistoryRow, error) {
	return s.taskRepo.GetTaskHistory(ctx, date, boardID, departmentID)
}
