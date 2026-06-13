package controller

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/internal/websocket"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func broadcastRoomEvent(eventType, roomID, p1Name, p2Name string) {
	websocket.DefaultEventDisplayHub.Broadcast(eventType, map[string]interface{}{
		"room_id":      roomID,
		"player1_name": p1Name,
		"player2_name": p2Name,
	})
}

var wsManager *websocket.Manager

func SetWSManager(m *websocket.Manager) {
	wsManager = m
}

const roomCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateRoomCode() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	code := ""
	for _, b := range bytes {
		code += string(roomCodeChars[int(b)%len(roomCodeChars)])
	}
	return code
}

// cleanupStaleRooms deletes non-playing rooms idle >5min, playing >30min
func cleanupStaleRooms() {
	now := time.Now()
	var rooms []model.Room

	if err := config.DB.Where("status IN ?", []string{"waiting", "ready"}).Find(&rooms).Error; err != nil {
		return
	}
	for _, room := range rooms {
		if now.Sub(room.LastActivity) > 5*time.Minute {
			config.DB.Delete(&room)
			log.Printf("[room] stale room %s deleted", room.ID)
		}
	}

	if err := config.DB.Where("status = ?", "playing").Find(&rooms).Error; err != nil {
		return
	}
	for _, room := range rooms {
		if now.Sub(room.LastActivity) > 30*time.Minute {
			config.DB.Model(&room).Update("status", "finished")
			log.Printf("[room] room %s marked finished (stale playing)", room.ID)
			broadcastRoomEvent("room_finished", room.ID, room.Player1Name, room.Player2Name)
		}
	}
}

// CreateRoomDB creates a new room with the given player as player1
func CreateRoomDB(playerName string) (*model.Room, error) {
	var room *model.Room
	for attempts := 0; attempts < 5; attempts++ {
		code := generateRoomCode()
		if code == "" {
			continue
		}
		room = &model.Room{
			ID:           code,
			Status:       "waiting",
			Player1Name:  playerName,
			LastActivity: time.Now(),
		}
		if err := config.DB.Create(room).Error; err == nil {
			return room, nil
		}
	}
	return nil, fmt.Errorf("failed to create room after 5 attempts")
}

// GetActiveRoomsDB returns waiting/ready rooms after cleanup
func GetActiveRoomsDB() ([]model.Room, error) {
	cleanupStaleRooms()
	var rooms []model.Room
	err := config.DB.Where("status IN ?", []string{"waiting", "ready", "playing", "finished"}).
		Order("created_at DESC").
		Find(&rooms).Error
	return rooms, err
}

// GetRoomByCodeDB fetches a room by its 4-char code
func GetRoomByCodeDB(code string) (*model.Room, error) {
	var room model.Room
	code = strings.ToUpper(code)
	err := config.DB.Where("id = ?", code).First(&room).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// JoinRoomDB adds player2 to a room
func JoinRoomDB(code, playerName string) (*model.Room, error) {
	code = strings.ToUpper(code)
	var room model.Room
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", code).First(&room).Error; err != nil {
			return err
		}
		if room.Player2Name != "" {
			return fmt.Errorf("room is full")
		}
		if room.Player1Name == playerName {
			return fmt.Errorf("already player1")
		}
		updates := map[string]interface{}{
			"player2_name":   playerName,
			"status":        "ready",
			"last_activity": time.Now(),
		}
		return tx.Model(&room).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	// Reload to get updated fields
	config.DB.Where("id = ?", code).First(&room)
	return &room, nil
}

