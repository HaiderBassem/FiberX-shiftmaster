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

// getDepartmentID gets the department ID from context or header
func getDepartmentID(c *gin.Context) *uuid.UUID {
	role, exists := c.Get("role")
	if exists && role == "admin" {
		headerVal := c.GetHeader("X-Department-ID")
		if headerVal != "" && headerVal != "null" {
			parsed, err := uuid.Parse(headerVal)
			if err == nil {
				return &parsed
			}
		}
		return nil
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
