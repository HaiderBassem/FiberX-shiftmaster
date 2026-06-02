package handlers

import (
	"net/http"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type HelpDocumentHandler struct {
	svc *service.HelpDocumentService
}

func NewHelpDocumentHandler(svc *service.HelpDocumentService) *HelpDocumentHandler {
	return &HelpDocumentHandler{svc: svc}
}

func (h *HelpDocumentHandler) GetVisibleDocuments(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	depID := getDepartmentID(c)

	docs, err := h.svc.GetVisibleDocuments(c.Request.Context(), depID, empID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": docs})
}

func (h *HelpDocumentHandler) GetDocument(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	doc, err := h.svc.GetDocumentByID(c.Request.Context(), docID, empID, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if doc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": doc})
}

func (h *HelpDocumentHandler) CreateDocument(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	depID := getDepartmentID(c)

	if depID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user must belong to a department"})
		return
	}

	var req models.HelpDocument
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	req.DepartmentID = *depID
	req.CreatedBy = &empID

	created, err := h.svc.CreateDocument(c.Request.Context(), &req, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *HelpDocumentHandler) UpdateDocument(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var req models.HelpDocument
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	req.ID = docID

	updated, err := h.svc.UpdateDocument(c.Request.Context(), &req, empID, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *HelpDocumentHandler) DeleteDocument(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	err = h.svc.DeleteDocument(c.Request.Context(), docID, empID, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *HelpDocumentHandler) GetAccessList(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	list, err := h.svc.GetDocumentAccessList(c.Request.Context(), docID, empID, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *HelpDocumentHandler) SetEmployeeAccess(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))
	roleStr, _ := c.Get("role")
	role := roleStr.(string)

	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document id"})
		return
	}

	var req struct {
		EmployeeID  uuid.UUID `json:"employee_id"`
		AccessLevel string    `json:"access_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	err = h.svc.SetEmployeeAccess(c.Request.Context(), docID, req.EmployeeID, req.AccessLevel, empID, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
