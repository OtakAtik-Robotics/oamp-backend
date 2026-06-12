package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type EventDisplayHub struct {
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

var DefaultEventDisplayHub = &EventDisplayHub{
	clients: make(map[*websocket.Conn]bool),
}

func (h *EventDisplayHub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
}

func (h *EventDisplayHub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

func (h *EventDisplayHub) Broadcast(eventType string, payload map[string]interface{}) {
	msg := map[string]interface{}{
		"type":    eventType,
		"data":    payload,
		"time":    time.Now().Unix(),
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	var dead []*websocket.Conn
	for conn := range h.clients {
		conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		err := conn.WriteMessage(websocket.TextMessage, raw)
		if err != nil {
			log.Printf("[ws-event] write error, marking dead: %v", err)
			dead = append(dead, conn)
		}
	}
	h.mu.RUnlock()

	// Remove dead clients (needs write lock, done outside RLock)
	if len(dead) > 0 {
		h.mu.Lock()
		for _, conn := range dead {
			delete(h.clients, conn)
		}
		h.mu.Unlock()
		log.Printf("[ws-event] removed %d dead clients, %d remaining", len(dead), len(h.clients))
	}
}

func HandleEventDisplay(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws-event] upgrade failed: %v", err)
		return
	}

	DefaultEventDisplayHub.Add(conn)
	defer DefaultEventDisplayHub.Remove(conn)

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send initial ping with write deadline
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	pingMsg, _ := json.Marshal(map[string]interface{}{
		"type": "connected",
		"data": map[string]interface{}{"message": "EventDisplay connected"},
	})
	conn.WriteMessage(websocket.TextMessage, pingMsg)

	// Keep-alive ticker
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range ticker.C {
			h := DefaultEventDisplayHub
			h.mu.RLock()
			_, exists := h.clients[conn]
			h.mu.RUnlock()
			if !exists {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			return
		}
	}
}
