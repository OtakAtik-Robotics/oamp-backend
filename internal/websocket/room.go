package websocket

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
)

type Client struct {
	Conn     *websocket.Conn
	PlayerID string
	Role     string // "player" or "spectator"
	Send     chan []byte
}

type Room struct {
	ID         string
	Players    map[string]*Client // max 2
	Spectators map[string]*Client
	GameOvers  int // track how many players finished
	mu         sync.RWMutex
}

func newRoom(id string) *Room {
	return &Room{
		ID:         id,
		Players:    make(map[string]*Client),
		Spectators: make(map[string]*Client),
	}
}

func (r *Room) addClient(c *Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c.Role == "player" {
		if len(r.Players) >= 2 {
			return false
		}
		r.Players[c.PlayerID] = c
	} else {
		r.Spectators[c.PlayerID] = c
	}
	return true
}

func (r *Room) removeClient(c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c.Role == "player" {
		delete(r.Players, c.PlayerID)
	} else {
		delete(r.Spectators, c.PlayerID)
	}
	close(c.Send)
}

func (r *Room) broadcastToSpectators(payload []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, spec := range r.Spectators {
		select {
		case spec.Send <- payload:
		default:
			log.Printf("[ws] spectator %s send buffer full, dropping", spec.PlayerID)
		}
	}
}

func (r *Room) broadcastAll(payload []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.Players {
		select {
		case c.Send <- payload:
		default:
		}
	}
	for _, c := range r.Spectators {
		select {
		case c.Send <- payload:
		default:
		}
	}
}

func (r *Room) writePump(c *Client) {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

type GameMessage struct {
	Type       string  `json:"type"`       // "score_update", "join", "leave", "GAME_OVER"
	PlayerID   string  `json:"player_id"`
	GameScore  int     `json:"game_score,omitempty"`
	BlocksHit  int     `json:"blocks_hit,omitempty"`
	PlayDuration float64 `json:"play_duration,omitempty"`
}

type Manager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

func (m *Manager) JoinRoom(roomID, playerID, role string, conn *websocket.Conn) *Client {
	m.mu.Lock()
	room, ok := m.rooms[roomID]
	if !ok {
		room = newRoom(roomID)
		m.rooms[roomID] = room
	}
	m.mu.Unlock()

	client := &Client{
		Conn:     conn,
		PlayerID: playerID,
		Role:     role,
		Send:     make(chan []byte, 64),
	}

	if !room.addClient(client) {
		conn.WriteJSON(map[string]string{"error": "room full"})
		conn.Close()
		return nil
	}

	go room.writePump(client)

	// Notify spectators of join
	joinMsg, _ := json.Marshal(GameMessage{
		Type:     "join",
		PlayerID: playerID,
	})
	room.broadcastToSpectators(joinMsg)

	return client
}

func (m *Manager) HandlePlayerMessage(roomID string, client *Client, raw []byte) {
	m.mu.RLock()
	room, ok := m.rooms[roomID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	var msg GameMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}
	msg.PlayerID = client.PlayerID

	if msg.Type == "GAME_OVER" {
		m.handleGameOver(room, client, &msg)
		return
	}

	// Player msg → forward exact JSON to spectators
	room.broadcastToSpectators(raw)
}

func (m *Manager) handleGameOver(room *Room, client *Client, msg *GameMessage) {
	// Broadcast GAME_OVER to everyone (players + spectators)
	broadcast, _ := json.Marshal(msg)
	room.broadcastAll(broadcast)

	// Persist to DB
	if config.DB != nil {
		participantID, err := strconv.Atoi(client.PlayerID)
		if err == nil {
			result := model.PureGameResult{
				ParticipantID:      uint(participantID),
				GameScore:          msg.GameScore,
				BlocksHit:          msg.BlocksHit,
				PlayDuration:       msg.PlayDuration,
				HandTrackingStatus: "active",
				Timestamp:          time.Now(),
			}
			if err := config.DB.Create(&result).Error; err != nil {
				log.Printf("[ws] failed to persist game result for player %s: %v", client.PlayerID, err)
			} else {
				log.Printf("[ws] game result persisted for player %s, score=%d", client.PlayerID, msg.GameScore)
			}
		}
	}

	// Track game-over count, cleanup room when both players finish
	room.mu.Lock()
	room.GameOvers++
	gameOvers := room.GameOvers
	room.mu.Unlock()

	if gameOvers >= 2 {
		m.destroyRoom(room.ID)
	}
}

func (m *Manager) destroyRoom(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return
	}

	// Close all connections
	room.mu.Lock()
	for _, c := range room.Players {
		close(c.Send)
	}
	for _, c := range room.Spectators {
		close(c.Send)
	}
	room.mu.Unlock()

	delete(m.rooms, roomID)
	log.Printf("[ws] room %s destroyed after match completion", roomID)
}

func (m *Manager) LeaveRoom(roomID string, client *Client) {
	m.mu.RLock()
	room, ok := m.rooms[roomID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	leaveMsg, _ := json.Marshal(GameMessage{
		Type:     "leave",
		PlayerID: client.PlayerID,
	})
	room.broadcastToSpectators(leaveMsg)
	room.removeClient(client)

	// Cleanup empty room
	m.mu.Lock()
	room.mu.RLock()
	empty := len(room.Players) == 0 && len(room.Spectators) == 0
	room.mu.RUnlock()
	if empty {
		delete(m.rooms, roomID)
	}
	m.mu.Unlock()
}
