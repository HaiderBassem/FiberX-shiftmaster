package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

// TaskHandler handles task schedule, assignment, execution, and board endpoints.
type TaskHandler struct {
	taskSvc *service.TaskService
}

func NewTaskHandler(taskSvc *service.TaskService) *TaskHandler {
	return &TaskHandler{taskSvc: taskSvc}
}

// ═══════════════════════════════════════════
// Board Endpoints
// ═══════════════════════════════════════════

func (h *TaskHandler) ListBoards(c *gin.Context) {
	deptID := getDepartmentID(c)
	boards, err := h.taskSvc.GetAllBoards(c.Request.Context(), deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if boards == nil {
		boards = []models.TaskBoard{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": boards, "meta": gin.H{"count": len(boards)}})
}

type createBoardRequest struct {
	Name           string  `json:"name" binding:"required"`
	Description    *string `json:"description"`
	RecurrenceType string  `json:"recurrence_type" binding:"required"`
}

func (h *TaskHandler) CreateBoard(c *gin.Context) {
	var req createBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	createdByStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(createdByStr.(string))
	deptID := getDepartmentID(c)

	b := &models.TaskBoard{
		Name:           req.Name,
		Description:    req.Description,
		RecurrenceType: req.RecurrenceType,
		IsActive:       true,
		DepartmentID:   deptID,
		CreatedBy:      &empID,
	}
	if err := h.taskSvc.CreateBoard(c.Request.Context(), b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": b})
}

type updateBoardRequest struct {
	Name           string  `json:"name" binding:"required"`
	Description    *string `json:"description"`
	RecurrenceType string  `json:"recurrence_type" binding:"required"`
	IsActive       bool    `json:"is_active"`
}

func (h *TaskHandler) UpdateBoard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid board ID"})
		return
	}
	var req updateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	b := &models.TaskBoard{
		ID: id, Name: req.Name, Description: req.Description,
		RecurrenceType: req.RecurrenceType, IsActive: req.IsActive,
	}
	if err := h.taskSvc.UpdateBoard(c.Request.Context(), b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": b})
}

func (h *TaskHandler) DeleteBoard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid board ID"})
		return
	}
	if err := h.taskSvc.DeleteBoard(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "board deleted"}})
}

func (h *TaskHandler) GetBoardView(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid board ID"})
		return
	}
	var shiftID *uuid.UUID
	if sid := c.Query("shift_id"); sid != "" {
		parsed, err := uuid.Parse(sid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_id"})
			return
		}
		shiftID = &parsed
	}

	var fromDate *time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		d, err := parseTime(fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid from date"})
			return
		}
		fromDate = &d
	}

	var toDate *time.Time
	if toStr := c.Query("to"); toStr != "" {
		d, err := parseTime(toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid to date"})
			return
		}
		toDate = &d
	}

	rows, err := h.taskSvc.GetBoardView(c.Request.Context(), boardID, shiftID, fromDate, toDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if rows == nil {
		rows = []models.BoardViewRow{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows, "meta": gin.H{"count": len(rows)}})
}

