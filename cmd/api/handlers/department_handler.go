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

// MyManaged returns all departments that the authenticated manager is assigned to.
// Only meaningful for role=manager; other roles get an empty list.
func (h *DepartmentHandler) MyManaged(c *gin.Context) {
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	if role != "manager" {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}})
		return
	}

	empIDStr, _ := c.Get("employee_id")
	managerID, err := uuid.Parse(empIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid token"})
		return
	}

	departments, err := h.deptRepo.GetByManagerID(c.Request.Context(), managerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": departments, "meta": gin.H{"count": len(departments)}})
}

// createDepartmentRequest supports assigning multiple managers on creation.
type createDepartmentRequest struct {
	DepartmentCode string      `json:"department_code" binding:"required"`
	Name           string      `json:"name"            binding:"required"`
	Description    *string     `json:"description"`
	ManagerIDs     []uuid.UUID `json:"manager_ids"` // zero or more manager UUIDs
}

// Create creates a new department.
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req createDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	// Validate every supplied manager
	for _, mID := range req.ManagerIDs {
		emp, err := h.employeeRepo.GetByID(c.Request.Context(), mID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid manager_id: " + mID.String()})
			return
		}
		if emp.Role != "manager" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "employee " + mID.String() + " is not a manager",
			})
			return
		}
	}

	dept := &models.Department{
		DepartmentCode: req.DepartmentCode,
		Name:           req.Name,
		Description:    req.Description,
		ManagerIDs:     req.ManagerIDs,
	}

	if err := h.deptRepo.Create(c.Request.Context(), dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": dept})
}

// updateDepartmentRequest supports replacing the full list of managers.
// If manager_ids is omitted from the JSON body the existing assignments are kept.
type updateDepartmentRequest struct {
	Name        string      `json:"name"`
	Description *string     `json:"description"`
	ManagerIDs  []uuid.UUID `json:"manager_ids"` // if provided, replaces all current managers
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

	// Validate every supplied manager
	for _, mID := range req.ManagerIDs {
		emp, err := h.employeeRepo.GetByID(c.Request.Context(), mID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid manager_id: " + mID.String()})
			return
		}
		if emp.Role != "manager" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "employee " + mID.String() + " is not a manager",
			})
			return
		}
	}

	dept := &models.Department{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		ManagerIDs:  req.ManagerIDs, // nil when key is absent → repo skips sync
	}

	if err := h.deptRepo.Update(c.Request.Context(), dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Re-fetch to return the fresh state (including updated manager_ids)
	updated, err := h.deptRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": dept})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": updated})
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

// AddManager links a single manager to a department.
// POST /departments/:id/managers   body: {"manager_id": "<uuid>"}
func (h *DepartmentHandler) AddManager(c *gin.Context) {
	deptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid department ID"})
		return
	}

	var body struct {
		ManagerID uuid.UUID `json:"manager_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	emp, err := h.employeeRepo.GetByID(c.Request.Context(), body.ManagerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid manager_id"})
		return
	}
	if emp.Role != "manager" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "employee is not a manager"})
		return
	}

	if err := h.deptRepo.AddManager(c.Request.Context(), deptID, body.ManagerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "manager added"}})
}

// RemoveManager unlinks a manager from a department.
// DELETE /departments/:id/managers/:manager_id
func (h *DepartmentHandler) RemoveManager(c *gin.Context) {
	deptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid department ID"})
		return
	}

	mgrID, err := uuid.Parse(c.Param("manager_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid manager ID"})
		return
	}

	if err := h.deptRepo.RemoveManager(c.Request.Context(), deptID, mgrID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "manager removed"}})
}
