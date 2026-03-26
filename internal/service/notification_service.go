package service

import (
	"context"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// NotificationService handles creating and managing notifications.
type NotificationService struct {
	notifRepo repository.NotificationRepository
}

func NewNotificationService(notifRepo repository.NotificationRepository) *NotificationService {
	return &NotificationService{notifRepo: notifRepo}
}

// SendNotification creates a new notification.
func (s *NotificationService) SendNotification(ctx context.Context, n *models.Notification) error {
	return s.notifRepo.Create(ctx, n)
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

// MarkAsRead marks a single notification as read.
func (s *NotificationService) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	return s.notifRepo.MarkAsRead(ctx, id)
}

// MarkAllAsRead marks all unread notifications as read for a recipient.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) error {
	return s.notifRepo.MarkAllAsRead(ctx, recipientID)
}

// Delete removes a notification.
func (s *NotificationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.notifRepo.Delete(ctx, id)
}