// GetBoardStats returns completion analytics for all active boards.
func (h *TaskHandler) GetBoardStats(c *gin.Context) {
	deptID := getDepartmentID(c)
	stats, err := h.taskSvc.GetBoardStats(c.Request.Context(), deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if stats == nil {
		stats = []models.TaskBoardStats{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": stats, "meta": gin.H{"count": len(stats)}})
}

// GetBoardEligibleEmployees returns employees eligible for task assignment (role=employee only).
func (h *TaskHandler) GetBoardEligibleEmployees(c *gin.Context) {
	var shiftID *uuid.UUID
	if sid := c.Query("shift_id"); sid != "" {
		parsed, err := uuid.Parse(sid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_id"})
			return
		}
		shiftID = &parsed
	}
	deptID := getDepartmentID(c)
	employees, err := h.taskSvc.GetBoardEligibleEmployees(c.Request.Context(), shiftID, nil, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if employees == nil {
		employees = []models.Employee{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": employees, "meta": gin.H{"count": len(employees)}})
}

// ═══════════════════════════════════════════
// Employee Weekly Tasks
// ═══════════════════════════════════════════

// MyWeeklyTasks returns the authenticated employee's tasks for an entire week.
func (h *TaskHandler) MyWeeklyTasks(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	weekStartStr := c.Query("week_start")
	if weekStartStr == "" {
		// Default to current week (Sunday start)
		now := time.Now()
		offset := int(now.Weekday())
		weekStart := now.AddDate(0, 0, -offset)
		weekStartStr = weekStart.Format("2006-01-02")
	}

	weekStart, err := parseTime(weekStartStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid week_start format"})
		return
	}
	weekEnd := weekStart.AddDate(0, 0, 6)

	tasks, err := h.taskSvc.GetMyWeeklyTasks(c.Request.Context(), empID, weekStart, weekEnd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if tasks == nil {
		tasks = []models.MyTaskRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tasks,
		"meta": gin.H{
			"count":      len(tasks),
			"week_start": weekStart.Format("2006-01-02"),
			"week_end":   weekEnd.Format("2006-01-02"),
		},
	})
}

// ═══════════════════════════════════════════
// Task Schedule Endpoints
// ═══════════════════════════════════════════

func (h *TaskHandler) ListSchedules(c *gin.Context) {
	ctx := c.Request.Context()
	var schedules []models.TaskSchedule
	var err error

	switch {
	case c.Query("type") != "":
		schedules, err = h.taskSvc.GetSchedulesByType(ctx, c.Query("type"))
	case c.Query("board_id") != "":
		boardID, parseErr := uuid.Parse(c.Query("board_id"))
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid board_id"})
			return
		}
		schedules, err = h.taskSvc.GetSchedulesByBoard(ctx, boardID)
	case c.Query("shift_id") != "":
		shiftID, parseErr := uuid.Parse(c.Query("shift_id"))
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_id"})
			return
		}
		schedules, err = h.taskSvc.GetSchedulesByShift(ctx, shiftID)
	default:
		deptID := getDepartmentID(c)
		schedules, err = h.taskSvc.GetAllSchedules(ctx, deptID)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if schedules == nil {
		schedules = []models.TaskSchedule{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": schedules, "meta": gin.H{"count": len(schedules)}})
}

func (h *TaskHandler) EligibleAssignees(c *gin.Context) {
	shiftID, err := uuid.Parse(c.Query("shift_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_id"})
		return
	}
	date, err := parseTime(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format"})
		return
	}
	employees, err := h.taskSvc.GetEligibleAssignees(c.Request.Context(), shiftID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if employees == nil {
		employees = []models.Employee{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": employees, "meta": gin.H{"count": len(employees)}})
}

type createTaskScheduleRequest struct {
	Title          string     `json:"title" binding:"required"`
	Description    *string    `json:"description"`
	ScheduleType   string     `json:"schedule_type"`
	BoardID        *uuid.UUID `json:"board_id"`
	ShiftID        *uuid.UUID `json:"shift_id"`
	Recurrence     string     `json:"recurrence" binding:"required"`
	RecurrenceDays []int      `json:"recurrence_days"`
	MaxAssignees   int        `json:"max_assignees"`
	IsActive       bool       `json:"is_active"`
}

func (h *TaskHandler) CreateSchedule(c *gin.Context) {
	var req createTaskScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	createdByStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(createdByStr.(string))

	ts := &models.TaskSchedule{
		Title:          req.Title,
		Description:    req.Description,
		ScheduleType:   req.ScheduleType,
		BoardID:        req.BoardID,
		ShiftID:        req.ShiftID,
		Recurrence:     req.Recurrence,
		RecurrenceDays: req.RecurrenceDays,
		MaxAssignees:   req.MaxAssignees,
		IsActive:       true,
		CreatedBy:      &empID,
	}
	if err := h.taskSvc.CreateSchedule(c.Request.Context(), ts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": ts})
}

func (h *TaskHandler) UpdateSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid schedule ID"})
		return
	}
	var req createTaskScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	ts := &models.TaskSchedule{
		ID: id, Title: req.Title, Description: req.Description, ScheduleType: req.ScheduleType,
		BoardID: req.BoardID, ShiftID: req.ShiftID, Recurrence: req.Recurrence,
		RecurrenceDays: req.RecurrenceDays, MaxAssignees: req.MaxAssignees, IsActive: req.IsActive,
	}
	if err := h.taskSvc.UpdateSchedule(c.Request.Context(), ts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": ts})
}

type toggleActiveRequest struct {
	IsActive bool `json:"is_active"`
}

func (h *TaskHandler) ToggleActive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid schedule ID"})
		return
	}
	var req toggleActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	if err := h.taskSvc.ToggleActive(c.Request.Context(), id, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "active status updated"}})
}

func (h *TaskHandler) DeleteSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid schedule ID"})
		return
	}
	if err := h.taskSvc.DeleteSchedule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "schedule deleted"}})
}

// ═══════════════════════════════════════════
// Task Assignment Endpoints
// ═══════════════════════════════════════════

type assignTaskRequest struct {
	ScheduleID   string `json:"schedule_id" binding:"required"`
	EmployeeID   string `json:"employee_id" binding:"required"`
	AssignedDate string `json:"assigned_date" binding:"required"`
}

func (h *TaskHandler) Assign(c *gin.Context) {
	var req assignTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	scheduleID, _ := uuid.Parse(req.ScheduleID)
	employeeID, _ := uuid.Parse(req.EmployeeID)
	assignedDate, err := parseTime(req.AssignedDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid assigned_date"})
		return
	}
	assignedByStr, _ := c.Get("employee_id")
	assignedBy, _ := uuid.Parse(assignedByStr.(string))

	ta := &models.TaskAssignment{
		ScheduleID: scheduleID, EmployeeID: employeeID,
		AssignedDate: assignedDate, AssignedBy: &assignedBy,
	}
	if err := h.taskSvc.AssignTask(c.Request.Context(), ta); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": ta})
}