// SetReadyDB marks a player as ready; auto-transitions to playing when both ready
func SetReadyDB(code, playerName string) (*model.Room, error) {
	code = strings.ToUpper(code)
	var room model.Room
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", code).First(&room).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{"last_activity": time.Now()}
		if room.Player1Name == playerName {
			updates["player1_ready"] = true
		} else if room.Player2Name == playerName {
			updates["player2_ready"] = true
		} else {
			return fmt.Errorf("player not found in room")
		}
		if err := tx.Model(&room).Updates(updates).Error; err != nil {
			return err
		}
		// Reload to check both ready
		if err := tx.Where("id = ?", code).First(&room).Error; err != nil {
			return err
		}
		if room.Player1Ready && room.Player2Ready {
			if err := tx.Model(&room).Update("status", "playing").Error; err != nil {
				return err
			}
			room.Status = "playing" // update local struct for WS broadcast below
			broadcastRoomEvent("room_playing", room.ID, room.Player1Name, room.Player2Name)
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Broadcast match_start via WS when both players are ready
	if wsManager != nil && room.Status == "playing" {
		msg, _ := json.Marshal(websocket.GameMessage{
			Type:        "match_start",
			RoomID:      room.ID,
			Player1Name: room.Player1Name,
			Player2Name: room.Player2Name,
		})
		wsManager.BroadcastToRoom(room.ID, msg)
	}
	return &room, nil
}

// LeaveRoomDB removes a player and their player_state; returns action taken
func LeaveRoomDB(code, playerName string) (string, error) {
	code = strings.ToUpper(code)
	var action string
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var room model.Room
		if err := tx.Where("id = ?", code).First(&room).Error; err != nil {
			return err
		}
		isPlayer1 := room.Player1Name == playerName
		isPlayer2 := room.Player2Name == playerName

		if !isPlayer1 && !isPlayer2 {
			return fmt.Errorf("player not found in room")
		}

		// Always clean up this player's state first
		if err := tx.Where("room_id = ? AND player_name = ?", code, playerName).
			Delete(&model.PlayerState{}).Error; err != nil {
			return err
		}

		if isPlayer1 && room.Player2Name != "" {
			// Promote player2 to player1
			action = "player_promoted"
			updates := map[string]interface{}{
				"player1_name":   room.Player2Name,
				"player2_name":   "",
				"player1_ready":  false,
				"player2_ready":  false,
				"status":         "waiting",
				"last_activity":  time.Now(),
			}
			return tx.Model(&room).Updates(updates).Error
		} else if isPlayer2 {
			// Player2 leaves
			action = "player2_left"
			updates := map[string]interface{}{
				"player2_name":   "",
				"player1_ready":  false,
				"player2_ready":  false,
				"status":         "waiting",
				"last_activity":  time.Now(),
			}
			return tx.Model(&room).Updates(updates).Error
		}
		// Only player in room — delete room entirely
		action = "room_deleted"
		return tx.Delete(&room).Error
	})
	return action, err
}

