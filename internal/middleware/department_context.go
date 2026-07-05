package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/repository"
)

// DepartmentContext validates and injects the context_department_id into the request context.
func DepartmentContext(deptRepo repository.DepartmentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.Next()
			return
		}
		roleStr := role.(string)

		empIDRaw, exists := c.Get("employee_id")
		if !exists {
			c.Next()
			return
		}
		empIDStr := empIDRaw.(string)
		empID, err := uuid.Parse(empIDStr)
		if err != nil {
			c.Next()
			return
		}

		// Get user's default department from JWT claims
		var defaultDeptID *uuid.UUID
		if defaultDeptIDRaw, exists := c.Get("department_id"); exists {
			idStr := defaultDeptIDRaw.(string)
			if id, err := uuid.Parse(idStr); err == nil {
				defaultDeptID = &id
			}
		}

		reqDeptIDStr := c.GetHeader("X-Department-ID")
		if reqDeptIDStr == "" {
			// No explicit department context requested, use default
			if defaultDeptID != nil {
				c.Set("context_department_id", *defaultDeptID)
			}
			c.Next()
			return
		}

		reqDeptID, err := uuid.Parse(reqDeptIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid X-Department-ID format"})
			return
		}

		// Validate access based on role
		if roleStr == "admin" || roleStr == "hr" {
			// Admins and HR have global access
			c.Set("context_department_id", reqDeptID)
			c.Next()
			return
		}

		if roleStr == "manager" {
			// Check if manager manages this department
			managedDepts, err := deptRepo.GetByManagerID(c.Request.Context(), empID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to verify department access"})
				return
			}
			
			hasAccess := false
			for _, d := range managedDepts {
				if d.ID == reqDeptID {
					hasAccess = true
					break
				}
			}

			if !hasAccess {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "you do not have access to this department"})
				return
			}

			c.Set("context_department_id", reqDeptID)
			c.Next()
			return
		}

		// Standard employees and team leaders can ONLY access their own department
		if defaultDeptID != nil && *defaultDeptID == reqDeptID {
			c.Set("context_department_id", reqDeptID)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "you can only access your own department"})
		return
	}
}
