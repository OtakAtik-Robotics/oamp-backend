package controller

import (
	"log"
	"sync"
	"time"

	"oamp-backend/internal/model"
)

type matchRoom struct {
	Code         string
	Status       string // waiting, ready, playing
	Player1Name  string
	Player2Name  string
	Player1Ready bool
	Player2Ready bool
	LastActivity time.Time
	mu           sync.RWMutex
}

type roomManager struct {
	rooms map[string]*matchRoom
	mu    sync.RWMutex
}

var rm *roomManager

func initRoomManager() *roomManager {
	if rm == nil {
		rm = &roomManager{rooms: make(map[string]*matchRoom)}
		go rm.cleanupLoop()
	}
	return rm
}

func (m *roomManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for code, room := range m.rooms {
			room.mu.RLock()
			inactive := now.Sub(room.LastActivity) > 5*time.Minute
			playing := room.Status == "playing"
			room.mu.RUnlock()
			if inactive && !playing {
				m.mu.Unlock()
				m.deleteRoom(code)
				m.mu.Lock()
			}
		}
		m.mu.Unlock()
	}
}

func (m *roomManager) deleteRoom(code string) {
	m.mu.Lock()
	delete(m.rooms, code)
	m.mu.Unlock()
	log.Printf("[match] room %s deleted (stale)", code)
}

func generateRoomCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	seed := time.Now().UnixNano()
	code := ""
	for j := 0; j < 4; j++ {
		seed = seed*1103515245 + 12345
		code += string(chars[int(seed)%len(chars)])
	}
	return code
}

func CreateMatchRoom(player1Name string) *model.MatchRoom {
	manager := initRoomManager()

	code := generateRoomCode()
	if code == "" {
		return nil
	}

	room := &matchRoom{
		Code:         code,
		Status:       "waiting",
		Player1Name:  player1Name,
		LastActivity: time.Now(),
	}

	manager.mu.Lock()
	manager.rooms[code] = room
	manager.mu.Unlock()

	return &model.MatchRoom{
		ID:           code,
		Status:       room.Status,
		Player1Name:  room.Player1Name,
		Player2Name:  room.Player2Name,
		Player1Ready: room.Player1Ready,
		Player2Ready: room.Player2Ready,
		LastActivity: room.LastActivity,
	}
}

func GetActiveRooms() []model.MatchRoom {
	manager := initRoomManager()
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	var result []model.MatchRoom
	for _, room := range manager.rooms {
		room.mu.RLock()
		if room.Status == "waiting" || room.Status == "ready" {
			result = append(result, model.MatchRoom{
				ID:           room.Code,
				Status:       room.Status,
				Player1Name:  room.Player1Name,
				Player2Name: room.Player2Name,
				LastActivity: room.LastActivity,
			})
		}
		room.mu.RUnlock()
	}
	return result
}

func GetRoomByCode(code string) *model.MatchRoom {
	manager := initRoomManager()
	manager.mu.RLock()
	room, ok := manager.rooms[code]
	manager.mu.RUnlock()
	if !ok {
		return nil
	}

	room.mu.RLock()
	defer room.mu.RUnlock()
	return &model.MatchRoom{
		ID:           room.Code,
		Status:       room.Status,
		Player1Name:  room.Player1Name,
		Player2Name:  room.Player2Name,
		Player1Ready: room.Player1Ready,
		Player2Ready: room.Player2Ready,
		LastActivity: room.LastActivity,
	}
}

func JoinMatchRoom(code, playerName string) (*model.MatchRoom, bool) {
	manager := initRoomManager()
	manager.mu.RLock()
	room, ok := manager.rooms[code]
	manager.mu.RUnlock()
	if !ok {
		return nil, false
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Player2Name != "" {
		return nil, false // full
	}
	if room.Player1Name == playerName {
		return nil, false // already player1
	}

	room.Player2Name = playerName
	room.Status = "ready"
	room.LastActivity = time.Now()

	return &model.MatchRoom{
		ID:           room.Code,
		Status:       room.Status,
		Player1Name:  room.Player1Name,
		Player2Name:  room.Player2Name,
		Player1Ready: room.Player1Ready,
		Player2Ready: room.Player2Ready,
		LastActivity: room.LastActivity,
	}, true
}

func LeaveMatchRoom(code, playerName string) bool {
	manager := initRoomManager()
	manager.mu.RLock()
	room, ok := manager.rooms[code]
	manager.mu.RUnlock()
	if !ok {
		return false
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Player1Name == playerName && room.Player2Name == "" {
		// Only player1 in room — delete
		manager.mu.Lock()
		delete(manager.rooms, code)
		manager.mu.Unlock()
	} else if room.Player1Name == playerName && room.Player2Name != "" {
		// Promote player2 to player1
		room.Player1Name = room.Player2Name
		room.Player2Name = ""
		room.Player1Ready = false
		room.Player2Ready = false
		room.Status = "waiting"
		room.LastActivity = time.Now()
	} else if room.Player2Name == playerName {
		// Player2 leaves
		room.Player2Name = ""
		room.Player1Ready = false
		room.Player2Ready = false
		room.Status = "waiting"
		room.LastActivity = time.Now()
	}

	return true
}

func SetPlayerReady(code, playerName string) (*model.MatchRoom, bool) {
	manager := initRoomManager()
	manager.mu.RLock()
	room, ok := manager.rooms[code]
	manager.mu.RUnlock()
	if !ok {
		return nil, false
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Player1Name == playerName {
		room.Player1Ready = true
	} else if room.Player2Name == playerName {
		room.Player2Ready = true
	} else {
		return nil, false
	}

	if room.Player1Ready && room.Player2Ready {
		room.Status = "playing"
	}

	room.LastActivity = time.Now()

	return &model.MatchRoom{
		ID:           room.Code,
		Status:       room.Status,
		Player1Name:  room.Player1Name,
		Player2Name:  room.Player2Name,
		Player1Ready: room.Player1Ready,
		Player2Ready: room.Player2Ready,
		LastActivity: room.LastActivity,
	}, true
}

func HandleJoinRoom(roomID, playerName string) {
	manager := initRoomManager()
	manager.mu.RLock()
	room, ok := manager.rooms[roomID]
	manager.mu.RUnlock()
	if !ok {
		return
	}
	room.mu.Lock()
	room.Status = "playing"
	room.LastActivity = time.Now()
	room.mu.Unlock()
}