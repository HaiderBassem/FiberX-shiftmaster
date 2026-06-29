package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// ShiftHandler handles shift CRUD endpoints.
type ShiftHandler struct {
	shiftRepo repository.ShiftRepository
}

func NewShiftHandler(shiftRepo repository.ShiftRepository) *ShiftHandler {
	return &ShiftHandler{shiftRepo: shiftRepo}
}

// List returns all shifts.
func (h *ShiftHandler) List(c *gin.Context) {
	deptID := getDepartmentID(c)
	shifts, err := h.shiftRepo.GetAll(c.Request.Context(), deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if shifts == nil {
		shifts = []models.Shift{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": shifts, "meta": gin.H{"count": len(shifts)}})
}

// GetByID returns a single shift.
func (h *ShiftHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift ID"})
		return
	}

	shift, err := h.shiftRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": shift})
}

type createShiftRequest struct {
	ShiftCode       string  `json:"shift_code" binding:"required"`
	Name            string  `json:"name" binding:"required"`
	NameEn          *string `json:"name_en"`
	StartTime       string  `json:"start_time" binding:"required"`
	EndTime         string  `json:"end_time" binding:"required"`
	ColorCode       *string `json:"color_code"`
	RequiresVehicle bool    `json:"requires_vehicle"`
	MinRestHours    int     `json:"min_rest_hours"`
}

func parseTimeStr(t string) (time.Time, error) {
	if len(t) > 5 {
		t = t[:5]
	}
	return time.Parse("15:04", t)
}

// Create creates a new shift.
func (h *ShiftHandler) Create(c *gin.Context) {
	var req createShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	startTime, err := parseTimeStr(req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid start_time format, expected HH:MM"})
		return
	}
	endTime, err := parseTimeStr(req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid end_time format, expected HH:MM"})
		return
	}

	deptID := getDepartmentID(c)

	shift := &models.Shift{
		ShiftCode:       req.ShiftCode,
		Name:            req.Name,
		NameEn:          req.NameEn,
		StartTime:       startTime,
		EndTime:         endTime,
		ColorCode:       req.ColorCode,
		RequiresVehicle: req.RequiresVehicle,
		MinRestHours:    req.MinRestHours,
		DepartmentID:    deptID,
	}

	if err := h.shiftRepo.Create(c.Request.Context(), shift); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": shift})
}

// Update updates a shift.
func (h *ShiftHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift ID"})
		return
	}

	var req createShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	startTime, err := parseTimeStr(req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid start_time format"})
		return
	}
	endTime, err := parseTimeStr(req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid end_time format"})
		return
	}

	shift := &models.Shift{
		ID:              id,
		ShiftCode:       req.ShiftCode,
		Name:            req.Name,
		NameEn:          req.NameEn,
		StartTime:       startTime,
		EndTime:         endTime,
		ColorCode:       req.ColorCode,
		RequiresVehicle: req.RequiresVehicle,
		MinRestHours:    req.MinRestHours,
	}

	if err := h.shiftRepo.Update(c.Request.Context(), shift); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": shift})
}

// Delete deletes a shift.
func (h *ShiftHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid shift ID"})
		return
	}

	if err := h.shiftRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "shift deleted"}})
}
