package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func assertJSON(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	return resp
}

func TestOK(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { OK(c, gin.H{"key": "value"}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	resp := assertJSON(t, w)
	if resp.Status != "success" {
		t.Errorf("expected status success, got %s", resp.Status)
	}
}

func TestError(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) { Error(c, http.StatusBadRequest, "bad request") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	resp := assertJSON(t, w)
	if resp.Status != "error" {
		t.Errorf("expected status error, got %s", resp.Status)
	}
	if resp.Message != "bad request" {
		t.Errorf("expected message 'bad request', got %s", resp.Message)
	}
	if resp.Data != nil {
		t.Error("expected nil data for error response")
	}
}

func TestCreatedWithMessage(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) {
		CreatedWithMessage(c, "created", gin.H{"id": 1})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	resp := assertJSON(t, w)
	if resp.Message != "created" {
		t.Errorf("expected message 'created', got %s", resp.Message)
	}
}

func TestFormatBindError_InvalidJSON(t *testing.T) {
	msg := FormatBindError(errors.New("not a validation error"))
	if msg != "Invalid request body" {
		t.Errorf("expected 'Invalid request body', got %s", msg)
	}
}
