package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn       *websocket.Conn
	PlayerID   string
	PlayerName string
	Role       string // "player" or "spectator"
	PlayerNum  int    // 1 or 2 (only for role="player")
	Send       chan []byte
}

type Room struct {
	ID          string
	Players     map[string]*Client // max 2
	Spectators  map[string]*Client
	ReadyCount  int // how many players sent "player_ready" via WS
	GameOvers   int
	readyNum    map[int]bool // player_num that already sent ready (dedup)
	overNum     map[int]bool // player_num that already sent GAME_OVER (dedup)
	destroyed   bool // set before channels are closed
	mu          sync.RWMutex
}

func newRoom(id string) *Room {
	return &Room{
		ID:         id,
		Players:    make(map[string]*Client),
		Spectators: make(map[string]*Client),
		readyNum:   make(map[int]bool),
		overNum:    make(map[int]bool),
	}
}

func (r *Room) addClient(c *Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c.Role == "player" {
		if len(r.Players) >= 2 {
			return false
		}
		// Assign player number
		if c.PlayerNum == 0 {
			// Auto-assign: P1 if slot empty, else P2
			hasP1 := false
			for _, p := range r.Players {
				if p.PlayerNum == 1 {
					hasP1 = true
				}
			}
			if !hasP1 {
				c.PlayerNum = 1
			} else {
				c.PlayerNum = 2
			}
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
	if r.destroyed {
		return
	}

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
	if r.destroyed {
		return
	}

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
	Type        string   `json:"type"`
	PlayerID    string   `json:"player_id,omitempty"`
	PlayerName  string   `json:"player_name,omitempty"`
	PlayerNum   int      `json:"player_num,omitempty"`
	GameScore   int      `json:"game_score,omitempty"`
	BlocksHit   int      `json:"blocks_hit,omitempty"`
	Level       int      `json:"level,omitempty"`
	PlayDuration float64 `json:"play_duration,omitempty"`
	Winner      string   `json:"winner,omitempty"`
	P1Score     float64  `json:"p1_score,omitempty"`
	P2Score     float64  `json:"p2_score,omitempty"`
	Status      string   `json:"status,omitempty"`
	Player1Name string   `json:"player1_name,omitempty"`
	Player2Name string   `json:"player2_name,omitempty"`
	RoomID      string   `json:"room_id,omitempty"`
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

func (m *Manager) JoinRoom(roomID, playerID, playerName, role string, playerNum int, conn *websocket.Conn) *Client {
	m.mu.Lock()
	room, ok := m.rooms[roomID]
	if !ok {
		room = newRoom(roomID)
		m.rooms[roomID] = room
	}
	m.mu.Unlock()

	client := &Client{
		Conn:       conn,
		PlayerID:   playerID,
		PlayerName: playerName,
		Role:       role,
		PlayerNum:  playerNum,
		Send:       make(chan []byte, 64),
	}

	if !room.addClient(client) {
		conn.WriteJSON(map[string]string{"error": "room full"})
		conn.Close()
		return nil
	}

	go room.writePump(client)

	joinMsg, _ := json.Marshal(GameMessage{
		Type:       "join",
		PlayerID:   playerID,
		PlayerName: playerName,
		PlayerNum:  client.PlayerNum,
	})
	room.broadcastAll(joinMsg)

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
	msg.PlayerNum = client.PlayerNum
	msg.PlayerName = client.PlayerName

	switch msg.Type {
	case "player_ready":
		m.handlePlayerReady(room, client)

	case "score_update":
		broadcast, _ := json.Marshal(msg)
		room.broadcastAll(broadcast)
		relayToEventDisplay(&msg, room.ID)

	case "level_start":
		broadcast, _ := json.Marshal(msg)
		room.broadcastAll(broadcast)
		relayToEventDisplay(&msg, room.ID)

	case "GAME_OVER":
		m.handleGameOver(room, client, &msg)

	default:
		broadcast, _ := json.Marshal(msg)
		room.broadcastToSpectators(broadcast)
	}
}

func (m *Manager) handlePlayerReady(room *Room, client *Client) {
	room.mu.Lock()
	// Dedup: each player_num can only ready once
	if room.readyNum[client.PlayerNum] {
		room.mu.Unlock()
		return
	}
	room.readyNum[client.PlayerNum] = true
	room.ReadyCount++
	readyCount := room.ReadyCount
	room.mu.Unlock()

	if readyCount >= 2 {
		startMsg, _ := json.Marshal(GameMessage{
			Type:  "match_start",
			RoomID: room.ID,
		})
		room.broadcastAll(startMsg)

		// Broadcast to EventDisplay
		p1Name, p2Name := "", ""
		room.mu.RLock()
		for _, p := range room.Players {
			if p.PlayerNum == 1 {
				p1Name = p.PlayerName
			} else if p.PlayerNum == 2 {
				p2Name = p.PlayerName
			}
		}
		room.mu.RUnlock()
		DefaultEventDisplayHub.Broadcast("room_playing", map[string]interface{}{
			"room_id":      room.ID,
			"player1_name": p1Name,
			"player2_name": p2Name,
		})
	}
}

func (m *Manager) handleGameOver(room *Room, client *Client, msg *GameMessage) {
	broadcast, _ := json.Marshal(msg)
	room.broadcastAll(broadcast)

	room.mu.Lock()
	// Dedup: each player_num can only send GAME_OVER once
	if room.overNum[client.PlayerNum] {
		room.mu.Unlock()
		return
	}
	room.overNum[client.PlayerNum] = true
	room.GameOvers++
	gameOvers := room.GameOvers
	room.mu.Unlock()

	if gameOvers >= 2 {
		p1Name, p2Name := "", ""
		room.mu.RLock()
		for _, p := range room.Players {
			if p.PlayerNum == 1 {
				p1Name = p.PlayerName
			} else if p.PlayerNum == 2 {
				p2Name = p.PlayerName
			}
		}
		room.mu.RUnlock()
		DefaultEventDisplayHub.Broadcast("room_finished", map[string]interface{}{
			"room_id":      room.ID,
			"player1_name": p1Name,
			"player2_name": p2Name,
		})
		m.destroyRoom(room.ID)
	}
}

func (m *Manager) BroadcastToRoom(roomID string, payload []byte) {
	m.mu.RLock()
	room, ok := m.rooms[roomID]
	m.mu.RUnlock()

	if !ok {
		return
	}
	room.broadcastAll(payload)
}

func (m *Manager) destroyRoom(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return
	}

	room.mu.Lock()
	room.destroyed = true
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
		Type:       "leave",
		PlayerID:   client.PlayerID,
		PlayerName: client.PlayerName,
		PlayerNum:  client.PlayerNum,
	})
	room.broadcastAll(leaveMsg)
	room.removeClient(client)

	m.mu.Lock()
	room.mu.RLock()
	empty := len(room.Players) == 0 && len(room.Spectators) == 0
	room.mu.RUnlock()
	if empty {
		delete(m.rooms, roomID)
	}
	m.mu.Unlock()
}

func relayToEventDisplay(msg *GameMessage, roomID string) {
	data := map[string]interface{}{
		"room_id":      roomID,
		"player_id":    msg.PlayerID,
		"player_name":  msg.PlayerName,
		"player_num":   msg.PlayerNum,
	}
	if msg.Type == "score_update" {
		data["level"] = msg.GameScore
		data["time_sec"] = msg.BlocksHit
		data["completed_levels"] = msg.GameScore
		data["is_finished"] = msg.GameScore >= 8
	} else if msg.Type == "level_start" {
		data["level"] = msg.Level
		data["completed_levels"] = msg.Level - 1
	}
	DefaultEventDisplayHub.Broadcast(msg.Type, data)
}