package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"shiftmaster-backend/internal/service"
)

func IPBlockerMiddleware(securityService *service.SecurityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if securityService.IsIPBlocked(c.Request.Context(), clientIP) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "Access denied. Your IP address has been temporarily blocked due to suspicious activity.",
			})
			return
		}

		c.Next()
	}
}
