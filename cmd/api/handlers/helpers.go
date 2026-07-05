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

// getDepartmentID gets the department ID securely validated by DepartmentContext middleware.
func getDepartmentID(c *gin.Context) *uuid.UUID {
	if idRaw, exists := c.Get("context_department_id"); exists {
		id := idRaw.(uuid.UUID)
		return &id
	}
	return nil
}
