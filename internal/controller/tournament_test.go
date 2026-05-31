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

func seedParticipant(t *testing.T, uid, name string, age int) *model.Participant {
	p := &model.Participant{UID: uid, Name: name, Age: age, Grade: "A", Gender: "male", Height: 160, Weight: 50}
	if err := config.DB.Create(p).Error; err != nil {
		t.Fatalf("seed participant failed: %v", err)
	}
	return p
}

func TestNextPowerOf2(t *testing.T) {
	cases := []struct{ in, want int }{
		{1, 1}, {2, 2}, {3, 4}, {4, 4}, {5, 8}, {8, 8}, {9, 16}, {17, 32},
	}
	for _, c := range cases {
		got := nextPowerOf2(c.in)
		if got != c.want {
			t.Errorf("nextPowerOf2(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestCreateTournamentHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments", CreateTournament)

	body, _ := json.Marshal(CreateTournamentPayload{Name: "Cup A", MaxPlayers: 8})
	req := httptest.NewRequest(http.MethodPost, "/tournaments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateTournamentHTTP_InvalidMaxPlayers(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments", CreateTournament)

	body, _ := json.Marshal(CreateTournamentPayload{Name: "Cup", MaxPlayers: 1})
	req := httptest.NewRequest(http.MethodPost, "/tournaments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestJoinTournamentHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	seedParticipant(t, "UID001", "Alice", 10)
	config.DB.Create(&model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "registration"})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments/:id/join", JoinTournament)

	body, _ := json.Marshal(JoinTournamentPayload{UID: "UID001"})
	req := httptest.NewRequest(http.MethodPost, "/tournaments/1/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJoinTournamentHTTP_AlreadyJoined(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p := seedParticipant(t, "UID001", "Alice", 10)
	tour := model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "registration"}
	config.DB.Create(&tour)
	config.DB.Create(&model.TournamentPlayer{TournamentID: tour.ID, ParticipantID: p.ID, Name: p.Name})
	config.DB.Model(&tour).Update("player_count", 1)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments/:id/join", JoinTournament)

	body, _ := json.Marshal(JoinTournamentPayload{UID: "UID001"})
	req := httptest.NewRequest(http.MethodPost, "/tournaments/1/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestStartTournamentHTTP_2Players(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p1 := seedParticipant(t, "UID001", "Alice", 10)
	p2 := seedParticipant(t, "UID002", "Bob", 11)
	tour := model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "registration", PlayerCount: 2}
	config.DB.Create(&tour)
	config.DB.Create(&model.TournamentPlayer{TournamentID: tour.ID, ParticipantID: p1.ID, Name: p1.Name})
	config.DB.Create(&model.TournamentPlayer{TournamentID: tour.ID, ParticipantID: p2.ID, Name: p2.Name})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments/:id/start", StartTournament)

	req := httptest.NewRequest(http.MethodPost, "/tournaments/1/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var matches []model.TournamentMatch
	config.DB.Where("tournament_id = ?", tour.ID).Find(&matches)
	if len(matches) != 1 {
		t.Errorf("expected 1 match for 2 players, got %d", len(matches))
	}
	if matches[0].Status != "ready" {
		t.Errorf("expected first match status ready, got %s", matches[0].Status)
	}
	if matches[0].RoomID == "" {
		t.Error("expected room code assigned to first match")
	}
}

func TestStartTournamentHTTP_3Players_WithBye(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p1 := seedParticipant(t, "UID001", "Alice", 10)
	p2 := seedParticipant(t, "UID002", "Bob", 11)
	p3 := seedParticipant(t, "UID003", "Charlie", 12)
	tour := model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "registration", PlayerCount: 3}
	config.DB.Create(&tour)
	config.DB.Create(&model.TournamentPlayer{TournamentID: tour.ID, ParticipantID: p1.ID, Name: p1.Name})
	config.DB.Create(&model.TournamentPlayer{TournamentID: tour.ID, ParticipantID: p2.ID, Name: p2.Name})
	config.DB.Create(&model.TournamentPlayer{TournamentID: tour.ID, ParticipantID: p3.ID, Name: p3.Name})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments/:id/start", StartTournament)

	req := httptest.NewRequest(http.MethodPost, "/tournaments/1/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var matches []model.TournamentMatch
	config.DB.Where("tournament_id = ?", tour.ID).Order("round asc, match_number asc").Find(&matches)
	if len(matches) != 3 {
		t.Errorf("expected 3 matches for 3 players (1 bye + 1 scheduled + 1 final), got %d", len(matches))
	}

	byeCount := 0
	for _, m := range matches {
		if m.Status == "bye" {
			byeCount++
		}
	}
	if byeCount != 1 {
		t.Errorf("expected exactly 1 bye match, got %d", byeCount)
	}
}

func TestSubmitMatchResultHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p1 := seedParticipant(t, "UID001", "Alice", 10)
	p2 := seedParticipant(t, "UID002", "Bob", 11)
	tour := model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "in_progress", CurrentRound: 1}
	config.DB.Create(&tour)
	m := model.TournamentMatch{
		TournamentID: tour.ID,
		Round:        1,
		MatchNumber:  1,
		Player1ID:    &p1.ID,
		Player1Name:  p1.Name,
		Player2ID:    &p2.ID,
		Player2Name:  p2.Name,
		Status:       "ready",
	}
	config.DB.Create(&m)
	config.DB.Model(&tour).Update("current_match_id", m.ID)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments/:id/matches/:mid/result", SubmitMatchResult)

	body, _ := json.Marshal(SubmitResultPayload{Player1Score: 10, Player2Score: 5, WinnerID: p1.ID})
	req := httptest.NewRequest(http.MethodPost, "/tournaments/1/matches/1/result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated model.TournamentMatch
	config.DB.First(&updated, m.ID)
	if updated.Status != "finished" {
		t.Errorf("expected finished, got %s", updated.Status)
	}
	if updated.WinnerID == nil || *updated.WinnerID != p1.ID {
		t.Error("expected winner Alice")
	}
}

func TestSubmitMatchResultHTTP_AlreadyFinished(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p1 := seedParticipant(t, "UID001", "Alice", 10)
	seedParticipant(t, "UID002", "Bob", 11)
	tour := model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "in_progress"}
	config.DB.Create(&tour)
	m := model.TournamentMatch{
		TournamentID: tour.ID,
		Round:        1,
		MatchNumber:  1,
		Status:       "finished",
		WinnerID:     &p1.ID,
	}
	config.DB.Create(&m)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments/:id/matches/:mid/result", SubmitMatchResult)

	body, _ := json.Marshal(SubmitResultPayload{WinnerID: p1.ID})
	req := httptest.NewRequest(http.MethodPost, "/tournaments/1/matches/1/result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestRegisterTournamentPlayersHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	seedParticipant(t, "UID001", "Alice", 10)
	seedParticipant(t, "UID002", "Bob", 11)
	config.DB.Create(&model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "registration"})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/tournaments/:id/register", RegisterTournamentPlayers)

	body, _ := json.Marshal(RegisterPlayersPayload{UIDs: []string{"UID001", "UID002"}})
	req := httptest.NewRequest(http.MethodPost, "/tournaments/1/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result struct {
		Data struct {
			Added  int      `json:"added"`
			Errors []string `json:"errors"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Data.Added != 2 {
		t.Errorf("expected 2 added, got %d", result.Data.Added)
	}
	if len(result.Data.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Data.Errors)
	}
}

func TestDeleteTournamentHTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	config.DB.Create(&model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "registration"})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/tournaments/:id", DeleteTournament)

	req := httptest.NewRequest(http.MethodDelete, "/tournaments/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var count int64
	config.DB.Model(&model.Tournament{}).Count(&count)
	if count != 0 {
		t.Error("expected tournament deleted")
	}
}

func TestAdvanceWinner_PromotesToParent(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p1 := seedParticipant(t, "UID001", "Alice", 10)
	p2 := seedParticipant(t, "UID002", "Bob", 11)
	p3 := seedParticipant(t, "UID003", "Charlie", 12)
	p4 := seedParticipant(t, "UID004", "Dave", 13)

	tour := model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "in_progress"}
	config.DB.Create(&tour)

	// Round 1 matches
	m1 := model.TournamentMatch{TournamentID: tour.ID, Round: 1, MatchNumber: 1, Player1ID: &p1.ID, Player1Name: p1.Name, Player2ID: &p2.ID, Player2Name: p2.Name, Status: "ready"}
	m2 := model.TournamentMatch{TournamentID: tour.ID, Round: 1, MatchNumber: 2, Player1ID: &p3.ID, Player1Name: p3.Name, Player2ID: &p4.ID, Player2Name: p4.Name, Status: "ready"}
	config.DB.Create(&m1)
	config.DB.Create(&m2)

	// Final match (parent)
	final := model.TournamentMatch{TournamentID: tour.ID, Round: 2, MatchNumber: 1, Status: "scheduled"}
	config.DB.Create(&final)

	// Link children to parent
	config.DB.Model(&m1).Updates(map[string]interface{}{"parent_match_id": final.ID, "parent_slot": 1})
	config.DB.Model(&m2).Updates(map[string]interface{}{"parent_match_id": final.ID, "parent_slot": 2})

	// Submit m1 result — Alice wins
	advanceWinner(tour.ID, &m1, p1.ID)

	var updatedFinal model.TournamentMatch
	config.DB.First(&updatedFinal, final.ID)
	if updatedFinal.Player1ID == nil || *updatedFinal.Player1ID != p1.ID {
		t.Error("expected Alice promoted to final player1")
	}
	if updatedFinal.Player1Name != "Alice" {
		t.Errorf("expected Alice name, got %s", updatedFinal.Player1Name)
	}
}

func TestGetActiveMatchByUID_HTTP(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()

	p1 := seedParticipant(t, "UID001", "Alice", 10)
	p2 := seedParticipant(t, "UID002", "Bob", 11)
	tour := model.Tournament{Name: "Cup", MaxPlayers: 8, Status: "in_progress"}
	config.DB.Create(&tour)
	config.DB.Create(&model.TournamentPlayer{TournamentID: tour.ID, ParticipantID: p1.ID, Name: p1.Name})
	config.DB.Create(&model.TournamentPlayer{TournamentID: tour.ID, ParticipantID: p2.ID, Name: p2.Name})

	m := model.TournamentMatch{
		TournamentID: tour.ID,
		Round:        1,
		MatchNumber:  1,
		Player1ID:    &p1.ID,
		Player1Name:  p1.Name,
		Player2ID:    &p2.ID,
		Player2Name:  p2.Name,
		Status:       "ready",
		RoomID:       "ABCD",
	}
	config.DB.Create(&m)
	config.DB.Model(&tour).Update("current_match_id", m.ID)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/tournaments/active-match/:uid", GetActiveMatchByUID)

	req := httptest.NewRequest(http.MethodGet, "/tournaments/active-match/UID001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			HasMatch bool `json:"has_match"`
			RoomCode string `json:"room_code"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Data.HasMatch {
		t.Error("expected has_match true")
	}
	if resp.Data.RoomCode != "ABCD" {
		t.Errorf("expected room code ABCD, got %s", resp.Data.RoomCode)
	}
}
