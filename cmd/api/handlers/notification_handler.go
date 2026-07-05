package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/notification"
	"shiftmaster-backend/internal/service"
)

// NotificationHandler handles notification endpoints.
type NotificationHandler struct {
	notifSvc *service.NotificationService
}

func NewNotificationHandler(notifSvc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifSvc: notifSvc}
}

// List returns all notifications for the authenticated user.
func (h *NotificationHandler) List(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	notifs, err := h.notifSvc.GetNotifications(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if notifs == nil {
		notifs = []models.Notification{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": notifs, "meta": gin.H{"count": len(notifs)}})
}

// Unread returns unread notifications.
func (h *NotificationHandler) Unread(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	notifs, err := h.notifSvc.GetUnread(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if notifs == nil {
		notifs = []models.Notification{}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": notifs, "meta": gin.H{"count": len(notifs)}})
}

// UnreadCount returns the count of unread notifications.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	count, err := h.notifSvc.GetUnreadCount(c.Request.Context(), empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"count": count}})
}

// MarkAsRead marks a single notification as read.
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid notification ID"})
		return
	}

	if err := h.notifSvc.MarkAsRead(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "notification updated"}})
}

// ServeWS handles WebSocket connection for real-time notifications
func (h *NotificationHandler) ServeWS(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, err := uuid.Parse(empIDStr.(string))
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	notification.ServeWS(c.Writer, c.Request, empID)
}

// MarkAllAsRead marks all notifications as read.
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	if err := h.notifSvc.MarkAllAsRead(c.Request.Context(), empID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "all notifications marked as read"}})
}
