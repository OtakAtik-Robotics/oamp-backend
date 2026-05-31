package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCreateRoomDB(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, err := CreateRoomDB("Alice")
	if err != nil {
		t.Fatalf("CreateRoomDB failed: %v", err)
	}
	if room.ID == "" {
		t.Error("expected room code to be generated")
	}
	if room.Status != "waiting" {
		t.Errorf("expected status waiting, got %s", room.Status)
	}
	if room.Player1Name != "Alice" {
		t.Errorf("expected player1 Alice, got %s", room.Player1Name)
	}
	if len(room.ID) != 4 {
		t.Errorf("expected 4-char code, got %d chars", len(room.ID))
	}
}

func TestCreateRoomDB_CodeExcludesConfusable(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	for i := 0; i < 20; i++ {
		room, err := CreateRoomDB("Tester")
		if err != nil {
			t.Fatalf("CreateRoomDB failed: %v", err)
		}
		for _, ch := range room.ID {
			if ch == 'I' || ch == 'O' || ch == '0' || ch == '1' {
				t.Errorf("room code %s contains confusable char %c", room.ID, ch)
			}
		}
	}
}

func TestJoinRoomDB(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	updated, err := JoinRoomDB(room.ID, "Bob")
	if err != nil {
		t.Fatalf("JoinRoomDB failed: %v", err)
	}
	if updated.Player2Name != "Bob" {
		t.Errorf("expected player2 Bob, got %s", updated.Player2Name)
	}
	if updated.Status != "ready" {
		t.Errorf("expected status ready, got %s", updated.Status)
	}
}

func TestJoinRoomDB_RoomFull(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	JoinRoomDB(room.ID, "Bob")
	_, err := JoinRoomDB(room.ID, "Charlie")
	if err == nil || err.Error() != "room is full" {
		t.Errorf("expected 'room is full' error, got %v", err)
	}
}

func TestJoinRoomDB_AlreadyPlayer1(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	_, err := JoinRoomDB(room.ID, "Alice")
	if err == nil || err.Error() != "already player1" {
		t.Errorf("expected 'already player1' error, got %v", err)
	}
}

func TestLeaveRoomDB_Player2Leaves(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	JoinRoomDB(room.ID, "Bob")
	action, err := LeaveRoomDB(room.ID, "Bob")
	if err != nil {
		t.Fatalf("LeaveRoomDB failed: %v", err)
	}
	if action != "player2_left" {
		t.Errorf("expected player2_left, got %s", action)
	}
	var r model.Room
	config.DB.Where("id = ?", room.ID).First(&r)
	if r.Player2Name != "" {
		t.Errorf("expected player2 empty, got %s", r.Player2Name)
	}
	if r.Status != "waiting" {
		t.Errorf("expected status waiting, got %s", r.Status)
	}
}

func TestLeaveRoomDB_Player1Leaves_Alone(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	action, err := LeaveRoomDB(room.ID, "Alice")
	if err != nil {
		t.Fatalf("LeaveRoomDB failed: %v", err)
	}
	if action != "room_deleted" {
		t.Errorf("expected room_deleted, got %s", action)
	}
	var count int64
	config.DB.Model(&model.Room{}).Where("id = ?", room.ID).Count(&count)
	if count != 0 {
		t.Error("expected room to be deleted")
	}
}

func TestLeaveRoomDB_Player1Leaves_WithPlayer2(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	JoinRoomDB(room.ID, "Bob")
	action, err := LeaveRoomDB(room.ID, "Alice")
	if err != nil {
		t.Fatalf("LeaveRoomDB failed: %v", err)
	}
	if action != "player_promoted" {
		t.Errorf("expected player_promoted, got %s", action)
	}
	var r model.Room
	config.DB.Where("id = ?", room.ID).First(&r)
	if r.Player1Name != "Bob" {
		t.Errorf("expected player1 promoted to Bob, got %s", r.Player1Name)
	}
	if r.Player2Name != "" {
		t.Errorf("expected player2 empty, got %s", r.Player2Name)
	}
}

func TestSetReadyDB_BothReady_TransitionsToPlaying(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	JoinRoomDB(room.ID, "Bob")
	SetReadyDB(room.ID, "Alice")
	updated, err := SetReadyDB(room.ID, "Bob")
	if err != nil {
		t.Fatalf("SetReadyDB failed: %v", err)
	}
	if updated.Status != "playing" {
		t.Errorf("expected status playing, got %s", updated.Status)
	}
	if !updated.Player1Ready || !updated.Player2Ready {
		t.Error("expected both players ready")
	}
}

func TestSetReadyDB_PlayerNotInRoom(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	_, err := SetReadyDB(room.ID, "Bob")
	if err == nil || err.Error() != "player not found in room" {
		t.Errorf("expected 'player not found in room', got %v", err)
	}
}

