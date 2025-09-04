package ws

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]struct{})}
}

func (h *Hub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

func (h *Hub) Broadcast(message []byte) {
	h.mu.Lock()
	for conn := range h.clients {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := conn.Write(ctx, websocket.MessageText, message)
		cancel()
		if err != nil {
			_ = conn.Close(websocket.StatusNormalClosure, "")
			delete(h.clients, conn)
		}
	}
	h.mu.Unlock()
}
