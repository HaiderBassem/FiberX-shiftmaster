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

func (h *WSHub) SendToEmployee(employeeID uuid.UUID, payload PushPayload) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[employeeID]; ok {
		data, _ := json.Marshal(payload)
		for client := range clients {
			select {
			case client.Send <- data:
			default:
				// If send buffer is full, remove the client
				// It will be removed in a goroutine to prevent deadlock
			}
		}
	}
}

func (h *WSHub) Broadcast(payload PushPayload) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, _ := json.Marshal(payload)
	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
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
			w.Write(message)

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
		conn.Close()
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