func TestCleanupStaleRooms(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	// Create a stale waiting room
	config.DB.Create(&model.Room{
		ID:           "STAL",
		Status:       "waiting",
		Player1Name:  "Old",
		LastActivity: time.Now().Add(-10 * time.Minute),
	})
	// Create a fresh waiting room
	config.DB.Create(&model.Room{
		ID:           "FRES",
		Status:       "waiting",
		Player1Name:  "New",
		LastActivity: time.Now(),
	})

	cleanupStaleRooms()

	var count int64
	config.DB.Model(&model.Room{}).Where("id = ?", "STAL").Count(&count)
	if count != 0 {
		t.Error("expected stale room to be deleted")
	}
	config.DB.Model(&model.Room{}).Where("id = ?", "FRES").Count(&count)
	if count != 1 {
		t.Error("expected fresh room to remain")
	}
}

func TestCleanupStaleRooms_Playing(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	config.DB.Create(&model.Room{
		ID:           "PLAY",
		Status:       "playing",
		Player1Name:  "A",
		LastActivity: time.Now().Add(-40 * time.Minute),
	})

	cleanupStaleRooms()

	var r model.Room
	config.DB.Where("id = ?", "PLAY").First(&r)
	if r.Status != "finished" {
		t.Errorf("expected finished, got %s", r.Status)
	}
}

func TestUpsertPlayerStateDB_JoinRoom(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	err := UpsertPlayerStateDB(model.GameEvent{
		Type:       "join_room",
		RoomID:     "ROOM",
		PlayerName: "Alice",
	})
	if err != nil {
		t.Fatalf("UpsertPlayerStateDB failed: %v", err)
	}
	var ps model.PlayerState
	config.DB.Where("room_id = ? AND player_name = ?", "ROOM", "Alice").First(&ps)
	if ps.RoomID != "ROOM" {
		t.Errorf("expected room ROOM, got %s", ps.RoomID)
	}
}

func TestUpsertPlayerStateDB_LevelComplete(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	// Pre-create player state
	config.DB.Create(&model.PlayerState{
		RoomID:          "ROOM",
		PlayerName:      "Alice",
		CompletedLevels: 0,
		LevelTimes:      model.Float64Array{},
	})

	level := 1
	err := UpsertPlayerStateDB(model.GameEvent{
		Type:       "level_complete",
		RoomID:     "ROOM",
		PlayerName: "Alice",
		Level:      &level,
		TimeSec:    12.5,
	})
	if err != nil {
		t.Fatalf("UpsertPlayerStateDB failed: %v", err)
	}
	var ps model.PlayerState
	config.DB.Where("room_id = ? AND player_name = ?", "ROOM", "Alice").First(&ps)
	if ps.CompletedLevels != 1 {
		t.Errorf("expected 1 completed level, got %d", ps.CompletedLevels)
	}
	if len(ps.LevelTimes) != 1 || ps.LevelTimes[0] != 12.5 {
		t.Errorf("expected level times [12.5], got %v", ps.LevelTimes)
	}
}

// ---------- HTTP Handler Tests ----------

func TestCreateRoomHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/rooms", CreateRoom)

	body, _ := json.Marshal(map[string]string{"player_name": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/rooms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	var resp model.Room
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ID == "" {
		t.Error("expected room code in response")
	}
}

func TestCreateRoomHTTP_MissingPlayerName(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/rooms", CreateRoom)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/rooms", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestJoinRoomHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/rooms/:code/join", JoinRoom)

	body, _ := json.Marshal(map[string]string{"player_name": "Bob"})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID+"/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp model.Room
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Player2Name != "Bob" {
		t.Errorf("expected Bob as player2, got %s", resp.Player2Name)
	}
}

func TestJoinRoomHTTP_RoomFull(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")
	JoinRoomDB(room.ID, "Bob")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/rooms/:code/join", JoinRoom)

	body, _ := json.Marshal(map[string]string{"player_name": "Charlie"})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID+"/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestGetRoomHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/rooms/:code", GetRoom)

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+room.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp model.Room
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Player1Name != "Alice" {
		t.Errorf("expected Alice, got %s", resp.Player1Name)
	}
}

func TestLeaveRoomHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	room, _ := CreateRoomDB("Alice")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/rooms/:code/leave", LeaveRoom)

	body, _ := json.Marshal(map[string]string{"player_name": "Alice"})
	req := httptest.NewRequest(http.MethodPost, "/rooms/"+room.ID+"/leave", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGameEventHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/game/event", GameEvent)

	body, _ := json.Marshal(map[string]interface{}{
		"type":        "join_room",
		"room_id":     "ABCD",
		"player_name": "Alice",
	})
	req := httptest.NewRequest(http.MethodPost, "/game/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGameEventHTTP_MissingFields(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/game/event", GameEvent)

	body, _ := json.Marshal(map[string]string{"type": "join_room"})
	req := httptest.NewRequest(http.MethodPost, "/game/event", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
