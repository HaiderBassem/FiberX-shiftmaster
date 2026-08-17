package notification

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// allowedOrigins holds the exact origins permitted to open a WebSocket. It is
// populated once at startup by ConfigureOrigins.
//
// The upgrade handshake is a plain HTTP request that carries the user's cookies and
// is not subject to the same-origin policy, so an unrestricted CheckOrigin lets any
// website on the internet open an authenticated socket on the visitor's behalf
// (cross-site WebSocket hijacking). Until ConfigureOrigins runs, every upgrade is
// refused rather than allowed: failing closed means a wiring mistake shows up as a
// broken socket in testing, not as an open door in production.
var (
	originMu       sync.RWMutex
	allowedOrigins []string
	originsSet     bool
)

// ConfigureOrigins sets the origins accepted by the WebSocket upgrader.
func ConfigureOrigins(origins []string) {
	originMu.Lock()
	defer originMu.Unlock()

	allowedOrigins = make([]string, 0, len(origins))
	for _, o := range origins {
		if trimmed := strings.TrimSpace(strings.TrimSuffix(o, "/")); trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}
	originsSet = true
}

// isOriginAllowed reports whether the handshake's Origin header is acceptable.
func isOriginAllowed(r *http.Request) bool {
	originMu.RLock()
	defer originMu.RUnlock()

	if !originsSet {
		log.Println("WS: rejecting upgrade, allowed origins are not configured")
		return false
	}

	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		// Browsers always send Origin on a WebSocket handshake. An absent header
		// means a non-browser client, which cannot be the victim of a cross-site
		// request, but also cannot be vouched for — refuse it.
		return false
	}

	origin = strings.TrimSuffix(origin, "/")
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}

	log.Printf("WS: rejected upgrade from disallowed origin %q", origin)
	return false
}

var upgrader = websocket.Upgrader{
	CheckOrigin:      isOriginAllowed,
	HandshakeTimeout: 10 * time.Second,
}

// Client represents a single websocket connection
type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
}

// WSHub maintains the set of active clients
type WSHub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]bool
}

var DefaultWSHub = &WSHub{
	clients: make(map[uuid.UUID]map[*Client]bool),
}

func (h *WSHub) AddClient(employeeID uuid.UUID, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[employeeID] == nil {
		h.clients[employeeID] = make(map[*Client]bool)
	}
	h.clients[employeeID][client] = true
}

func (h *WSHub) RemoveClient(employeeID uuid.UUID, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[employeeID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.Send)
			if len(clients) == 0 {
				delete(h.clients, employeeID)
			}
		}
	}
}

// deliver enqueues data for a client, reporting whether it was accepted.
//
// A full buffer means the peer has stopped reading: a suspended tab, a dropped
// connection the TCP stack has not given up on yet, or a client too slow to keep
// pace. Previously the write was discarded and the client left registered, so it
// silently missed every later message while holding a connection, a goroutine and
// a 256-slot channel for as long as the process lived. Such a client is now
// evicted instead, and reconnects on its own.
func deliver(client *Client, data []byte) bool {
	select {
	case client.Send <- data:
		return true
	default:
		return false
	}
}

// evict deregisters an unresponsive client and closes its connection.
//
// Callers must not hold h.mu: RemoveClient takes the write lock. RemoveClient is
// idempotent, so the deferred call in ServeWS that follows the connection close
// is a no-op, and the send channel is closed exactly once.
func (h *WSHub) evict(employeeID uuid.UUID, client *Client) {
	log.Printf("WS: evicting unresponsive client %s for employee %s", client.ID, employeeID)
	h.RemoveClient(employeeID, client)
	if client.Conn != nil {
		_ = client.Conn.Close()
	}
}

func (h *WSHub) SendToEmployee(employeeID uuid.UUID, payload PushPayload) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WS: failed to marshal payload: %v", err)
		return
	}

	// Collect stalled clients under the read lock, then evict outside it: evicting
	// closes a connection, which wakes ServeWS's reader, which calls RemoveClient
	// and takes the write lock. Doing that here would deadlock.
	var stalled []*Client

	h.mu.RLock()
	for client := range h.clients[employeeID] {
		if !deliver(client, data) {
			stalled = append(stalled, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range stalled {
		h.evict(employeeID, client)
	}
}

// SendToEmployees fans a payload out to several recipients.
func (h *WSHub) SendToEmployees(employeeIDs []uuid.UUID, payload PushPayload) {
	for _, id := range employeeIDs {
		h.SendToEmployee(id, payload)
	}
}

func (h *WSHub) Broadcast(payload PushPayload) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WS: failed to marshal payload: %v", err)
		return
	}

	type stalledClient struct {
		employeeID uuid.UUID
		client     *Client
	}
	var stalled []stalledClient

	h.mu.RLock()
	for employeeID, clients := range h.clients {
		for client := range clients {
			if !deliver(client, data) {
				stalled = append(stalled, stalledClient{employeeID, client})
			}
		}
	}
	h.mu.RUnlock()

	for _, s := range stalled {
		h.evict(s.employeeID, s.client)
	}
}

// ConnectedEmployees reports how many distinct employees currently hold a socket.
// Used by tests and for operational visibility.
func (h *WSHub) ConnectedEmployees() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				// The peer has gone; closing the writer would only mask it.
				return
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWS handles websocket requests from the peer.
func ServeWS(w http.ResponseWriter, r *http.Request, employeeID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS Upgrade error:", err)
		return
	}

	client := &Client{
		ID:   uuid.New().String(),
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	DefaultWSHub.AddClient(employeeID, client)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()

	// Initial connect message
	client.Send <- []byte(`{"type":"connected"}`)

	// Read pump to handle incoming messages (e.g. pongs) and detect disconnects
	defer func() {
		DefaultWSHub.RemoveClient(employeeID, client)
		_ = conn.Close()
	}()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS close error: %v", err)
			}
			break
		}
	}
}
