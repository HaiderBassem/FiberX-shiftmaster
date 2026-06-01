package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// parseTime parses a date string in YYYY-MM-DD format.
func parseTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// getDepartmentID gets the department ID from context or header.
// For admin and manager roles the optional X-Department-ID request header
// overrides the department stored in the JWT token, allowing them to switch
// between departments they are authorised to manage.
func getDepartmentID(c *gin.Context) *uuid.UUID {
	role, exists := c.Get("role")
	roleStr, _ := role.(string)

	if exists && (roleStr == "admin" || roleStr == "manager") {
		headerVal := c.GetHeader("X-Department-ID")
		if headerVal != "" && headerVal != "null" {
			parsed, err := uuid.Parse(headerVal)
			if err == nil {
				return &parsed
			}
		}
		// Admin with no header → nil (all departments)
		// Manager with no header → fall through to JWT department_id
		if roleStr == "admin" {
			return nil
		}
	}

	deptIDStr, exists := c.Get("department_id")
	if exists {
		parsed, err := uuid.Parse(deptIDStr.(string))
		if err == nil {
			return &parsed
		}
	}
	return nil
}
