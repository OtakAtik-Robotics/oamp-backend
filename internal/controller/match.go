package controller

import (
	"net/http"
	"strings"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// GET /api/v1/rooms — list active rooms
func GetRooms(c *gin.Context) {
	rooms := GetActiveRooms()
	response.OKWithMessage(c, "Rooms fetched", rooms)
}

// POST /api/v1/rooms — create a new room
func CreateRoom(c *gin.Context) {
	var req struct {
		PlayerName string `json:"player_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "player_name required")
		return
	}

	room := CreateMatchRoom(req.PlayerName)
	if room == nil {
		response.Error(c, http.StatusServiceUnavailable, "Failed to create room")
		return
	}

	response.CreatedWithMessage(c, "Room created", room)
}

// GET /api/v1/rooms/:code — get room details
func GetRoom(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	room := GetRoomByCode(code)
	if room == nil {
		response.Error(c, http.StatusNotFound, "Room not found")
		return
	}
	response.OKWithMessage(c, "Room fetched", room)
}

// POST /api/v1/rooms/:code/join — join as player2
func JoinRoom(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	var req struct {
		PlayerName string `json:"player_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "player_name required")
		return
	}

	room, ok := JoinMatchRoom(code, req.PlayerName)
	if !ok {
		response.Error(c, http.StatusNotFound, "Room not found")
		return
	}
	if room == nil {
		response.Error(c, http.StatusConflict, "Room is full")
		return
	}
	response.OKWithMessage(c, "Joined room", room)
}

// POST /api/v1/rooms/:code/leave — leave a room
func LeaveRoom(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	var req struct {
		PlayerName string `json:"player_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "player_name required")
		return
	}

	ok := LeaveMatchRoom(code, req.PlayerName)
	if !ok {
		response.Error(c, http.StatusNotFound, "Room not found")
		return
	}
	response.OKWithMessage(c, "Left room", gin.H{"ok": true})
}

// POST /api/v1/rooms/:code/ready — mark player as ready
func SetReady(c *gin.Context) {
	code := strings.ToUpper(c.Param("code"))
	var req struct {
		PlayerName string `json:"player_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "player_name required")
		return
	}

	room, ok := SetPlayerReady(code, req.PlayerName)
	if !ok {
		response.Error(c, http.StatusNotFound, "Room not found")
		return
	}
	response.OKWithMessage(c, "Ready set", room)
}

// GET /api/v1/ranking — participants sorted by avg time
func GetRanking(c *gin.Context) {
	var participants []model.Participant
	if err := config.DB.Where("ai_analysis IS NOT NULL").
		Order("created_at ASC").
		Limit(100).
		Find(&participants).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch ranking")
		return
	}

	type RankEntry struct {
		Rank      int     `json:"rank"`
		UID       string  `json:"uid"`
		Name      string  `json:"name"`
		Age       int     `json:"age"`
		TaskAvg   float64 `json:"task_avg"`
		CognitiveAge int  `json:"cognitive_age"`
	}
	ranking := make([]RankEntry, len(participants))
	for i, p := range participants {
		ranking[i] = RankEntry{
			Rank:      i + 1,
			UID:       p.UID,
			Name:      p.Name,
			Age:       p.Age,
			TaskAvg:   0,
			CognitiveAge: 0,
		}
	}
	response.OKWithMessage(c, "Ranking fetched", ranking)
}

// GET /api/v1/stats — aggregate statistics
func GetStats(c *gin.Context) {
	var participants []model.Participant
	if err := config.DB.Find(&participants).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch stats")
		return
	}

	n := len(participants)
	if n == 0 {
		response.OKWithMessage(c, "Stats fetched", gin.H{
			"total_participants": 0,
			"avg_time":           0,
			"min_time":            0,
			"max_time":            0,
			"avg_cognitive_age":   0,
			"avg_visuo_spatial":   0,
		})
		return
	}

	var sumCognitiveAge float64
	for _, p := range participants {
		sumCognitiveAge += float64(p.Age)
	}
	response.OKWithMessage(c, "Stats fetched", gin.H{
		"total_participants": n,
		"avg_time":           0,
		"min_time":           0,
		"max_time":           0,
		"avg_cognitive_age":  sumCognitiveAge / float64(n),
		"avg_visuo_spatial":  0,
	})
}

// POST /api/v1/game/event — game event from desktop app (join_room, level_start, level_complete, leave_room)
func GameEvent(c *gin.Context) {
	var event struct {
		Type      string  `json:"type" binding:"required"`
		RoomID    string  `json:"room_id"`
		PlayerName string `json:"player_name"`
		Level     int     `json:"level"`
		TimeSec   float64 `json:"time_sec"`
	}
	if err := c.ShouldBindJSON(&event); err != nil {
		response.Error(c, http.StatusBadRequest, "type, room_id, player_name required")
		return
	}

	// Handle in-memory room state
	switch event.Type {
	case "join_room":
		// Room already exists via CreateRoom/JoinRoom; just ensure it's "playing"
		HandleJoinRoom(event.RoomID, event.PlayerName)
	case "leave_room":
		if event.RoomID != "" && event.PlayerName != "" {
			LeaveMatchRoom(event.RoomID, event.PlayerName)
		}
	}
	response.OKWithMessage(c, "Event processed", gin.H{"ok": true})
}