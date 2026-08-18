package service

import (
	"context"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/notification"
	"shiftmaster-backend/internal/repository"
)

// NotificationService handles creating and managing notifications.
type NotificationService struct {
	notifRepo   repository.NotificationRepository
	pushService notification.PushService
}

func NewNotificationService(notifRepo repository.NotificationRepository, pushService notification.PushService) *NotificationService {
	return &NotificationService{notifRepo: notifRepo, pushService: pushService}
}

// SendNotification creates a notification and delivers it in real time.
//
// Persisting and signalling live together so every notification row reaches a
// connected client over the WebSocket (and web push when subscribed) the
// moment it exists. Before this, each domain service decided for itself:
// leaves pushed, swaps and schedules did not, so their notifications sat
// invisible until the 20-second poll.
func (s *NotificationService) SendNotification(ctx context.Context, n *models.Notification) error {
	if err := s.notifRepo.Create(ctx, n); err != nil {
		return err
	}

	if s.pushService != nil {
		message := ""
		if n.Message != nil {
			message = *n.Message
		}
		url := "/notifications"
		if n.ActionUrl != nil && *n.ActionUrl != "" {
			url = *n.ActionUrl
		}
		// Delivery is best-effort and must not block or fail the caller; the
		// row is already durable and the poll will find it regardless.
		go func(recipient uuid.UUID, title, message, url string) {
			_ = s.pushService.SendToEmployee(context.Background(), recipient, title, message, url)
		}(n.RecipientID, n.Title, message, url)
	}
	return nil
}

// GetNotifications returns all notifications for a recipient (most recent first).
func (s *NotificationService) GetNotifications(ctx context.Context, recipientID uuid.UUID) ([]models.Notification, error) {
	return s.notifRepo.GetByRecipient(ctx, recipientID)
}

// GetUnread returns unread notifications for a recipient.
func (s *NotificationService) GetUnread(ctx context.Context, recipientID uuid.UUID) ([]models.Notification, error) {
	return s.notifRepo.GetUnread(ctx, recipientID)
}

// GetUnreadCount returns the count of unread notifications.
func (s *NotificationService) GetUnreadCount(ctx context.Context, recipientID uuid.UUID) (int, error) {
	return s.notifRepo.GetUnreadCount(ctx, recipientID)
}

// MarkAsRead marks one of the recipient's own notifications as read.
func (s *NotificationService) MarkAsRead(ctx context.Context, id, recipientID uuid.UUID) error {
	return s.notifRepo.MarkAsRead(ctx, id, recipientID)
}

// MarkAllAsRead marks all unread notifications as read for a recipient.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) error {
	return s.notifRepo.MarkAllAsRead(ctx, recipientID)
}

// Delete removes a notification.
func (s *NotificationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.notifRepo.Delete(ctx, id)
}
