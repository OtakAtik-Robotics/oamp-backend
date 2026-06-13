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
	writeMu sync.Mutex // serializes writes to prevent concurrent gorilla/websocket writes
}

var DefaultEventDisplayHub = &EventDisplayHub{
	clients: make(map[*websocket.Conn]bool),
}

func (h *EventDisplayHub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

func (h *EventDisplayHub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
}

func (h *EventDisplayHub) writeText(conn *websocket.Conn, msg []byte) error {
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, msg)
}

func (h *EventDisplayHub) writePing(conn *websocket.Conn) error {
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	return conn.WriteMessage(websocket.PingMessage, nil)
}

func (h *EventDisplayHub) Broadcast(eventType string, payload map[string]interface{}) {
	msg := map[string]interface{}{
		"type": eventType,
		"data": payload,
		"time": time.Now().Unix(),
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	var dead []*websocket.Conn
	for conn := range h.clients {
		if err := h.writeText(conn, raw); err != nil {
			log.Printf("[ws-event] write error, marking dead: %v", err)
			dead = append(dead, conn)
		}
	}
	h.mu.RUnlock()

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
	defer conn.Close()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send initial connected message
	pingMsg, _ := json.Marshal(map[string]interface{}{
		"type": "connected",
		"data": map[string]interface{}{"message": "EventDisplay connected"},
	})
	DefaultEventDisplayHub.writeText(conn, pingMsg)

	// Keep-alive ticker — uses writeMu to safely write
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			h := DefaultEventDisplayHub
			h.mu.RLock()
			_, exists := h.clients[conn]
			h.mu.RUnlock()
			if !exists {
				return
			}
			if err := h.writePing(conn); err != nil {
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
