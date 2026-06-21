package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/service"
)

// ScheduleHandler handles schedule generation, publishing, shifts, and replacements.
type ScheduleHandler struct {
	scheduleSvc *service.ScheduleService
}

func NewScheduleHandler(scheduleSvc *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleSvc: scheduleSvc}
}

type generateScheduleRequest struct {
	WeekStart string `json:"week_start" binding:"required"` // YYYY-MM-DD
}

// Generate creates a weekly schedule from templates.
func (h *ScheduleHandler) Generate(c *gin.Context) {
	var req generateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	weekStart, err := parseTime(req.WeekStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid week_start format, expected YYYY-MM-DD"})
		return
	}

	createdByStr, _ := c.Get("employee_id")
	createdBy, _ := uuid.Parse(createdByStr.(string))

	ws, err := h.scheduleSvc.GenerateWeeklySchedule(c.Request.Context(), weekStart, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": ws})
}

// Publish publishes a draft schedule.
func (h *ScheduleHandler) Publish(c *gin.Context) {
	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid schedule ID"})
		return
	}

	publishedByStr, _ := c.Get("employee_id")
	publishedBy, _ := uuid.Parse(publishedByStr.(string))

	if err := h.scheduleSvc.PublishSchedule(c.Request.Context(), scheduleID, publishedBy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "schedule published"}})
}

// DailyShifts returns all shifts for a specific date.
func (h *ScheduleHandler) DailyShifts(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := parseTime(dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format"})
		return
	}

	deptID := getDepartmentID(c)
	shifts, err := h.scheduleSvc.GetDailyShifts(c.Request.Context(), date, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": shifts, "meta": gin.H{"count": len(shifts)}})
}

// EmployeeShifts returns an employee's shifts for a date range.
func (h *ScheduleHandler) EmployeeShifts(c *gin.Context) {
	empID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee ID"})
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "'from' and 'to' query params required (YYYY-MM-DD)"})
		return
	}

	from, err := parseTime(fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid 'from' date"})
		return
	}
	to, err := parseTime(toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid 'to' date"})
		return
	}

	shifts, err := h.scheduleSvc.GetEmployeeShifts(c.Request.Context(), empID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": shifts, "meta": gin.H{"count": len(shifts)}})
}

// DepartmentShifts returns the shifts for the user's department for a date range.
func (h *ScheduleHandler) DepartmentShifts(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "'from' and 'to' query params required (YYYY-MM-DD)"})
		return
	}

	from, err := parseTime(fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid 'from' date"})
		return
	}
	to, err := parseTime(toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid 'to' date"})
		return
	}

	deptID := getDepartmentID(c)
	if deptID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "user has no department"})
		return
	}

	shifts, err := h.scheduleSvc.GetDepartmentShiftsInRange(c.Request.Context(), from, to, *deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": shifts, "meta": gin.H{"count": len(shifts)}})
}

// CheckIn records an employee's check-in.
func (h *ScheduleHandler) CheckIn(c *gin.Context) {
	shiftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift ID"})
		return
	}

	if err := h.scheduleSvc.CheckIn(c.Request.Context(), shiftID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "checked in"}})
}

// CheckOut records an employee's check-out.
func (h *ScheduleHandler) CheckOut(c *gin.Context) {
	shiftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift ID"})
		return
	}

	if err := h.scheduleSvc.CheckOut(c.Request.Context(), shiftID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "checked out"}})
}

// AvailableReplacements returns employees available for replacement.
func (h *ScheduleHandler) AvailableReplacements(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := parseTime(dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format"})
		return
	}

	deptID := getDepartmentID(c)

	employees, err := h.scheduleSvc.GetAvailableReplacements(c.Request.Context(), date, deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": employees, "meta": gin.H{"count": len(employees)}})
}

type assignReplacementRequest struct {
	ReplacementEmployeeID string `json:"replacement_employee_id" binding:"required"`
}

type setEmployeeShiftRequest struct {
	EmployeeID   string  `json:"employee_id" binding:"required"`
	ShiftDate    string  `json:"shift_date" binding:"required"` // YYYY-MM-DD
	ShiftStatus  string  `json:"shift_status" binding:"required"`
	ShiftID      *string `json:"shift_id"`
	LeaveReason  *string `json:"leave_reason"`
}

// SetEmployeeShift upserts one employee shift row for the given date.
func (h *ScheduleHandler) SetEmployeeShift(c *gin.Context) {
	var req setEmployeeShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	employeeID, err := uuid.Parse(req.EmployeeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid employee_id"})
		return
	}

	shiftDate, err := parseTime(req.ShiftDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_date format"})
		return
	}

	var shiftID *uuid.UUID
	if req.ShiftID != nil && *req.ShiftID != "" {
		parsed, err := uuid.Parse(*req.ShiftID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_id"})
			return
		}
		shiftID = &parsed
	}

	createdByStr, _ := c.Get("employee_id")
	createdBy, _ := uuid.Parse(createdByStr.(string))
	roleVal, _ := c.Get("role")
	roleStr, _ := roleVal.(string)

	es, err := h.scheduleSvc.SetEmployeeShift(c.Request.Context(), employeeID, shiftDate, shiftID, req.ShiftStatus, req.LeaveReason, createdBy, roleStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": es})
}

// AssignReplacement assigns a replacement employee for a shift.
func (h *ScheduleHandler) AssignReplacement(c *gin.Context) {
	shiftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift ID"})
		return
	}

	var req assignReplacementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	replacementID, err := uuid.Parse(req.ReplacementEmployeeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid replacement_employee_id"})
		return
	}

	approvedByStr, _ := c.Get("employee_id")
	approvedBy, _ := uuid.Parse(approvedByStr.(string))

	if err := h.scheduleSvc.AssignReplacement(c.Request.Context(), shiftID, replacementID, approvedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "replacement assigned"}})
}

// DeleteEmployeeShift deletes an employee shift record (e.g. removing an off-day).
func (h *ScheduleHandler) DeleteEmployeeShift(c *gin.Context) {
	shiftID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift ID"})
		return
	}

	if err := h.scheduleSvc.DeleteEmployeeShift(c.Request.Context(), shiftID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "employee shift deleted"}})
}
