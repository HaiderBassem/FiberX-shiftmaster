package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/models"
)

// capturingPush records real-time deliveries so the test can assert that
// persisting a notification also signals it.
type capturingPush struct {
	mu    sync.Mutex
	calls []pushCall
	done  chan struct{}
}

type pushCall struct {
	recipient uuid.UUID
	title     string
	message   string
	url       string
}

func (p *capturingPush) SendToEmployee(_ context.Context, id uuid.UUID, title, message, url string) error {
	p.mu.Lock()
	p.calls = append(p.calls, pushCall{id, title, message, url})
	p.mu.Unlock()
	select {
	case p.done <- struct{}{}:
	default:
	}
	return nil
}

func (p *capturingPush) SendToDepartment(context.Context, uuid.UUID, string, string, string) error {
	return nil
}
func (p *capturingPush) Broadcast(context.Context, string, string, string) error { return nil }

// fakeNotifRepo satisfies just enough of the repository for SendNotification;
// the unused methods return zero values.
type fakeNotifRepo struct {
	created []*models.Notification
}

func (f *fakeNotifRepo) Create(_ context.Context, n *models.Notification) error {
	f.created = append(f.created, n)
	return nil
}

func (f *fakeNotifRepo) GetByRecipient(context.Context, uuid.UUID) ([]models.Notification, error) {
	return nil, nil
}
func (f *fakeNotifRepo) GetUnread(context.Context, uuid.UUID) ([]models.Notification, error) {
	return nil, nil
}
func (f *fakeNotifRepo) GetUnreadCount(context.Context, uuid.UUID) (int, error) { return 0, nil }
func (f *fakeNotifRepo) MarkAsRead(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeNotifRepo) MarkAllAsRead(context.Context, uuid.UUID) error         { return nil }
func (f *fakeNotifRepo) Delete(context.Context, uuid.UUID) error                { return nil }
func (f *fakeNotifRepo) SavePushSubscription(context.Context, *models.PushSubscription) error {
	return nil
}
func (f *fakeNotifRepo) GetPushSubscriptionsByEmployeeID(context.Context, uuid.UUID) ([]models.PushSubscription, error) {
	return nil, nil
}
func (f *fakeNotifRepo) GetPushSubscriptionsByDepartmentID(context.Context, uuid.UUID) ([]models.PushSubscription, error) {
	return nil, nil
}
func (f *fakeNotifRepo) GetAllPushSubscriptions(context.Context) ([]models.PushSubscription, error) {
	return nil, nil
}
func (f *fakeNotifRepo) DeletePushSubscription(context.Context, string) error { return nil }

// Every persisted notification must be signalled in real time: before this
// contract, only services that individually remembered to push did, so swap
// and schedule notifications sat invisible until the next poll.
func TestSendNotificationDeliversInRealTime(t *testing.T) {
	push := &capturingPush{done: make(chan struct{}, 1)}
	repo := &fakeNotifRepo{}
	svc := NewNotificationService(repo, push)

	recipient := uuid.New()
	msg := "your swap was approved"
	url := "/swaps"
	err := svc.SendNotification(context.Background(), &models.Notification{
		RecipientID: recipient,
		Title:       "Swap Approved",
		Message:     &msg,
		ActionUrl:   &url,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("persisted %d notifications, want 1", len(repo.created))
	}

	select {
	case <-push.done:
	case <-time.After(2 * time.Second):
		t.Fatal("no real-time delivery within 2s of persisting")
	}

	push.mu.Lock()
	defer push.mu.Unlock()
	call := push.calls[0]
	if call.recipient != recipient || call.title != "Swap Approved" || call.message != msg || call.url != "/swaps" {
		t.Errorf("delivered %+v, want recipient/title/message/url to mirror the row", call)
	}
}

func TestSendNotificationDefaultsActionURL(t *testing.T) {
	push := &capturingPush{done: make(chan struct{}, 1)}
	svc := NewNotificationService(&fakeNotifRepo{}, push)

	if err := svc.SendNotification(context.Background(), &models.Notification{
		RecipientID: uuid.New(),
		Title:       "t",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
	select {
	case <-push.done:
	case <-time.After(2 * time.Second):
		t.Fatal("no delivery")
	}
	push.mu.Lock()
	defer push.mu.Unlock()
	if push.calls[0].url != "/notifications" {
		t.Errorf("url = %q, want /notifications when the row has none", push.calls[0].url)
	}
}

// A nil push service is a supported configuration and must not panic.
func TestSendNotificationWithoutPushService(t *testing.T) {
	svc := NewNotificationService(&fakeNotifRepo{}, nil)
	if err := svc.SendNotification(context.Background(), &models.Notification{
		RecipientID: uuid.New(),
		Title:       "t",
	}); err != nil {
		t.Fatalf("send: %v", err)
	}
}
