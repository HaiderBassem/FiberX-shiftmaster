package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/pkg/database"
)

// NotificationRepository defines the interface for notification data access.
type NotificationRepository interface {
	GetByRecipient(ctx context.Context, recipientID uuid.UUID) ([]models.Notification, error)
	GetUnread(ctx context.Context, recipientID uuid.UUID) ([]models.Notification, error)
	GetUnreadCount(ctx context.Context, recipientID uuid.UUID) (int, error)
	Create(ctx context.Context, n *models.Notification) error
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type notificationRepo struct {
	db *database.DB
}

func NewNotificationRepository(db *database.DB) NotificationRepository {
	return &notificationRepo{db: db}
}

const notifColumns = `id, recipient_id, sender_id, type, title, message, related_entity_type,
	related_entity_id, priority, is_read, read_at, action_url, created_at`

func (r *notificationRepo) GetByRecipient(ctx context.Context, recipientID uuid.UUID) ([]models.Notification, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+notifColumns+` FROM notifications WHERE recipient_id = $1 ORDER BY created_at DESC LIMIT 50`, recipientID)
	if err != nil {
		return nil, fmt.Errorf("get notifications: %w", err)
	}
	defer rows.Close()
	return r.scanNotifications(rows)
}

func (r *notificationRepo) GetUnread(ctx context.Context, recipientID uuid.UUID) ([]models.Notification, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+notifColumns+` FROM notifications WHERE recipient_id = $1 AND is_read = false ORDER BY created_at DESC`, recipientID)
	if err != nil {
		return nil, fmt.Errorf("get unread notifications: %w", err)
	}
	defer rows.Close()
	return r.scanNotifications(rows)
}

func (r *notificationRepo) GetUnreadCount(ctx context.Context, recipientID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE recipient_id = $1 AND is_read = false`, recipientID,
	).Scan(&count)
	return count, err
}

func (r *notificationRepo) scanNotifications(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]models.Notification, error) {
	var notifs []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.RecipientID, &n.SenderID, &n.Type, &n.Title, &n.Message,
			&n.RelatedEntityType, &n.RelatedEntityID, &n.Priority, &n.IsRead, &n.ReadAt,
			&n.ActionUrl, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

func (r *notificationRepo) Create(ctx context.Context, n *models.Notification) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO notifications (recipient_id, sender_id, type, title, message,
			related_entity_type, related_entity_id, priority, action_url)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, is_read, created_at`,
		n.RecipientID, n.SenderID, n.Type, n.Title, n.Message,
		n.RelatedEntityType, n.RelatedEntityID, n.Priority, n.ActionUrl,
	).Scan(&n.ID, &n.IsRead, &n.CreatedAt)
}

func (r *notificationRepo) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET is_read=true, read_at=CURRENT_TIMESTAMP WHERE id=$1`, id)
	return err
}

func (r *notificationRepo) MarkAllAsRead(ctx context.Context, recipientID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET is_read=true, read_at=CURRENT_TIMESTAMP WHERE recipient_id=$1 AND is_read=false`, recipientID)
	return err
}

func (r *notificationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM notifications WHERE id=$1`, id)
	return err
}