// ═══════════════════════════════════════════
// Recurring Assignment Endpoints
// ═══════════════════════════════════════════

type recurringAssignRequest struct {
	ScheduleID string `json:"schedule_id" binding:"required"`
	EmployeeID string `json:"employee_id" binding:"required"`
	DayOfWeek  *int   `json:"day_of_week" binding:"required"`
}

func (h *TaskHandler) RecurringAssign(c *gin.Context) {
	var req recurringAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	scheduleID, _ := uuid.Parse(req.ScheduleID)
	employeeID, _ := uuid.Parse(req.EmployeeID)
	assignedByStr, _ := c.Get("employee_id")
	assignedBy, _ := uuid.Parse(assignedByStr.(string))

	ra := &models.TaskRecurringAssignment{
		ScheduleID: scheduleID,
		EmployeeID: employeeID,
		DayOfWeek:  *req.DayOfWeek,
		AssignedBy: &assignedBy,
	}
	if err := h.taskSvc.CreateRecurringAssignment(c.Request.Context(), ra); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": ra})
}

func (h *TaskHandler) DeleteRecurringAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid recurring assignment ID"})
		return
	}
	if err := h.taskSvc.DeleteRecurringAssignment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "recurring assignment removed"}})
}

type deleteRecurringByKeyRequest struct {
	ScheduleID string `json:"schedule_id" binding:"required"`
	EmployeeID string `json:"employee_id" binding:"required"`
	DayOfWeek  *int   `json:"day_of_week" binding:"required"`
}

func (h *TaskHandler) DeleteRecurringAssignmentByKey(c *gin.Context) {
	var req deleteRecurringByKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	scheduleID, _ := uuid.Parse(req.ScheduleID)
	employeeID, _ := uuid.Parse(req.EmployeeID)
	if err := h.taskSvc.DeleteRecurringAssignmentByKey(c.Request.Context(), scheduleID, employeeID, *req.DayOfWeek); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "recurring assignment removed"}})
}

func (h *TaskHandler) ListRecurringByBoard(c *gin.Context) {
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid board ID"})
		return
	}
	assignments, err := h.taskSvc.GetRecurringAssignmentsByBoard(c.Request.Context(), boardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if assignments == nil {
		assignments = []models.TaskRecurringAssignment{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": assignments, "meta": gin.H{"count": len(assignments)}})
}

func (h *TaskHandler) DeleteAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid assignment ID"})
		return
	}
	if err := h.taskSvc.DeleteAssignment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "assignment removed"}})
}

func (h *TaskHandler) DailyAssignments(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := parseTime(dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format"})
		return
	}
	assignments, err := h.taskSvc.GetDailyAssignments(c.Request.Context(), date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if assignments == nil {
		assignments = []models.TaskAssignment{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": assignments, "meta": gin.H{"count": len(assignments)}})
}

func (h *TaskHandler) MyTasks(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := parseTime(dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format"})
		return
	}
	assignments, err := h.taskSvc.GetEmployeeTasks(c.Request.Context(), empID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if assignments == nil {
		assignments = []models.TaskAssignment{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": assignments, "meta": gin.H{"count": len(assignments)}})
}

// ═══════════════════════════════════════════
// Task Execution Endpoints
// ═══════════════════════════════════════════

// StartExecution marks a task as in_progress with a started_at timestamp.
func (h *TaskHandler) StartExecution(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid execution ID"})
		return
	}
	if err := h.taskSvc.StartTask(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "task started"}})
}

type updateExecutionStatusRequest struct {
	Status string  `json:"status" binding:"required"`
	Notes  *string `json:"notes"`
}

func (h *TaskHandler) UpdateExecutionStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid execution ID"})
		return
	}
	var req updateExecutionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}
	if err := h.taskSvc.UpdateTaskStatus(c.Request.Context(), id, req.Status, req.Notes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "execution status updated"}})
}

type completeExecutionRequest struct {
	CompletionType string  `json:"completion_type" binding:"required"`
	Notes          *string `json:"notes"`
}

func (h *TaskHandler) CompleteExecution(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid execution ID"})
		return
	}
	var req completeExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "completion_type is required (without_issue or with_issue)"})
		return
	}
	if err := h.taskSvc.CompleteTask(c.Request.Context(), id, req.CompletionType, req.Notes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "task completed", "completion_type": req.CompletionType}})
}

// TaskHistory returns all task executions for a given date (for supervisors).
func (h *TaskHandler) TaskHistory(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := parseTime(dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format"})
		return
	}
	var boardID *uuid.UUID
	if bid := c.Query("board_id"); bid != "" {
		parsed, err := uuid.Parse(bid)
		if err == nil {
			boardID = &parsed
		}
	}
	deptID := getDepartmentID(c)
	history, err := h.taskSvc.GetTaskHistory(c.Request.Context(), date, boardID, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if history == nil {
		history = []models.TaskHistoryRow{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": history, "meta": gin.H{"count": len(history), "date": dateStr}})
}
