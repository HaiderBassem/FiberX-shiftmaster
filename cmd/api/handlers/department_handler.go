package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// DepartmentHandler handles department CRUD endpoints.
type DepartmentHandler struct {
	deptRepo     repository.DepartmentRepository
	employeeRepo repository.EmployeeRepository
}

func NewDepartmentHandler(deptRepo repository.DepartmentRepository, employeeRepo repository.EmployeeRepository) *DepartmentHandler {
	return &DepartmentHandler{deptRepo: deptRepo, employeeRepo: employeeRepo}
}

// List returns all departments.
func (h *DepartmentHandler) List(c *gin.Context) {
	departments, err := h.deptRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if departments == nil {
		departments = []models.Department{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": departments, "meta": gin.H{"count": len(departments)}})
}

// GetByID returns a single department.
func (h *DepartmentHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid department ID"})
		return
	}

	dept, err := h.deptRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": dept})
}

type createDepartmentRequest struct {
	DepartmentCode string     `json:"department_code" binding:"required"`
	Name           string     `json:"name" binding:"required"`
	Description    *string    `json:"description"`
	ManagerID      *uuid.UUID `json:"manager_id"`
}

// Create creates a new department.
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req createDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	if req.ManagerID != nil {
		emp, err := h.employeeRepo.GetByID(c.Request.Context(), *req.ManagerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid manager_id"})
			return
		}
		if emp.Role != "manager" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "manager_id must be a manager account"})
			return
		}
	}

	dept := &models.Department{
		DepartmentCode: req.DepartmentCode,
		Name:           req.Name,
		Description:    req.Description,
		ManagerID:      req.ManagerID,
	}

	if err := h.deptRepo.Create(c.Request.Context(), dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": dept})
}

type updateDepartmentRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	ManagerID   *uuid.UUID `json:"manager_id"`
}

// Update updates a department.
func (h *DepartmentHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid department ID"})
		return
	}

	var req updateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	if req.ManagerID != nil {
		emp, err := h.employeeRepo.GetByID(c.Request.Context(), *req.ManagerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid manager_id"})
			return
		}
		if emp.Role != "manager" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "manager_id must be a manager account"})
			return
		}
	}

	dept := &models.Department{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		ManagerID:   req.ManagerID,
	}

	if err := h.deptRepo.Update(c.Request.Context(), dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": dept})
}

// Delete deletes a department.
func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid department ID"})
		return
	}

	if err := h.deptRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "department deleted"}})
}
