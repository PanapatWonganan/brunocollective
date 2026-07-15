package services

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/websocket/v2"
)

// ChatHub fans chat events out to every connected admin WebSocket so the
// inbox updates in real time. Writes are serialized per connection.
type ChatHub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]*sync.Mutex
}

func NewChatHub() *ChatHub {
	return &ChatHub{clients: make(map[*websocket.Conn]*sync.Mutex)}
}

func (h *ChatHub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = &sync.Mutex{}
}

func (h *ChatHub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
}

// Broadcast sends payload (JSON-marshalled) to all connected admins.
// Dead connections are dropped silently — the read loop cleans them up.
func (h *ChatHub) Broadcast(payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.mu.Lock()
	conns := make(map[*websocket.Conn]*sync.Mutex, len(h.clients))
	for c, m := range h.clients {
		conns[c] = m
	}
	h.mu.Unlock()

	for conn, writeMu := range conns {
		writeMu.Lock()
		_ = conn.WriteMessage(websocket.TextMessage, data)
		writeMu.Unlock()
	}
}
