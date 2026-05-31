package controller

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"oamp-backend/internal/config"
	"oamp-backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
	err := config.DB.Where("status IN ?", []string{"waiting", "ready"}).
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
			return tx.Model(&room).Update("status", "playing").Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// LeaveRoomDB removes a player and their player_state; returns action taken
func LeaveRoomDB(code, playerName string) (string, error) {
	code = strings.ToUpper(code)
	var room model.Room
	err := config.DB.Transaction(func(tx *gorm.DB) error {
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
		return tx.Delete(&room).Error
	})
	if err != nil {
		return "", err
	}

	if room.Player1Name == playerName && room.Player2Name == "" {
		return "room_deleted", nil
	} else if room.Player1Name == playerName && room.Player2Name != "" {
		return "player_promoted", nil
	}
	return "player2_left", nil
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

// GameEvent — POST /api/v1/game/event
func GameEvent(c *gin.Context) {
	var event model.GameEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type, room_id, player_name required"})
		return
	}
	if event.Type == "" || event.RoomID == "" || event.PlayerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type, room_id, player_name required"})
		return
	}
	if err := UpsertPlayerStateDB(event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
