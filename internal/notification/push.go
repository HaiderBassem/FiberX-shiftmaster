package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

type PushService interface {
	SendToEmployee(ctx context.Context, employeeID uuid.UUID, title, message, url string) error
	SendToDepartment(ctx context.Context, departmentID uuid.UUID, title, message, url string) error
	Broadcast(ctx context.Context, title, message, url string) error
}

type pushService struct {
	repo   repository.NotificationRepository
	config config.VAPIDConfig
}

func NewPushService(repo repository.NotificationRepository, cfg config.VAPIDConfig) PushService {
	return &pushService{
		repo:   repo,
		config: cfg,
	}
}

// PushPayload defines the JSON structure expected by our Service Worker
type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon"`
	Url   string `json:"url"`
}

func (s *pushService) send(ctx context.Context, subs []models.PushSubscription, payload PushPayload) error {
	if len(subs) == 0 {
		return nil
	}

	if s.config.PublicKey == "" || s.config.PrivateKey == "" {
		log.Println("Push notifications skipped: VAPID keys not configured")
		return nil
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		// Send notification
		res, err := webpush.SendNotification(payloadBytes, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}, &webpush.Options{
			Subscriber:      s.config.Subject,
			VAPIDPublicKey:  s.config.PublicKey,
			VAPIDPrivateKey: s.config.PrivateKey,
			TTL:             30, // 30 seconds TTL for fast delivery
		})

		if err != nil {
			log.Printf("Failed to send push to endpoint %s: %v", sub.Endpoint, err)
			continue
		}
		defer res.Body.Close()

		if res.StatusCode == 410 || res.StatusCode == 404 {
			// Subscription is no longer valid, delete it
			_ = s.repo.DeletePushSubscription(ctx, sub.Endpoint)
		}
	}
	return nil
}

func (s *pushService) SendToEmployee(ctx context.Context, employeeID uuid.UUID, title, message, url string) error {
	subs, err := s.repo.GetPushSubscriptionsByEmployeeID(ctx, employeeID)
	if err != nil {
		return fmt.Errorf("fetch subscriptions: %w", err)
	}

	return s.send(ctx, subs, PushPayload{
		Title: title,
		Body:  message,
		Icon:  "/icon-192x192.png",
		Url:   url,
	})
}

func (s *pushService) SendToDepartment(ctx context.Context, departmentID uuid.UUID, title, message, url string) error {
	subs, err := s.repo.GetPushSubscriptionsByDepartmentID(ctx, departmentID)
	if err != nil {
		return fmt.Errorf("fetch subscriptions: %w", err)
	}

	return s.send(ctx, subs, PushPayload{
		Title: title,
		Body:  message,
		Icon:  "/icon-192x192.png",
		Url:   url,
	})
}

func (s *pushService) Broadcast(ctx context.Context, title, message, url string) error {
	subs, err := s.repo.GetAllPushSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("fetch subscriptions: %w", err)
	}

	return s.send(ctx, subs, PushPayload{
		Title: title,
		Body:  message,
		Icon:  "/icon-192x192.png",
		Url:   url,
	})
}
