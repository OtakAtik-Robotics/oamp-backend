package controller

import (
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestPaymentWebhook_InvalidSignature(t *testing.T) {
	midtransServerKey = "test-server-key"
	defer func() { midtransServerKey = "" }()

	r := setupTestRouter()
	r.POST("/webhook", PaymentWebhook)

	payload := map[string]interface{}{
		"order_id":          "OAMP-BCR-001-123456",
		"status_code":       "200",
		"gross_amount":      "10000.00",
		"signature_key":     "wrong_signature",
		"transaction_status": "settlement",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestPaymentWebhook_ValidSignature_Settlement(t *testing.T) {
	midtransServerKey = "test-server-key"
	defer func() { midtransServerKey = "" }()

	orderID := "OAMP-BCR-001-123456"
	statusCode := "200"
	grossAmount := "10000.00"

	hash := sha512.Sum512([]byte(orderID + statusCode + grossAmount + midtransServerKey))
	sig := fmt.Sprintf("%x", hash)

	r := setupTestRouter()
	r.POST("/webhook", PaymentWebhook)

	payload := map[string]interface{}{
		"order_id":           orderID,
		"status_code":        statusCode,
		"gross_amount":       grossAmount,
		"signature_key":      sig,
		"transaction_status": "settlement",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Will be 200 even if DB update fails (no DB in test)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPaymentWebhook_MissingFields(t *testing.T) {
	midtransServerKey = "test-server-key"
	defer func() { midtransServerKey = "" }()

	r := setupTestRouter()
	r.POST("/webhook", PaymentWebhook)

	payload := map[string]interface{}{
		"order_id": "OAMP-BCR-001-123456",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPaymentWebhook_InvalidJSON(t *testing.T) {
	r := setupTestRouter()
	r.POST("/webhook", PaymentWebhook)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Webhook always returns 200 on parse failure
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCheckout_ServiceUnavailable(t *testing.T) {
	// Ensure no MIDTRANS_SERVER_KEY
	os.Unsetenv("MIDTRANS_SERVER_KEY")
	midtransServerKey = ""
	midtransOnce = sync.Once{} // reset

	r := setupTestRouter()
	r.POST("/checkout/:uid", Checkout)

	req := httptest.NewRequest(http.MethodPost, "/checkout/BCR-001", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestInitMidtrans_SetsLogLevel(t *testing.T) {
	os.Setenv("MIDTRANS_SERVER_KEY", "test-key")
	defer os.Unsetenv("MIDTRANS_SERVER_KEY")
	midtransServerKey = ""
	midtransOnce = sync.Once{}

	initMidtrans()

	if midtransServerKey != "test-key" {
		t.Errorf("expected server key to be set, got %s", midtransServerKey)
	}

	// Call again — should not panic or change
	midtransOnce = sync.Once{}
	initMidtrans()
	if midtransServerKey != "test-key" {
		t.Errorf("expected server key to remain set, got %s", midtransServerKey)
	}
}

func TestVerifySignature_Correct(t *testing.T) {
	midtransServerKey = "SB-Mid-server-TEST"
	defer func() { midtransServerKey = "" }()

	orderID := "OAMP-BCR-001-1234567890"
	statusCode := "200"
	grossAmount := "10000.00"

	hash := sha512.Sum512([]byte(orderID + statusCode + grossAmount + midtransServerKey))
	sig := fmt.Sprintf("%x", hash)

	if !verifySignature(orderID, statusCode, grossAmount, sig) {
		t.Error("expected signature to be valid")
	}
}

func TestVerifySignature_Incorrect(t *testing.T) {
	midtransServerKey = "SB-Mid-server-TEST"
	defer func() { midtransServerKey = "" }()

	if verifySignature("order", "200", "10000", "badsig") {
		t.Error("expected signature to be invalid")
	}
}

func TestVerifySignature_TamperedAmount(t *testing.T) {
	midtransServerKey = "SB-Mid-server-TEST"
	defer func() { midtransServerKey = "" }()

	orderID := "OAMP-BCR-001-1234"
	statusCode := "200"
	grossAmount := "10000.00"
	tamperedAmount := "5000.00"

	hash := sha512.Sum512([]byte(orderID + statusCode + grossAmount + midtransServerKey))
	sig := fmt.Sprintf("%x", hash)

	if verifySignature(orderID, statusCode, tamperedAmount, sig) {
		t.Error("expected tampered amount to fail signature check")
	}
}
