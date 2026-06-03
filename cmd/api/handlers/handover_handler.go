package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
)

type HandoverHandler struct {
	handoverRepo repository.HandoverRepository
	employeeRepo repository.EmployeeRepository
	notifService *service.NotificationService
}

func NewHandoverHandler(hr repository.HandoverRepository, er repository.EmployeeRepository, ns *service.NotificationService) *HandoverHandler {
	return &HandoverHandler{
		handoverRepo: hr,
		employeeRepo: er,
		notifService: ns,
	}
}

type createHandoverRequest struct {
	ShiftSummary  string `json:"shift_summary" binding:"required"`
	PendingIssues string `json:"pending_issues"`
}

func (h *HandoverHandler) CreateHandover(c *gin.Context) {
	var req createHandoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	emp, err := h.employeeRepo.GetByID(c.Request.Context(), empID)
	if err != nil || emp.DepartmentID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employee department not found"})
		return
	}

	handover := &models.Handover{
		DepartmentID:  *emp.DepartmentID,
		CreatorID:     empID,
		ShiftSummary:  req.ShiftSummary,
		PendingIssues: req.PendingIssues,
	}

	if err := h.handoverRepo.Create(c.Request.Context(), handover); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create handover"})
		return
	}

	// Notify all employees in the department
	deptEmps, _ := h.employeeRepo.GetByDepartment(c.Request.Context(), *emp.DepartmentID)
	for _, e := range deptEmps {
		if e.ID != empID {
			msg := fmt.Sprintf("%s %s created a new shift handover.", emp.FirstName, emp.LastName)
			_ = h.notifService.SendNotification(c.Request.Context(), &models.Notification{
				RecipientID: e.ID,
				SenderID:    &empID,
				Type:        "handover",
				Title:       "New Shift Handover",
				Message:     &msg,
				Priority:    "high",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": handover})
}

func (h *HandoverHandler) GetHandovers(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	emp, err := h.employeeRepo.GetByID(c.Request.Context(), empID)
	if err != nil || emp.DepartmentID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employee department not found"})
		return
	}

	handovers, err := h.handoverRepo.GetByDepartment(c.Request.Context(), *emp.DepartmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get handovers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": handovers})
}

func (h *HandoverHandler) ClaimHandover(c *gin.Context) {
	handoverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid handover id"})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	if err := h.handoverRepo.Claim(c.Request.Context(), handoverID, empID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to claim handover"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *HandoverHandler) CompleteHandover(c *gin.Context) {
	handoverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid handover id"})
		return
	}

	if err := h.handoverRepo.Complete(c.Request.Context(), handoverID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete handover"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