// UpsertPlayerStateDB processes game events from desktop client
func UpsertPlayerStateDB(event model.GameEvent) error {
	switch event.Type {
	case "join_room":
		// Upsert room only if not exists (ignoreDuplicates: true behavior)
		var existing model.Room
		if err := config.DB.Where("id = ?", event.RoomID).First(&existing).Error; err == gorm.ErrRecordNotFound {
			room := model.Room{
				ID:           event.RoomID,
				Status:       "playing",
				LastActivity: time.Now(),
			}
			config.DB.Create(&room)
		}

		ps := model.PlayerState{
			RoomID:          event.RoomID,
			PlayerName:      event.PlayerName,
			CurrentLevel:    0,
			ElapsedTime:     0,
			CompletedLevels: 0,
			LevelTimes:      []float64{},
			IsFinished:      false,
		}
		return config.DB.Where("room_id = ? AND player_name = ?", event.RoomID, event.PlayerName).
			Assign(ps).FirstOrCreate(&ps).Error

	case "level_start":
		config.DB.Model(&model.Room{}).Where("id = ?", event.RoomID).
			Update("last_activity", time.Now())
		return config.DB.Model(&model.PlayerState{}).
			Where("room_id = ? AND player_name = ?", event.RoomID, event.PlayerName).
			Updates(map[string]interface{}{
				"current_level": event.Level,
				"elapsed_time":   0,
				"updated_at":     time.Now(),
			}).Error

	case "level_complete":
		return config.DB.Transaction(func(tx *gorm.DB) error {
			// Update room last_activity
			tx.Model(&model.Room{}).Where("id = ?", event.RoomID).
				Update("last_activity", time.Now())

			var ps model.PlayerState
			if err := tx.Where("room_id = ? AND player_name = ?", event.RoomID, event.PlayerName).First(&ps).Error; err != nil {
				return err
			}

			newTimes := append(ps.LevelTimes, event.TimeSec)
			completedLevels := ps.CompletedLevels + 1

			updates := map[string]interface{}{
				"level_times":       newTimes,
				"completed_levels":   completedLevels,
				"is_finished":       completedLevels >= 8,
				"current_level":     event.Level,
				"elapsed_time":      event.TimeSec, // single level time, not accumulated
				"updated_at":        time.Now(),
			}
			return tx.Model(&ps).Updates(updates).Error
		})

	case "leave_room":
		return config.DB.Transaction(func(tx *gorm.DB) error {
			var room model.Room
			if err := tx.Where("id = ?", event.RoomID).First(&room).Error; err != nil {
				return err
			}
			isPlayer1 := room.Player1Name == event.PlayerName
			isPlayer2 := room.Player2Name == event.PlayerName

			if !isPlayer1 && !isPlayer2 {
				return nil // player not in room, nothing to do
			}

			// Delete player_state
			if err := tx.Where("room_id = ? AND player_name = ?", event.RoomID, event.PlayerName).
				Delete(&model.PlayerState{}).Error; err != nil {
				return err
			}

			if isPlayer1 && room.Player2Name != "" {
				// Promote player2 to player1
				return tx.Model(&room).Updates(map[string]interface{}{
					"player1_name":   room.Player2Name,
					"player2_name":   "",
					"player1_ready":  false,
					"player2_ready":  false,
					"status":         "waiting",
					"last_activity":  time.Now(),
				}).Error
			} else if isPlayer2 {
				// Player2 leaves
				return tx.Model(&room).Updates(map[string]interface{}{
					"player2_name":   "",
					"player1_ready":  false,
					"player2_ready":  false,
					"status":         "waiting",
					"last_activity":  time.Now(),
				}).Error
			}
			// Only player — delete room
			return tx.Delete(&room).Error
		})

	case "heartbeat":
		var ps model.PlayerState
		err := config.DB.Where("room_id = ? AND player_name = ?", event.RoomID, event.PlayerName).First(&ps).Error
		if err == nil {
			return config.DB.Model(&ps).Update("updated_at", time.Now()).Error
		}
		ps = model.PlayerState{
			RoomID:     event.RoomID,
			PlayerName: event.PlayerName,
		}
		return config.DB.Create(&ps).Error
	}
	return nil
}

// HTTP Handlers — raw JSON response (matches web-server-api)

// GetRooms — GET /api/v1/rooms
func GetRooms(c *gin.Context) {
	rooms, err := GetActiveRoomsDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

// GetRoomsV1 — GET /api/v1/rooms (wrapped response per v1 convention)
func GetRoomsV1(c *gin.Context) {
	rooms, err := GetActiveRoomsDB()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch rooms")
		return
	}
	response.OKWithMessage(c, "Rooms fetched", rooms)
}

// CreateRoom — POST /api/v1/rooms
func CreateRoom(c *gin.Context) {
	var req struct {
		PlayerName string `json:"player_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "player_name is required"})
		return
	}
	room, err := CreateRoomDB(req.PlayerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate a unique room code. Please try again."})
		return
	}
	c.JSON(http.StatusCreated, room)
}

// GetRoom — GET /api/v1/rooms/:code
func GetRoom(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	room, err := GetRoomByCodeDB(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}
	c.JSON(http.StatusOK, room)
}

// JoinRoom — POST /api/v1/rooms/:code/join
func JoinRoom(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	var req struct {
		PlayerName string `json:"player_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "player_name is required"})
		return
	}

	room, err := GetRoomByCodeDB(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	if room.Player2Name != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "Room is full"})
		return
	}
	if room.Player1Name == req.PlayerName {
		c.JSON(http.StatusConflict, gin.H{"error": "You are already in this room as player 1"})
		return
	}

	room, err = JoinRoomDB(code, req.PlayerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	// Broadcast room_update via WS so lobby updates in real time
	if wsManager != nil {
		msg, _ := json.Marshal(websocket.GameMessage{
			Type:        "room_update",
			RoomID:      room.ID,
			Status:      room.Status,
			Player1Name: room.Player1Name,
			Player2Name: room.Player2Name,
		})
		wsManager.BroadcastToRoom(room.ID, msg)
	}
	c.JSON(http.StatusOK, room)
}

