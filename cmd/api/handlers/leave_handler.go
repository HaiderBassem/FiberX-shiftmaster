package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

// LeaveHandler handles leave request endpoints.
type LeaveHandler struct {
	leaveSvc *service.LeaveService
}

func NewLeaveHandler(leaveSvc *service.LeaveService) *LeaveHandler {
	return &LeaveHandler{leaveSvc: leaveSvc}
}

type createLeaveRequest struct {
	LeaveTypeID uuid.UUID `json:"leave_type_id" binding:"required"`
	StartDate   string  `json:"start_date" binding:"required"`
	EndDate     string  `json:"end_date" binding:"required"`
	Reason      *string `json:"reason"`
	Attachments *string `json:"attachments"`
	StartTime   *string `json:"start_time"` // For hourly leaves (HH:MM)
	EndTime     *string `json:"end_time"`   // For hourly leaves (HH:MM)
}

// Request creates a new leave request.
func (h *LeaveHandler) Request(c *gin.Context) {
	var req createLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	startDate, err := parseTime(req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid start_date format"})
		return
	}
	endDate, err := parseTime(req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid end_date format"})
		return
	}

	// Validate hourly leave requires start_time and end_time (can be relaxed if we check it in DB later)
	if req.StartTime != nil && req.EndTime != nil && (*req.StartTime == "" || *req.EndTime == "") {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "start_time and end_time must be valid for hourly leave"})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	leave := &models.Leave{
		EmployeeID:  empID,
		LeaveTypeID: req.LeaveTypeID,
		StartDate:   startDate,
		EndDate:     endDate,
		Reason:      req.Reason,
		Attachments: req.Attachments,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	}

	if err := h.leaveSvc.RequestLeave(c.Request.Context(), leave); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": leave})
}

// MyLeaves returns the authenticated employee's leaves.
func (h *LeaveHandler) MyLeaves(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	leaves, err := h.leaveSvc.GetEmployeeLeaves(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if leaves == nil {
		leaves = []models.Leave{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": leaves, "meta": gin.H{"count": len(leaves)}})
}

// PendingForApproval returns leaves pending for the approver's role and department.
func (h *LeaveHandler) PendingForApproval(c *gin.Context) {
	approverStr, _ := c.Get("employee_id")
	approverID, _ := uuid.Parse(approverStr.(string))

	leaves, err := h.leaveSvc.GetPendingForApproval(c.Request.Context(), approverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if leaves == nil {
		leaves = []models.Leave{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": leaves, "meta": gin.H{"count": len(leaves)}})
}

// PendingRich returns pending leaves with full employee details for the approval dashboard.
func (h *LeaveHandler) PendingRich(c *gin.Context) {
	approverStr, _ := c.Get("employee_id")
	approverID, _ := uuid.Parse(approverStr.(string))

	leaves, err := h.leaveSvc.GetPendingLeavesRich(c.Request.Context(), approverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if leaves == nil {
		leaves = []models.PendingLeaveRich{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": leaves, "meta": gin.H{"count": len(leaves)}})
}

// CoveragePreview returns the staffing coverage for a specific shift and date.
func (h *LeaveHandler) CoveragePreview(c *gin.Context) {
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

	coverage, err := h.leaveSvc.GetShiftCoveragePreview(c.Request.Context(), shiftID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": coverage})
}

// ApproveByTeamLeader approves a leave at the team leader level.
func (h *LeaveHandler) ApproveByTeamLeader(c *gin.Context) {
	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid leave ID"})
		return
	}

	approverStr, _ := c.Get("employee_id")
	approverID, _ := uuid.Parse(approverStr.(string))

	if err := h.leaveSvc.ApproveByTeamLeader(c.Request.Context(), leaveID, approverID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "leave approved by team leader"}})
}

// ApproveByManager gives final approval to a leave.
func (h *LeaveHandler) ApproveByManager(c *gin.Context) {
	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid leave ID"})
		return
	}

	approverStr, _ := c.Get("employee_id")
	approverID, _ := uuid.Parse(approverStr.(string))

	if err := h.leaveSvc.ApproveByManager(c.Request.Context(), leaveID, approverID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "leave approved by manager"}})
}

type rejectLeaveRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// Reject rejects a leave request.
func (h *LeaveHandler) Reject(c *gin.Context) {
	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid leave ID"})
		return
	}

	var req rejectLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	rejectedByStr, _ := c.Get("employee_id")
	rejectedByID, _ := uuid.Parse(rejectedByStr.(string))
	roleVal, _ := c.Get("role")
	roleStr, _ := roleVal.(string)

	if err := h.leaveSvc.RejectLeave(c.Request.Context(), leaveID, rejectedByID, roleStr, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "leave rejected"}})
}

// LeaveHistory returns all leaves with approval details for supervisors, optionally filtered by department.
func (h *LeaveHandler) LeaveHistory(c *gin.Context) {
	deptID := getDepartmentID(c)
	history, err := h.leaveSvc.GetLeaveHistory(c.Request.Context(), deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if history == nil {
		history = []models.LeaveHistoryRow{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": history, "meta": gin.H{"count": len(history)}})
}
