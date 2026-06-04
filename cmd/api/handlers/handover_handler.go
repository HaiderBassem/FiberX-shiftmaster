package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
	"shiftmaster-backend/internal/service"
)

type HandoverHandler struct {
	handoverRepo repository.HandoverRepository
	employeeRepo repository.EmployeeRepository
	shiftRepo    repository.ShiftRepository
	scheduleRepo repository.ScheduleRepository
	notifService *service.NotificationService
}

func NewHandoverHandler(hr repository.HandoverRepository, er repository.EmployeeRepository, sr repository.ShiftRepository, sch repository.ScheduleRepository, ns *service.NotificationService) *HandoverHandler {
	return &HandoverHandler{
		handoverRepo: hr,
		employeeRepo: er,
		shiftRepo:    sr,
		scheduleRepo: sch,
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

	// Determine Current Shift based on Time
	now := time.Now()
	hour := now.Hour()
	var nextShiftCode string
	var nextShiftName string

	// MORNING: 08:00 - 16:00 -> Next is EVENING
	// EVENING: 16:00 - 00:00 -> Next is NIGHT
	// NIGHT:   00:00 - 08:00 -> Next is MORNING
	if hour >= 8 && hour < 16 {
		nextShiftCode = "E"
		nextShiftName = "Evening"
	} else if hour >= 16 && hour < 24 {
		nextShiftCode = "N"
		nextShiftName = "Night"
	} else {
		nextShiftCode = "M"
		nextShiftName = "Morning"
	}

	// Fetch all shifts for the department to find the target Shift ID
	shifts, _ := h.shiftRepo.GetAll(c.Request.Context(), emp.DepartmentID)
	var targetShiftID *uuid.UUID
	for _, s := range shifts {
		if strings.EqualFold(s.ShiftCode, nextShiftCode) {
			targetShiftID = &s.ID
			break
		}
	}

	deptEmps, _ := h.employeeRepo.GetByDepartment(c.Request.Context(), *emp.DepartmentID)
	
	// Track if we successfully found employees to notify
	notifiedCount := 0

	for _, e := range deptEmps {
		if e.ID != empID {
			// If targetShiftID is found, only notify employees whose DefaultShiftID matches.
			if targetShiftID != nil && e.DefaultShiftID != nil && *e.DefaultShiftID != *targetShiftID {
				continue
			}

			msg := fmt.Sprintf("%s %s created a new handover for the %s shift.", emp.FirstName, emp.LastName, nextShiftName)
			err := h.notifService.SendNotification(c.Request.Context(), &models.Notification{
				RecipientID: e.ID,
				SenderID:    &empID,
				Type:        "system_alert",
				Title:       "New Shift Handover",
				Message:     &msg,
				Priority:    "high",
			})
			if err == nil {
				notifiedCount++
			}
		}
	}

	// If no one was notified (maybe all DefaultShiftID are null?), fallback to notifying everyone
	if notifiedCount == 0 && targetShiftID != nil {
		for _, e := range deptEmps {
			if e.ID != empID {
				msg := fmt.Sprintf("%s %s created a new handover for the %s shift.", emp.FirstName, emp.LastName, nextShiftName)
				_ = h.notifService.SendNotification(c.Request.Context(), &models.Notification{
					RecipientID: e.ID,
					SenderID:    &empID,
					Type:        "system_alert",
					Title:       "New Shift Handover",
					Message:     &msg,
					Priority:    "high",
				})
			}
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
