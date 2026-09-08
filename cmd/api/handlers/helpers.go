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

// actorID returns the authenticated employee's ID from the JWT context.
// The boolean is false only when the middleware chain did not run, which on a
// protected route means the request must be refused, not guessed at.
func actorID(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("employee_id")
	if !exists {
		return uuid.Nil, false
	}
	s, ok := raw.(string)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// actorRole returns the authenticated employee's role from the JWT context.
func actorRole(c *gin.Context) string {
	raw, exists := c.Get("role")
	if !exists {
		return ""
	}
	s, _ := raw.(string)
	return s
}
