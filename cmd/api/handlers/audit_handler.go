package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/service"
)

// AuditHandler exposes activity history endpoints.
type AuditHandler struct {
	auditSvc *service.AuditService
}

func NewAuditHandler(auditSvc *service.AuditService) *AuditHandler {
	return &AuditHandler{auditSvc: auditSvc}
}

// ListActivity returns recent audit logs for the authenticated employee.
func (h *AuditHandler) ListActivity(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := h.auditSvc.GetActivityForEmployee(c.Request.Context(), empID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if logs == nil {
		logs = []models.AuditLog{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": logs, "meta": gin.H{"count": len(logs)}})
}

