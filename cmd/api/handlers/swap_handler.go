package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

// SwapHandler handles shift swap endpoints.
type SwapHandler struct {
	swapSvc *service.SwapService
}

func NewSwapHandler(swapSvc *service.SwapService) *SwapHandler {
	return &SwapHandler{swapSvc: swapSvc}
}

type createSwapRequest struct {
	TargetEmployeeID string `json:"target_employee_id" binding:"required"`
	ShiftDate        string `json:"shift_date" binding:"required"`
	// Phase 3: frontend may omit shift_id; backend will infer from requester's shift.
	ShiftID          string `json:"shift_id"`
	Reason           *string `json:"reason"`
}

// Request creates a new shift swap request.
func (h *SwapHandler) Request(c *gin.Context) {
	var req createSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	targetID, err := uuid.Parse(req.TargetEmployeeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid target_employee_id"})
		return
	}
	var shiftID uuid.UUID
	if req.ShiftID == "" {
		shiftID = uuid.Nil
	} else {
		shiftID, err = uuid.Parse(req.ShiftID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_id"})
			return
		}
	}
	shiftDate, err := parseTime(req.ShiftDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift_date format"})
		return
	}

	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))

	swap := &models.ShiftSwap{
		RequesterID:      requesterID,
		TargetEmployeeID: targetID,
		ShiftDate:        shiftDate,
		ShiftID:          shiftID,
		Reason:           req.Reason,
	}

	if err := h.swapSvc.RequestSwap(c.Request.Context(), swap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": swap})
}

// MyRequests returns swap requests created by the authenticated employee.
func (h *SwapHandler) MyRequests(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	swaps, err := h.swapSvc.GetMySwapRequests(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if swaps == nil {
		swaps = []models.ShiftSwap{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": swaps, "meta": gin.H{"count": len(swaps)}})
}

// PendingForMe returns swaps waiting for my acceptance.
func (h *SwapHandler) PendingForMe(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	swaps, err := h.swapSvc.GetPendingSwapsForMe(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if swaps == nil {
		swaps = []models.ShiftSwap{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": swaps, "meta": gin.H{"count": len(swaps)}})
}

// PendingForManager returns swaps waiting for manager approval for the approver's department.
func (h *SwapHandler) PendingForManager(c *gin.Context) {
	approverStr, _ := c.Get("employee_id")
	approverID, _ := uuid.Parse(approverStr.(string))

	swaps, err := h.swapSvc.GetPendingSwapsForManager(c.Request.Context(), approverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if swaps == nil {
		swaps = []models.ShiftSwap{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": swaps, "meta": gin.H{"count": len(swaps)}})
}

// EligibleTargets returns employees in the requester's department for a target date.
func (h *SwapHandler) EligibleTargets(c *gin.Context) {
	date, err := parseTime(c.Query("date"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid date format"})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	targets, err := h.swapSvc.GetEligibleSwapTargets(c.Request.Context(), empID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if targets == nil {
		targets = []models.SwapEligibleEmployee{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": targets, "meta": gin.H{"count": len(targets)}})
}

type respondRequest struct {
	Accept bool `json:"accept"`
}

// Respond handles the target employee's acceptance or rejection.
func (h *SwapHandler) Respond(c *gin.Context) {
	swapID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid swap ID"})
		return
	}

	var req respondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	if err := h.swapSvc.EmployeeRespond(c.Request.Context(), swapID, empID, req.Accept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	msg := "swap rejected"
	if req.Accept {
		msg = "swap accepted, awaiting team leader approval"
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": msg}})
}

// Approve handles manager/team leader approval.
func (h *SwapHandler) Approve(c *gin.Context) {
	swapID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid swap ID"})
		return
	}

	approverStr, _ := c.Get("employee_id")
	approverID, _ := uuid.Parse(approverStr.(string))
	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	if err := h.swapSvc.ApproveSwap(c.Request.Context(), swapID, approverID, roleStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "swap approved"}})
}

// Reject handles manager rejection.
func (h *SwapHandler) Reject(c *gin.Context) {
	swapID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid swap ID"})
		return
	}

	approverStr, _ := c.Get("employee_id")
	approverID, _ := uuid.Parse(approverStr.(string))
	role, _ := c.Get("role")
	roleStr, _ := role.(string)

	if err := h.swapSvc.RejectSwap(c.Request.Context(), swapID, approverID, roleStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "swap rejected"}})
}

// Cancel cancels a swap request.
func (h *SwapHandler) Cancel(c *gin.Context) {
	swapID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid swap ID"})
		return
	}

	requesterStr, _ := c.Get("employee_id")
	requesterID, _ := uuid.Parse(requesterStr.(string))

	if err := h.swapSvc.CancelSwap(c.Request.Context(), swapID, requesterID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "swap cancelled"}})
}
