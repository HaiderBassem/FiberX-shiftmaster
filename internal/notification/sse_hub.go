package notification

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// SSEHub manages Server-Sent Events connections
type SSEHub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[chan string]bool
}

var DefaultSSEHub = &SSEHub{
	clients: make(map[uuid.UUID]map[chan string]bool),
}

func (h *SSEHub) AddClient(employeeID uuid.UUID, clientChan chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[employeeID] == nil {
		h.clients[employeeID] = make(map[chan string]bool)
	}
	h.clients[employeeID][clientChan] = true
}

func (h *SSEHub) RemoveClient(employeeID uuid.UUID, clientChan chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[employeeID]; ok {
		delete(clients, clientChan)
		if len(clients) == 0 {
			delete(h.clients, employeeID)
		}
	}
}

func (h *SSEHub) SendToEmployee(employeeID uuid.UUID, payload PushPayload) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[employeeID]; ok {
		data, _ := json.Marshal(payload)
		for clientChan := range clients {
			select {
			case clientChan <- string(data):
			default:
				// Channel blocked, skip
			}
		}
	}
}

// Broadcast sends to all connected clients
func (h *SSEHub) Broadcast(payload PushPayload) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, _ := json.Marshal(payload)
	for _, clients := range h.clients {
		for clientChan := range clients {
			select {
			case clientChan <- string(data):
			default:
			}
		}
	}
}
