package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"shiftmaster-backend/internal/service"
)

type SecurityHandler struct {
	securityService *service.SecurityService
}

func NewSecurityHandler(s *service.SecurityService) *SecurityHandler {
	return &SecurityHandler{securityService: s}
}

func (h *SecurityHandler) GetBlockedIPs(c *gin.Context) {
	blocks, err := h.securityService.GetBlockedIPs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": blocks})
}

func (h *SecurityHandler) UnblockIP(c *gin.Context) {
	ip := c.Param("ip")
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "IP address is required"})
		return
	}

	if err := h.securityService.UnblockIP(c.Request.Context(), ip); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "IP unblocked successfully"}})
}
