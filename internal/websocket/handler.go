package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleWebSocket(m *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomID := c.Param("room_id")
		role := c.DefaultQuery("role", "spectator")
		playerID := c.Query("player_id")

		if playerID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "player_id required"})
			return
		}
		if role != "player" && role != "spectator" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be player or spectator"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[ws] upgrade failed: %v", err)
			return
		}

		client := m.JoinRoom(roomID, playerID, role, conn)
		if client == nil {
			return
		}
		defer m.LeaveRoom(roomID, client)

		conn.SetReadLimit(4096)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("[ws] read error: %v", err)
				}
				return
			}

			if role == "player" {
				m.HandlePlayerMessage(roomID, client, msg)
			}
		}
	}
}