// SetReady — POST /api/v1/rooms/:code/ready
func SetReady(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	var req struct {
		PlayerName string `json:"player_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "player_name is required"})
		return
	}
	room, err := SetReadyDB(code, req.PlayerName)
	if err != nil {
		if err.Error() == "player not found in room" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found in this room"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, room)
}

// LeaveRoom — POST /api/v1/rooms/:code/leave
func LeaveRoom(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	var req struct {
		PlayerName string `json:"player_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "player_name is required"})
		return
	}

	// Check room exists
	room, err := GetRoomByCodeDB(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	isPlayer1 := room.Player1Name == req.PlayerName
	isPlayer2 := room.Player2Name == req.PlayerName
	if !isPlayer1 && !isPlayer2 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Player not found in this room"})
		return
	}

	_, err = LeaveRoomDB(code, req.PlayerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SubmitDuelResultDB — player submits score after game ends; determines winner when both submit
// Score is computed server-side from game_sessions when possible; falls back to client-provided value
func SubmitDuelResultDB(code, playerUID string, playerNum int, score float64) (*model.Room, string, error) {
	code = strings.ToUpper(code)
	var room model.Room
	var matchStatus string

	// Validate score: must be finite, non-negative
	if math.IsNaN(score) || math.IsInf(score, 0) || score < 0 {
		return nil, "", fmt.Errorf("invalid score: must be finite, non-negative")
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", code).First(&room).Error; err != nil {
			return err
		}
		if room.Status != "playing" {
			return fmt.Errorf("room is not in playing state")
		}

		// Verify player_uid matches the player in the room
		switch playerNum {
		case 1:
			if room.Player1UID != "" && room.Player1UID != playerUID {
				return fmt.Errorf("player1 UID mismatch")
			}
		case 2:
			if room.Player2UID != "" && room.Player2UID != playerUID {
				return fmt.Errorf("player2 UID mismatch")
			}
		}

		// Verify the player_uid belongs to a registered participant
		var participant model.Participant
		if err := tx.Where("uid = ?", playerUID).First(&participant).Error; err != nil {
			return fmt.Errorf("participant with uid '%s' not found", playerUID)
		}

		// Verify the participant's name matches the room's player name
		switch playerNum {
		case 1:
			if room.Player1Name != "" && room.Player1Name != participant.Name {
				return fmt.Errorf("player1 name mismatch: expected '%s', got '%s'", room.Player1Name, participant.Name)
			}
		case 2:
			if room.Player2Name != "" && room.Player2Name != participant.Name {
				return fmt.Errorf("player2 name mismatch: expected '%s', got '%s'", room.Player2Name, participant.Name)
			}
		}

		// Anti re-submit: reject if already submitted
		if playerNum == 1 && room.Player1Submitted {
			return fmt.Errorf("player 1 already submitted")
		}
		if playerNum == 2 && room.Player2Submitted {
			return fmt.Errorf("player 2 already submitted")
		}

		// Compute real score from game_sessions (server-trusted) — filter by competition/tournament mode only
		realScore := score
		var session model.GameSession
		if err := tx.Where("participant_id = ? AND mode IN ?", participant.ID, []string{"competition", "tournament"}).
			Order("created_at DESC").First(&session).Error; err == nil {
			// Use total_time from the latest game session in competition/tournament mode (lower = faster)
			realScore = session.TotalTime
		}

		updates := map[string]interface{}{"last_activity": time.Now()}
		switch playerNum {
		case 1:
			updates["player1_uid"] = playerUID
			updates["player1_score"] = realScore
			updates["player1_submitted"] = true
		case 2:
			updates["player2_uid"] = playerUID
			updates["player2_score"] = realScore
			updates["player2_submitted"] = true
		default:
			return fmt.Errorf("player_num must be 1 or 2")
		}

		if err := tx.Model(&room).Updates(updates).Error; err != nil {
			return err
		}

		if err := tx.Where("id = ?", code).First(&room).Error; err != nil {
			return err
		}

		if !room.Player1Submitted || !room.Player2Submitted {
			matchStatus = "waiting"
			return nil
		}

		var winner string
		if room.Player1Score < room.Player2Score {
			winner = "1"
		} else if room.Player2Score < room.Player1Score {
			winner = "2"
		} else {
			winner = "draw"
		}

		if err := tx.Model(&room).Updates(map[string]interface{}{
			"winner":        winner,
			"status":        "finished",
			"last_activity": time.Now(),
		}).Error; err != nil {
			return err
		}

		if err := tx.Where("id = ?", code).First(&room).Error; err != nil {
			return err
		}

		matchStatus = "decided"
		broadcastRoomEvent("room_finished", room.ID, room.Player1Name, room.Player2Name)
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return &room, matchStatus, nil
}

// GetDuelResultDB — fetch room with winner info for polling
func GetDuelResultDB(code string) (*model.Room, error) {
	code = strings.ToUpper(code)
	var room model.Room
	if err := config.DB.Where("id = ?", code).First(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

// SubmitDuelResult — POST /api/v1/rooms/:code/result
func SubmitDuelResult(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	var req struct {
		PlayerUID string  `json:"player_uid" binding:"required"`
		PlayerNum int     `json:"player_num" binding:"required,oneof=1 2"`
		Score     float64 `json:"score" binding:"gte=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": response.FormatBindError(err)})
		return
	}

	// Server-recomputed score from game_sessions (pure speed: lower = faster)
	room, matchStatus, err := SubmitDuelResultDB(code, req.PlayerUID, req.PlayerNum, req.Score)
	if err != nil {
		if err.Error() == "room is not in playing state" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	result := gin.H{
		"room_id":       room.ID,
		"match_status":  matchStatus,
		"player1_score": room.Player1Score,
		"player2_score": room.Player2Score,
	}
	if matchStatus == "decided" {
		result["winner"] = room.Winner
		// Broadcast match_result via WS so players get instant notification
		if wsManager != nil {
			msg, _ := json.Marshal(websocket.GameMessage{
				Type:    "match_result",
				Winner:  room.Winner,
				P1Score: room.Player1Score,
				P2Score: room.Player2Score,
			})
			wsManager.BroadcastToRoom(room.ID, msg)
		}
	}
	c.JSON(http.StatusOK, result)
}

// GetDuelResult — GET /api/v1/rooms/:code/result
func GetDuelResult(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	room, err := GetDuelResultDB(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	result := gin.H{
		"room_id":       room.ID,
		"status":        room.Status,
		"player1_name":  room.Player1Name,
		"player2_name":  room.Player2Name,
		"player1_score": room.Player1Score,
		"player2_score": room.Player2Score,
	}
	if room.Winner != "" {
		result["winner"] = room.Winner
	}
	c.JSON(http.StatusOK, result)
}

// GameEvent — POST /api/v1/game/event
func GameEvent(c *gin.Context) {
	var event model.GameEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type, room_id, player_name required"})
		return
	}
	if event.Type == "" || event.PlayerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type and player_name required"})
		return
	}
	if event.Type != "heartbeat" && event.RoomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room_id required for non-heartbeat events"})
		return
	}
	if event.Type == "heartbeat" && event.RoomID == "" {
		event.RoomID = "_training"
	}
	if err := UpsertPlayerStateDB(event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	// Note: level_start and score_update are broadcast directly by game client via WS.
	// Server does not echo them — avoids duplicate messages.

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
