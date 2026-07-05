package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

type ItemRequestHandler struct {
	svc *service.ItemRequestService
}

func NewItemRequestHandler(svc *service.ItemRequestService) *ItemRequestHandler {
	return &ItemRequestHandler{svc: svc}
}

// GetCategories returns categories for the user's department
func (h *ItemRequestHandler) GetCategories(c *gin.Context) {
	deptIDStr, exists := c.Get("department_id")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "user has no department"})
		return
	}
	deptID, _ := uuid.Parse(deptIDStr.(string))

	cats, err := h.svc.GetCategoriesByDepartment(c.Request.Context(), deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": cats})
}

// CreateCategory (Team Leader / Admin only)
func (h *ItemRequestHandler) CreateCategory(c *gin.Context) {
	deptIDStr, _ := c.Get("department_id")
	deptID, _ := uuid.Parse(deptIDStr.(string))

	var req struct {
		Name     string  `json:"name" binding:"required"`
		ToEmails string  `json:"to_emails" binding:"required"`
		CCEmails *string `json:"cc_emails"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	cat := &models.ItemRequestCategory{
		DepartmentID: deptID,
		Name:         req.Name,
		ToEmails:     req.ToEmails,
		CCEmails:     req.CCEmails,
	}

	if err := h.svc.CreateCategory(c.Request.Context(), cat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": cat})
}

// UpdateCategory (Team Leader / Admin only)
func (h *ItemRequestHandler) UpdateCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}

	var req struct {
		Name     string  `json:"name" binding:"required"`
		ToEmails string  `json:"to_emails" binding:"required"`
		CCEmails *string `json:"cc_emails"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	cat := &models.ItemRequestCategory{
		ID:       id,
		Name:     req.Name,
		ToEmails: req.ToEmails,
		CCEmails: req.CCEmails,
	}

	if err := h.svc.UpdateCategory(c.Request.Context(), cat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": cat})
}

// DeleteCategory (Team Leader / Admin only)
func (h *ItemRequestHandler) DeleteCategory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}

	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		if err.Error() == "cannot delete category because it is used by existing requests" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetMyRequests
func (h *ItemRequestHandler) GetMyRequests(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	reqs, err := h.svc.GetRequestsByEmployee(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": reqs})
}

// SubmitRequest
func (h *ItemRequestHandler) SubmitRequest(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	var req struct {
		CategoryID  uuid.UUID `json:"category_id" binding:"required"`
		Description string    `json:"description" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	itemReq, err := h.svc.SubmitRequest(c.Request.Context(), empID, req.CategoryID, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": itemReq})
}

func (h *ItemRequestHandler) GetPendingRequests(c *gin.Context) {
	depID := getDepartmentID(c)
	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Department ID required"})
		return
	}
	reqs, err := h.svc.GetPendingRequests(c.Request.Context(), *depID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reqs})
}

func (h *ItemRequestHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"}); return }

	var req struct { Status string `json:"status" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.svc.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ItemRequestHandler) CancelRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"}); return }

	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	if err := h.svc.CancelRequest(c.Request.Context(), id, empID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

