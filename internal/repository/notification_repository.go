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
	
	// Web Push Subscriptions
	SavePushSubscription(ctx context.Context, sub *models.PushSubscription) error
	GetPushSubscriptionsByEmployeeID(ctx context.Context, employeeID uuid.UUID) ([]models.PushSubscription, error)
	GetPushSubscriptionsByDepartmentID(ctx context.Context, departmentID uuid.UUID) ([]models.PushSubscription, error)
	GetAllPushSubscriptions(ctx context.Context) ([]models.PushSubscription, error)
	DeletePushSubscription(ctx context.Context, endpoint string) error
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

// ── Web Push Subscriptions ──

func (r *notificationRepo) SavePushSubscription(ctx context.Context, sub *models.PushSubscription) error {
	query := `
		INSERT INTO push_subscriptions (employee_id, endpoint, p256dh, auth)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (employee_id, endpoint) DO UPDATE 
		SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth, updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(ctx, query, sub.EmployeeID, sub.Endpoint, sub.P256dh, sub.Auth)
	return err
}

func (r *notificationRepo) GetPushSubscriptionsByEmployeeID(ctx context.Context, employeeID uuid.UUID) ([]models.PushSubscription, error) {
	query := `SELECT id, employee_id, endpoint, p256dh, auth, created_at, updated_at FROM push_subscriptions WHERE employee_id = $1`
	rows, err := r.db.Query(ctx, query, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.PushSubscription
	for rows.Next() {
		var s models.PushSubscription
		if err := rows.Scan(&s.ID, &s.EmployeeID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *notificationRepo) GetPushSubscriptionsByDepartmentID(ctx context.Context, departmentID uuid.UUID) ([]models.PushSubscription, error) {
	query := `
		SELECT ps.id, ps.employee_id, ps.endpoint, ps.p256dh, ps.auth, ps.created_at, ps.updated_at 
		FROM push_subscriptions ps
		JOIN employees e ON ps.employee_id = e.id
		WHERE e.department_id = $1 AND e.status = 'active'
	`
	rows, err := r.db.Query(ctx, query, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.PushSubscription
	for rows.Next() {
		var s models.PushSubscription
		if err := rows.Scan(&s.ID, &s.EmployeeID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *notificationRepo) GetAllPushSubscriptions(ctx context.Context) ([]models.PushSubscription, error) {
	query := `
		SELECT ps.id, ps.employee_id, ps.endpoint, ps.p256dh, ps.auth, ps.created_at, ps.updated_at 
		FROM push_subscriptions ps
		JOIN employees e ON ps.employee_id = e.id
		WHERE e.status = 'active'
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.PushSubscription
	for rows.Next() {
		var s models.PushSubscription
		if err := rows.Scan(&s.ID, &s.EmployeeID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *notificationRepo) DeletePushSubscription(ctx context.Context, endpoint string) error {
	query := `DELETE FROM push_subscriptions WHERE endpoint = $1`
	_, err := r.db.Exec(ctx, query, endpoint)
	return err
}
