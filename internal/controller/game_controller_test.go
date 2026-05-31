package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSaveGameSession_Computation(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p := seedParticipant(t, "UID001", "Alice", 10)
	result := model.GameResult{
		UID:          "UID001",
		Mode:         "training",
		Age:          10,
		Task01:       5.0,
		Task02:       6.0,
		Task03:       7.0,
		Task04:       8.0,
		Task05:       0, // incomplete
		Task06:       0,
		Task07:       0,
		Task08:       0,
		TaskAvg:      6.5,
		CognitiveAge: 8.0,
		VisuoSpatial: 75.0,
	}

	err := saveGameSession(&result, p.ID)
	if err != nil {
		t.Fatalf("saveGameSession failed: %v", err)
	}

	var session model.GameSession
	config.DB.Where("participant_id = ?", p.ID).First(&session)

	if session.LevelReached != 4 {
		t.Errorf("expected level_reached 4, got %d", session.LevelReached)
	}
	if session.VisuoSpatialFit != 0.75 {
		t.Errorf("expected visuo_spatial_fit 0.75, got %f", session.VisuoSpatialFit)
	}
	// dexterity = cognitive_age / real_age = 8/10 = 0.8
	if session.DexterityScore != 0.8 {
		t.Errorf("expected dexterity_score 0.8, got %f", session.DexterityScore)
	}
	// score = (4*10) + (0.75*50) + (0.8*0.2) = 40 + 37.5 + 0.16 = 77.66
	expectedScore := float64(4)*10 + 0.75*50 + 0.8*0.2
	if session.Score != expectedScore {
		t.Errorf("expected score %f, got %f", expectedScore, session.Score)
	}
	if session.Mode != "training" {
		t.Errorf("expected mode training, got %s", session.Mode)
	}
}

func TestSaveGameSession_DexterityCap(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p := seedParticipant(t, "UID002", "Bob", 5)
	result := model.GameResult{
		UID:          "UID002",
		Mode:         "competition",
		Age:          5,
		CognitiveAge: 15.0, // 15/5 = 3.0 → capped at 2.0
		VisuoSpatial: 100.0,
		TaskAvg:      10.0,
	}

	err := saveGameSession(&result, p.ID)
	if err != nil {
		t.Fatalf("saveGameSession failed: %v", err)
	}

	var session model.GameSession
	config.DB.Where("participant_id = ?", p.ID).First(&session)

	if session.DexterityScore != 2.0 {
		t.Errorf("expected dexterity_score capped at 2.0, got %f", session.DexterityScore)
	}
	if session.Mode != "competition" {
		t.Errorf("expected mode competition, got %s", session.Mode)
	}
}

func TestSubmitGameResultHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	seedParticipant(t, "UID001", "Alice", 10)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/game/submit", SubmitGameResult)

	body, _ := json.Marshal(map[string]interface{}{
		"uid":           "UID001",
		"mode":          "training",
		"age":           10,
		"task01":        5.0,
		"task_avg":      5.0,
		"cognitive_age": 8.0,
		"visuo_spatial": 60.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/game/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify game_results created
	var gr model.GameResult
	config.DB.Where("uid = ?", "UID001").First(&gr)
	if gr.UID != "UID001" {
		t.Error("expected game_result saved")
	}

	// Verify game_sessions created
	var gs model.GameSession
	config.DB.Where("mode = ?", "training").First(&gs)
	if gs.Mode != "training" {
		t.Error("expected game_session saved with training mode")
	}
}

func TestSubmitGameResultHTTP_DefaultMode(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	seedParticipant(t, "UID001", "Alice", 10)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/game/submit", SubmitGameResult)

	body, _ := json.Marshal(map[string]interface{}{
		"uid":           "UID001",
		"age":           10,
		"task01":        5.0,
		"task_avg":      5.0,
		"cognitive_age": 8.0,
		"visuo_spatial": 60.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/game/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var gr model.GameResult
	config.DB.Where("uid = ?", "UID001").First(&gr)
	if gr.Mode != "training" {
		t.Errorf("expected default mode training, got %s", gr.Mode)
	}
}

func TestSubmitGameResultHTTP_MissingUID(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/game/submit", SubmitGameResult)

	body, _ := json.Marshal(map[string]interface{}{"mode": "training"})
	req := httptest.NewRequest(http.MethodPost, "/game/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubmitGameResultHTTP_ParticipantNotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/game/submit", SubmitGameResult)

	body, _ := json.Marshal(map[string]interface{}{"uid": "UNKNOWN", "mode": "training"})
	req := httptest.NewRequest(http.MethodPost, "/game/submit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetParticipantByUID_HTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	seedParticipant(t, "UID001", "Alice", 10)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/participants/uid/:uid", GetParticipantByUID)

	req := httptest.NewRequest(http.MethodGet, "/participants/uid/UID001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetParticipantByUID_HTTP_NotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/participants/uid/:uid", GetParticipantByUID)

	req := httptest.NewRequest(http.MethodGet, "/participants/uid/UNKNOWN", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
