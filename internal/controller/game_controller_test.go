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
	if session.DexterityScore != 0.0 {
		t.Errorf("expected dexterity_score 0.0, got %f", session.DexterityScore)
	}
	if session.CognitiveAge != 8 {
		t.Errorf("expected cognitive_age 8, got %d", session.CognitiveAge)
	}
	// totalTime = 5+6+7+8 = 26, levelReached=4
	// score = 4*1000 - 26*10 = 4000 - 260 = 3740
	expectedScore := 3740.0
	if session.Score != expectedScore {
		t.Errorf("expected score %f, got %f", expectedScore, session.Score)
	}
	if session.Mode != "training" {
		t.Errorf("expected mode training, got %s", session.Mode)
	}
}

func TestSaveGameSession_HighScore(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p := seedParticipant(t, "UID003", "Charlie", 12)
	result := model.GameResult{
		UID:          "UID003",
		Mode:         "competition",
		Age:          12,
		Task01:       8.0,
		Task02:       7.0,
		Task03:       6.0,
		Task04:       5.0,
		Task05:       4.0,
		Task06:       3.0,
		Task07:       2.0,
		Task08:       1.0,
		TaskAvg:      4.5,
		CognitiveAge: 10.0,
		VisuoSpatial: 100.0,
	}

	err := saveGameSession(&result, p.ID)
	if err != nil {
		t.Fatalf("saveGameSession failed: %v", err)
	}

	var session model.GameSession
	config.DB.Where("participant_id = ?", p.ID).First(&session)

	if session.LevelReached != 8 {
		t.Errorf("expected level_reached 8, got %d", session.LevelReached)
	}
	// totalTime = 8+7+6+5+4+3+2+1 = 36, levelReached=8
	// score = 8*1000 - 36*10 = 8000 - 360 = 7640
	expectedScore := 7640.0
	if session.Score != expectedScore {
		t.Errorf("expected score %f, got %f", expectedScore, session.Score)
	}
}

func TestSaveGameSession_DexterityRemoved(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p := seedParticipant(t, "UID002", "Bob", 5)
	result := model.GameResult{
		UID:          "UID002",
		Mode:         "competition",
		Age:          5,
		CognitiveAge: 9.0,
		VisuoSpatial: 100.0,
		Task01:       10.0,
		Task02:       0,
		TaskAvg:      10.0,
	}

	err := saveGameSession(&result, p.ID)
	if err != nil {
		t.Fatalf("saveGameSession failed: %v", err)
	}

	var session model.GameSession
	config.DB.Where("participant_id = ?", p.ID).First(&session)

	if session.DexterityScore != 0.0 {
		t.Errorf("expected dexterity_score 0.0, got %f", session.DexterityScore)
	}
	if session.CognitiveAge != 9 {
		t.Errorf("expected cognitive_age 9, got %d", session.CognitiveAge)
	}
	// levelReached=1, totalTime=10, score = 1*1000 - 10*10 = 900
	if session.Score != 900.0 {
		t.Errorf("expected score 900, got %f", session.Score)
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
		"visuo_spatial": 75.0,
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
		"visuo_spatial": 75.0,
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
