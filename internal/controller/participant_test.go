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

func TestUpdateParticipant_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()
	gin.SetMode(gin.TestMode)

	seedParticipant(t, "UID-HW-01", "Budi", 12)

	payload := map[string]interface{}{
		"height":        170.5,
		"weight":        65.2,
		"grip_strength": 45.0,
		"dexterity":     20.0,
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/api/v1/participants/uid/UID-HW-01", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "uid", Value: "UID-HW-01"}}

	UpdateParticipant(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Data    model.Participant `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Height != 170.5 {
		t.Errorf("height = %v, want 170.5", resp.Data.Height)
	}
	if resp.Data.Weight != 65.2 {
		t.Errorf("weight = %v, want 65.2", resp.Data.Weight)
	}
	if resp.Data.GripStrength != 45.0 {
		t.Errorf("grip_strength = %v, want 45.0", resp.Data.GripStrength)
	}
	if resp.Data.Dexterity != 20.0 {
		t.Errorf("dexterity = %v, want 20.0", resp.Data.Dexterity)
	}

	// Verify DB persistence
	var p model.Participant
	config.DB.Where("uid = ?", "UID-HW-01").First(&p)
	if p.Height != 170.5 || p.Weight != 65.2 || p.GripStrength != 45.0 || p.Dexterity != 20.0 {
		t.Errorf("DB not persisted: h=%v w=%v g=%v d=%v", p.Height, p.Weight, p.GripStrength, p.Dexterity)
	}
}

func TestUpdateParticipant_Partial(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()
	gin.SetMode(gin.TestMode)

	p := seedParticipant(t, "UID-HW-02", "Citra", 14)
	config.DB.Model(p).Updates(map[string]interface{}{
		"height": 160.0, "weight": 55.0, "grip_strength": 30.0, "dexterity": 10.0,
	})

	// Only update height, leave others unchanged
	payload := map[string]interface{}{"height": 165.0}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/api/v1/participants/uid/UID-HW-02", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "uid", Value: "UID-HW-02"}}

	UpdateParticipant(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data model.Participant `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Height != 165.0 {
		t.Errorf("height = %v, want 165.0", resp.Data.Height)
	}
	if resp.Data.Weight != 55.0 {
		t.Errorf("weight should stay 55.0, got %v", resp.Data.Weight)
	}
	if resp.Data.GripStrength != 30.0 {
		t.Errorf("grip_strength should stay 30.0, got %v", resp.Data.GripStrength)
	}
	if resp.Data.Dexterity != 10.0 {
		t.Errorf("dexterity should stay 10.0, got %v", resp.Data.Dexterity)
	}
}

func TestUpdateParticipant_NotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()
	gin.SetMode(gin.TestMode)

	payload := map[string]interface{}{"height": 170.0}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/api/v1/participants/uid/NONEXISTENT", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "uid", Value: "NONEXISTENT"}}

	UpdateParticipant(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateParticipant_NoFields(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()
	gin.SetMode(gin.TestMode)

	seedParticipant(t, "UID-HW-03", "Dian", 16)

	payload := map[string]interface{}{}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/api/v1/participants/uid/UID-HW-03", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "uid", Value: "UID-HW-03"}}

	UpdateParticipant(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateParticipant_InvalidRange(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB()
	gin.SetMode(gin.TestMode)

	seedParticipant(t, "UID-HW-04", "Eko", 10)

	payload := map[string]interface{}{"height": 9999.0} // exceeds lte=300
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/api/v1/participants/uid/UID-HW-04", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "uid", Value: "UID-HW-04"}}

	UpdateParticipant(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
