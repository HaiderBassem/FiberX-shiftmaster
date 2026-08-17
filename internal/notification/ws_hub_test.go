package notification

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

func newHub() *WSHub {
	return &WSHub{clients: make(map[uuid.UUID]map[*Client]bool)}
}

// newStalledClient builds a client whose send buffer is already full, standing in
// for a peer that has stopped reading: a suspended tab, or a dropped connection
// the TCP stack has not yet given up on. Conn is nil, which evict tolerates.
func newStalledClient() *Client {
	c := &Client{ID: uuid.NewString(), Send: make(chan []byte, 1)}
	c.Send <- []byte("occupies the only slot")
	return c
}

func newHealthyClient() *Client {
	return &Client{ID: uuid.NewString(), Send: make(chan []byte, 8)}
}

func TestDeliverReportsFullBuffer(t *testing.T) {
	healthy := newHealthyClient()
	if !deliver(healthy, []byte("x")) {
		t.Error("delivery to a client with a free buffer should succeed")
	}

	stalled := newStalledClient()
	if deliver(stalled, []byte("x")) {
		t.Error("delivery to a client with a full buffer should fail rather than block")
	}
}

// The original code discarded the write and left the client registered, so it
// silently missed every later message while holding a connection, a goroutine and
// a 256-slot channel for the lifetime of the process.
func TestSendToEmployeeEvictsStalledClient(t *testing.T) {
	hub := newHub()
	employee := uuid.New()

	stalled := newStalledClient()
	hub.AddClient(employee, stalled)

	if hub.ConnectedEmployees() != 1 {
		t.Fatalf("setup failed: hub reports %d employees", hub.ConnectedEmployees())
	}

	hub.SendToEmployee(employee, PushPayload{Title: "t", Body: "b"})

	if hub.ConnectedEmployees() != 0 {
		t.Fatal("a stalled client was left registered instead of being evicted")
	}
}

// Evicting one connection must not disturb the same employee's other sessions,
// e.g. a second browser tab that is keeping up.
func TestEvictionSparesHealthyClients(t *testing.T) {
	hub := newHub()
	employee := uuid.New()

	healthy := newHealthyClient()
	stalled := newStalledClient()
	hub.AddClient(employee, healthy)
	hub.AddClient(employee, stalled)

	hub.SendToEmployee(employee, PushPayload{Title: "t"})

	hub.mu.RLock()
	remaining := len(hub.clients[employee])
	_, healthyStillThere := hub.clients[employee][healthy]
	hub.mu.RUnlock()

	if remaining != 1 || !healthyStillThere {
		t.Fatalf("expected only the stalled client to be evicted, %d client(s) remain", remaining)
	}

	select {
	case <-healthy.Send:
	default:
		t.Error("the healthy client did not receive the payload")
	}
}

func TestBroadcastEvictsStalledClients(t *testing.T) {
	hub := newHub()

	healthyEmp, stalledEmp := uuid.New(), uuid.New()
	healthy := newHealthyClient()
	hub.AddClient(healthyEmp, healthy)
	hub.AddClient(stalledEmp, newStalledClient())

	hub.Broadcast(PushPayload{Title: "everyone"})

	if hub.ConnectedEmployees() != 1 {
		t.Fatalf("expected the stalled client to be evicted, %d employees remain", hub.ConnectedEmployees())
	}

	hub.mu.RLock()
	_, ok := hub.clients[healthyEmp]
	hub.mu.RUnlock()
	if !ok {
		t.Error("the wrong client was evicted")
	}
}

func TestRemoveClientIsIdempotent(t *testing.T) {
	hub := newHub()
	employee := uuid.New()
	client := newHealthyClient()

	hub.AddClient(employee, client)

	// Eviction removes it, and ServeWS's deferred cleanup removes it again. A
	// second close of the send channel here would panic.
	hub.RemoveClient(employee, client)
	hub.RemoveClient(employee, client)

	if hub.ConnectedEmployees() != 0 {
		t.Error("client was not removed")
	}
}

func TestSendToEmployeesFansOutToEveryRecipient(t *testing.T) {
	hub := newHub()

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	clients := make([]*Client, len(ids))
	for i, id := range ids {
		clients[i] = newHealthyClient()
		hub.AddClient(id, clients[i])
	}

	hub.SendToEmployees(ids, PushPayload{Title: "department wide"})

	for i, c := range clients {
		select {
		case <-c.Send:
		default:
			t.Errorf("recipient %d did not receive the payload", i)
		}
	}
}

// An employee with no socket open must not cause trouble for the rest.
func TestSendToEmployeesToleratesDisconnectedRecipients(t *testing.T) {
	hub := newHub()

	connected := uuid.New()
	client := newHealthyClient()
	hub.AddClient(connected, client)

	hub.SendToEmployees([]uuid.UUID{uuid.New(), connected, uuid.New()}, PushPayload{Title: "x"})

	select {
	case <-client.Send:
	default:
		t.Error("the connected recipient was skipped")
	}
}

func TestConcurrentSendAndRemove(t *testing.T) {
	hub := newHub()
	employee := uuid.New()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		client := newHealthyClient()
		hub.AddClient(employee, client)

		wg.Add(2)
		go func() {
			defer wg.Done()
			hub.SendToEmployee(employee, PushPayload{Title: "concurrent"})
		}()
		go func() {
			defer wg.Done()
			hub.RemoveClient(employee, client)
		}()
	}
	wg.Wait()

	if hub.ConnectedEmployees() != 0 {
		t.Errorf("expected every client to be removed, %d employees remain", hub.ConnectedEmployees())
	}
}
