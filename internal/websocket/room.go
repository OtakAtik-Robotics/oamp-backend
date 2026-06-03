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
	mu          sync.RWMutex
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

	for _, spec := range r.Spectators {
		select {
		case spec.Send <- payload:
		default:
			log.Printf("[ws] spectator %s send buffer full, dropping", spec.PlayerID)
		}
	}
}

func (r *Room) broadcastToPlayers(payload []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.Players {
		select {
		case p.Send <- payload:
		default:
			log.Printf("[ws] player %s send buffer full, dropping", p.PlayerID)
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

func (r *Room) sendToPlayer(playerNum int, payload []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.Players {
		if p.PlayerNum == playerNum {
			select {
			case p.Send <- payload:
			default:
				log.Printf("[ws] player %s send buffer full, dropping", p.PlayerID)
			}
			return
		}
	}
}

func (r *Room) getPlayerByNum(playerNum int) *Client {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.Players {
		if p.PlayerNum == playerNum {
			return p
		}
	}
	return nil
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

	case "GAME_OVER":
		m.handleGameOver(room, client, &msg)

	default:
		broadcast, _ := json.Marshal(msg)
		room.broadcastToSpectators(broadcast)
	}
}

func (m *Manager) handlePlayerReady(room *Room, client *Client) {
	room.mu.Lock()
	room.ReadyCount++
	readyCount := room.ReadyCount
	room.mu.Unlock()

	if readyCount >= 2 {
		startMsg, _ := json.Marshal(GameMessage{
			Type:  "match_start",
			RoomID: room.ID,
		})
		room.broadcastAll(startMsg)
	}
}

func (m *Manager) handleGameOver(room *Room, client *Client, msg *GameMessage) {
	broadcast, _ := json.Marshal(msg)
	room.broadcastAll(broadcast)

	room.mu.Lock()
	room.GameOvers++
	gameOvers := room.GameOvers
	room.mu.Unlock()

	if gameOvers >= 2 {
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