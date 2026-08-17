package notification

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// fakeNotifRepo returns a fixed subscription set. The embedded interface is nil,
// so any unexpected call panics rather than silently returning zero values.
type fakeNotifRepo struct {
	repository.NotificationRepository
	subs []models.PushSubscription
}

func (f *fakeNotifRepo) GetPushSubscriptionsByDepartmentID(context.Context, uuid.UUID) ([]models.PushSubscription, error) {
	return f.subs, nil
}

type fakeEmployeeRepo struct {
	repository.EmployeeRepository
	memberIDs []uuid.UUID
	err       error
}

func (f *fakeEmployeeRepo) GetActiveIDsByDepartment(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return f.memberIDs, f.err
}

// The regression for F-02: recipients used to be derived from the push
// subscription table, so anyone who had never granted browser notification
// permission received neither the push nor the socket message — nothing at all.
func TestDepartmentBroadcastReachesMembersWithoutPushSubscriptions(t *testing.T) {
	hub := DefaultWSHub
	department := uuid.New()

	subscribed := uuid.New()
	unsubscribedA := uuid.New()
	unsubscribedB := uuid.New()

	clients := map[uuid.UUID]*Client{}
	for _, id := range []uuid.UUID{subscribed, unsubscribedA, unsubscribedB} {
		c := &Client{ID: uuid.NewString(), Send: make(chan []byte, 4)}
		clients[id] = c
		hub.AddClient(id, c)
		defer hub.RemoveClient(id, c)
	}

	svc := &pushService{
		// Only one of the three has ever subscribed to web push.
		repo: &fakeNotifRepo{subs: []models.PushSubscription{{EmployeeID: subscribed}}},
		employeeRepo: &fakeEmployeeRepo{
			memberIDs: []uuid.UUID{subscribed, unsubscribedA, unsubscribedB},
		},
		config: config.VAPIDConfig{}, // unconfigured: no outbound push is attempted
	}

	if err := svc.SendToDepartment(context.Background(), department, "Announcement", "Body", "/"); err != nil {
		t.Fatalf("SendToDepartment: %v", err)
	}

	for id, c := range clients {
		select {
		case <-c.Send:
		default:
			t.Errorf("employee %s received no websocket message", id)
		}
	}
}

// A failure resolving membership must not suppress push delivery to the people
// who can still be reached.
func TestDepartmentBroadcastStillPushesWhenMembershipLookupFails(t *testing.T) {
	svc := &pushService{
		repo:         &fakeNotifRepo{subs: nil},
		employeeRepo: &fakeEmployeeRepo{err: context.DeadlineExceeded},
		config:       config.VAPIDConfig{},
	}

	if err := svc.SendToDepartment(context.Background(), uuid.New(), "t", "b", "/"); err != nil {
		t.Fatalf("a membership lookup failure should not fail the send: %v", err)
	}
}

// Each recipient must be messaged once even if the subscription table holds
// several rows for them (one per browser).
func TestDepartmentBroadcastDoesNotDuplicatePerSubscription(t *testing.T) {
	hub := DefaultWSHub
	employee := uuid.New()

	client := &Client{ID: uuid.NewString(), Send: make(chan []byte, 8)}
	hub.AddClient(employee, client)
	defer hub.RemoveClient(employee, client)

	svc := &pushService{
		// Three devices, one person.
		repo: &fakeNotifRepo{subs: []models.PushSubscription{
			{EmployeeID: employee}, {EmployeeID: employee}, {EmployeeID: employee},
		}},
		employeeRepo: &fakeEmployeeRepo{memberIDs: []uuid.UUID{employee}},
		config:       config.VAPIDConfig{},
	}

	if err := svc.SendToDepartment(context.Background(), uuid.New(), "t", "b", "/"); err != nil {
		t.Fatalf("SendToDepartment: %v", err)
	}

	count := 0
	for {
		select {
		case <-client.Send:
			count++
			continue
		default:
		}
		break
	}

	if count != 1 {
		t.Errorf("received %d websocket messages, want exactly 1", count)
	}
}
