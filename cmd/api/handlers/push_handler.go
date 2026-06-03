package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

type PushHandler struct {
	repo   repository.NotificationRepository
	config config.VAPIDConfig
}

func NewPushHandler(repo repository.NotificationRepository, cfg config.VAPIDConfig) *PushHandler {
	return &PushHandler{
		repo:   repo,
		config: cfg,
	}
}

// GetPublicKey returns the VAPID public key to the frontend.
func (h *PushHandler) GetPublicKey(c *gin.Context) {
	if h.config.PublicKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "push notifications not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"publicKey": h.config.PublicKey}})
}

type subscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
	Keys     struct {
		P256dh string `json:"p256dh" binding:"required"`
		Auth   string `json:"auth" binding:"required"`
	} `json:"keys" binding:"required"`
}

// Subscribe saves the user's push subscription.
func (h *PushHandler) Subscribe(c *gin.Context) {
	empIDStr, _ := c.Get("employee_id")
	empID, _ := uuid.Parse(empIDStr.(string))

	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid subscription data"})
		return
	}

	sub := &models.PushSubscription{
		EmployeeID: empID,
		Endpoint:   req.Endpoint,
		P256dh:     req.Keys.P256dh,
		Auth:       req.Keys.Auth,
	}

	if err := h.repo.SavePushSubscription(c.Request.Context(), sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to save subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "subscribed successfully"}})
}
