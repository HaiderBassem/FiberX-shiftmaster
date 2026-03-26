package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Response is the standard API response envelope.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta holds pagination or extra information.
type Meta struct {
	Total int `json:"total,omitempty"`
	Count int `json:"count,omitempty"`
}

func sendSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

func sendCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Success: true, Data: data})
}

func sendError(c *gin.Context, status int, message string) {
	c.JSON(status, Response{Success: false, Error: message})
}

func sendList(c *gin.Context, data interface{}, count int) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    &Meta{Count: count},
	})
}

// parseUUID extracts and parses a UUID path parameter.
func parseUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid "+param+" format")
		return uuid.Nil, false
	}
	return id, true
}

// parseDate parses a date query parameter (YYYY-MM-DD).
func parseDate(c *gin.Context, param string) (time.Time, bool) {
	dateStr := c.Query(param)
	if dateStr == "" {
		sendError(c, http.StatusBadRequest, param+" query parameter is required")
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid "+param+" format, expected YYYY-MM-DD")
		return time.Time{}, false
	}
	return t, true
}

// getEmployeeID retrieves the authenticated employee's ID from context.
func getEmployeeID(c *gin.Context) (uuid.UUID, bool) {
	idStr, exists := c.Get("employee_id")
	if !exists {
		sendError(c, http.StatusUnauthorized, "authentication required")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr.(string))
	if err != nil {
		sendError(c, http.StatusInternalServerError, "invalid employee ID in token")
		return uuid.Nil, false
	}
	return id, true
}

// getRole retrieves the authenticated employee's role from context.
func getRole(c *gin.Context) string {
	role, _ := c.Get("role")
	if r, ok := role.(string); ok {
		return r
	}
	return ""
}
