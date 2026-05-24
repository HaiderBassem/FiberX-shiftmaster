package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

type LeaveTypeHandler struct {
	service service.LeaveTypeService
}

func NewLeaveTypeHandler(s service.LeaveTypeService) *LeaveTypeHandler {
	return &LeaveTypeHandler{service: s}
}

func (h *LeaveTypeHandler) GetAll(c *gin.Context) {
	activeOnly := c.Query("active") == "true"

	var lts []models.LeaveType
	var err error

	if activeOnly {
		lts, err = h.service.GetActiveLeaveTypes(c.Request.Context())
	} else {
		lts, err = h.service.GetAllLeaveTypes(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to get leave types: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": lts})
}

func (h *LeaveTypeHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID format"})
		return
	}

	lt, err := h.service.GetLeaveTypeByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Leave type not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": lt})
}

func (h *LeaveTypeHandler) Create(c *gin.Context) {
	var lt models.LeaveType
	if err := c.ShouldBindJSON(&lt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request payload"})
		return
	}

	if lt.NameEn == "" || lt.NameAr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "English and Arabic names are required"})
		return
	}

	if err := h.service.CreateLeaveType(c.Request.Context(), &lt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create leave type: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": lt})
}

func (h *LeaveTypeHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID format"})
		return
	}

	var lt models.LeaveType
	if err := c.ShouldBindJSON(&lt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request payload"})
		return
	}
	lt.ID = id

	if err := h.service.UpdateLeaveType(c.Request.Context(), &lt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to update leave type: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": lt})
}

func (h *LeaveTypeHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid ID format"})
		return
	}

	if err := h.service.DeleteLeaveType(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to delete leave type (might be in use)"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Leave type deleted successfully"})
}
