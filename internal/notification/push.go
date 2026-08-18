package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// pushHTTPClient bounds every web-push request; one per process is enough.
var pushHTTPClient = &http.Client{Timeout: 15 * time.Second}

type PushService interface {
	SendToEmployee(ctx context.Context, employeeID uuid.UUID, title, message, url string) error
	SendToDepartment(ctx context.Context, departmentID uuid.UUID, title, message, url string) error
	Broadcast(ctx context.Context, title, message, url string) error
}

type pushService struct {
	repo         repository.NotificationRepository
	employeeRepo repository.EmployeeRepository
	config       config.VAPIDConfig
}

func NewPushService(repo repository.NotificationRepository, employeeRepo repository.EmployeeRepository, cfg config.VAPIDConfig) PushService {
	return &pushService{
		repo:         repo,
		employeeRepo: employeeRepo,
		config:       cfg,
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
			TTL:             86400, // 24 hours TTL
			Urgency:         webpush.UrgencyHigh,
			// The library's default client has no timeout; a hung push
			// endpoint would pin a delivery goroutine forever.
			HTTPClient:      pushHTTPClient,
		})

		if err != nil {
			log.Printf("Failed to send push to endpoint %s: %v", sub.Endpoint, err)
			continue
		}

		if res.StatusCode == 410 || res.StatusCode == 404 {
			// Subscription is no longer valid, delete it
			_ = s.repo.DeletePushSubscription(ctx, sub.Endpoint)
		}
		// Closed per iteration; a deferred close here would hold every response
		// body open until the whole fan-out finishes.
		res.Body.Close()
	}
	return nil
}

func (s *pushService) SendToEmployee(ctx context.Context, employeeID uuid.UUID, title, message, url string) error {
	subs, err := s.repo.GetPushSubscriptionsByEmployeeID(ctx, employeeID)
	if err != nil {
		return fmt.Errorf("fetch subscriptions: %w", err)
	}

	// Trigger WS for real-time in-app delivery
	DefaultWSHub.SendToEmployee(employeeID, PushPayload{
		Title: title,
		Body:  message,
		Icon:  "/icon-192x192.png",
		Url:   url,
	})

	return s.send(ctx, subs, PushPayload{
		Title: title,
		Body:  message,
		Icon:  "/icon-192x192.png",
		Url:   url,
	})
}

// SendToDepartment delivers to every active member of a department.
//
// The two channels have deliberately different recipient sets. WebSocket delivery
// is driven by department membership, because an employee with the app open
// should see the notification whether or not they ever granted browser
// notification permission. Push delivery is necessarily limited to the employees
// who do have a subscription. Deriving both from the subscription table, as this
// did before, meant anyone without a subscription received nothing at all.
func (s *pushService) SendToDepartment(ctx context.Context, departmentID uuid.UUID, title, message, url string) error {
	payload := PushPayload{
		Title: title,
		Body:  message,
		Icon:  "/icon-192x192.png",
		Url:   url,
	}

	if s.employeeRepo != nil {
		memberIDs, err := s.employeeRepo.GetActiveIDsByDepartment(ctx, departmentID)
		if err != nil {
			// A failure here must not suppress push delivery.
			log.Printf("Push: failed to resolve department %s members for websocket delivery: %v", departmentID, err)
		} else {
			DefaultWSHub.SendToEmployees(memberIDs, payload)
		}
	}

	subs, err := s.repo.GetPushSubscriptionsByDepartmentID(ctx, departmentID)
	if err != nil {
		return fmt.Errorf("fetch subscriptions: %w", err)
	}

	return s.send(ctx, subs, payload)
}

func (s *pushService) Broadcast(ctx context.Context, title, message, url string) error {
	subs, err := s.repo.GetAllPushSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("fetch subscriptions: %w", err)
	}

	payload := PushPayload{
		Title: title,
		Body:  message,
		Icon:  "/icon-192x192.png",
		Url:   url,
	}

	DefaultWSHub.Broadcast(payload)

	return s.send(ctx, subs, payload)
}
