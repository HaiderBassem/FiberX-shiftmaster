package handlers

import (
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InfoTableHandler struct {
	svc *service.InfoTableService
}

func NewInfoTableHandler(svc *service.InfoTableService) *InfoTableHandler {
	return &InfoTableHandler{svc: svc}
}

func (h *InfoTableHandler) GetVisibleTables(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)
	
	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, err := uuid.Parse(depStr.(string))
		if err == nil {
			depID = &id
		}
	}

	tables, err := h.svc.GetVisibleTables(c.Request.Context(), empID, role, depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tables})
}

func (h *InfoTableHandler) CreateTable(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	canCreate := false
	if cc, ok := c.Get("can_create_tables"); ok {
		canCreate = cc.(bool)
	}

	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, err := uuid.Parse(depStr.(string))
		if err == nil {
			depID = &id
		}
	}

	var req models.InfoTable
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	req.CreatedBy = &empID
	req.DepartmentID = depID

	created, err := h.svc.CreateTable(c.Request.Context(), &req, role, canCreate)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *InfoTableHandler) GetTableRows(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)
	
	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, _ := uuid.Parse(depStr.(string))
		depID = &id
	}

	tableID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table id"})
		return
	}

	rows, err := h.svc.GetTableRows(c.Request.Context(), tableID, empID, role, depID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rows})
}

func (h *InfoTableHandler) CreateTableRow(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)
	
	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, _ := uuid.Parse(depStr.(string))
		depID = &id
	}

	tableID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table id"})
		return
	}

	var req models.InfoTableRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	req.TableID = tableID
	req.CreatedBy = &empID

	created, err := h.svc.CreateTableRow(c.Request.Context(), &req, empID, role, depID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *InfoTableHandler) UpdateTableRow(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)
	
	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, _ := uuid.Parse(depStr.(string))
		depID = &id
	}

	rowID, err := uuid.Parse(c.Param("rowId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid row id"})
		return
	}

	var req models.InfoTableRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.ID = rowID

	tableID, err := uuid.Parse(c.Param("id"))
	if err == nil {
		req.TableID = tableID
	} else {
		// Try tableId if id fails (depends on route definitions)
		tableID, err = uuid.Parse(c.Param("tableId"))
		if err == nil {
			req.TableID = tableID
		}
	}

	updated, err := h.svc.UpdateTableRow(c.Request.Context(), &req, empID, role, depID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *InfoTableHandler) DeleteTableRow(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)
	
	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, _ := uuid.Parse(depStr.(string))
		depID = &id
	}

	tableID, _ := uuid.Parse(c.Param("id"))
	rowID, err := uuid.Parse(c.Param("rowId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid row id"})
		return
	}

	err = h.svc.DeleteTableRow(c.Request.Context(), rowID, tableID, empID, role, depID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Access Mgmt Endpoints
func (h *InfoTableHandler) ShareWithDepartment(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	tableID, _ := uuid.Parse(c.Param("id"))
	
	var req struct {
		DepartmentID uuid.UUID `json:"department_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	err := h.svc.ShareWithDepartment(c.Request.Context(), role, tableID, req.DepartmentID, empID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *InfoTableHandler) AddEmployeeAccess(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)
	
	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, _ := uuid.Parse(depStr.(string))
		depID = &id
	}

	tableID, _ := uuid.Parse(c.Param("id"))

	var req struct {
		EmployeeID  uuid.UUID `json:"employee_id"`
		AccessLevel string    `json:"access_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	err := h.svc.AddEmployeeAccess(c.Request.Context(), empID, role, depID, tableID, req.EmployeeID, req.AccessLevel)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *InfoTableHandler) RemoveEmployeeAccess(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)
	
	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, _ := uuid.Parse(depStr.(string))
		depID = &id
	}

	tableID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table id"})
		return
	}

	targetEmployeeID, err := uuid.Parse(c.Param("employeeId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employee id"})
		return
	}

	err = h.svc.RemoveEmployeeAccess(c.Request.Context(), empID, role, depID, tableID, targetEmployeeID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *InfoTableHandler) GetAccessLists(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)
	
	var depID *uuid.UUID
	if depStr, ok := c.Get("department_id"); ok && depStr != "" {
		id, _ := uuid.Parse(depStr.(string))
		depID = &id
	}

	tableID, _ := uuid.Parse(c.Param("id"))

	depAcc, empAcc, err := h.svc.GetAccessLists(c.Request.Context(), tableID, empID, role, depID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"departments": depAcc,
		"employees":   empAcc,
	})
}
