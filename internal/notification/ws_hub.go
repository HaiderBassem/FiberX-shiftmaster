package notification

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for the websocket
	},
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
