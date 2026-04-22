package controller

import (
	"bytes"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"
)

var (
	midtransServerKey string
	midtransOnce      sync.Once
)

func initMidtrans() {
	midtransOnce.Do(func() {
		midtransServerKey = os.Getenv("MIDTRANS_SERVER_KEY")
		// Suppress debug/info logs that leak Auth header (Base64 Server Key)
		midtrans.DefaultLoggerLevel = &midtrans.LoggerImplementation{LogLevel: midtrans.LogError}
	})
}

func Checkout(c *gin.Context) {
	initMidtrans()

	if midtransServerKey == "" {
		response.Error(c, http.StatusServiceUnavailable, "Payment service not configured")
		return
	}

	uid := c.Param("uid")

	var participant model.Participant
	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	orderID := fmt.Sprintf("OAMP-%s-%d", uid, time.Now().UnixNano())
	amount := int64(10000)

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: amount,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: participant.Name,
		},
	}

	midtrans.ServerKey = midtransServerKey
	midtrans.Environment = midtrans.Sandbox

	var s snap.Client
	s.New(midtransServerKey, midtrans.Sandbox)
	snapResp, err := s.CreateTransaction(req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create transaction")
		return
	}

	response.OKWithMessage(c, "Checkout initiated", gin.H{
		"token":        snapResp.Token,
		"redirect_url": snapResp.RedirectURL,
		"order_id":     orderID,
		"amount":       amount,
		"currency":     "IDR",
	})
}

func PaymentWebhook(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Signature validation: SHA512(order_id + status_code + gross_amount + ServerKey)
	orderID, _ := payload["order_id"].(string)
	statusCode, _ := payload["status_code"].(string)
	grossAmount, _ := payload["gross_amount"].(string)
	signatureKey, _ := payload["signature_key"].(string)

	if orderID == "" || statusCode == "" || grossAmount == "" || signatureKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "invalid payload"})
		return
	}

	if !verifySignature(orderID, statusCode, grossAmount, signatureKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "invalid signature"})
		return
	}

	status, ok := payload["transaction_status"].(string)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Settlement (QRIS, GoPay) or capture (credit card)
	if status == "settlement" || status == "capture" {
		parts := strings.SplitN(orderID, "-", 2)
		if len(parts) == 2 {
			uid := parts[1]
			lastIdx := strings.LastIndex(uid, "-")
			if lastIdx > 0 {
				uid = uid[:lastIdx]
			}
			if config.DB != nil {
				result := config.DB.Model(&model.Participant{}).Where("uid = ?", uid).Update("is_premium", true)
				if result.Error != nil {
					log.Printf("[webhook] failed to update premium status for %s: %v", uid, result.Error)
				} else {
					go sendTelegramNotification(uid)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func sendTelegramNotification(uid string) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return
	}

	body, _ := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       fmt.Sprintf("🎉 LUNAS! Peserta UID: *%s* baru saja membuka akses Premium AI Analysis (Rp 10.000).", uid),
		"parse_mode": "Markdown",
	})

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[telegram] failed to send notification: %v", err)
		return
	}
	resp.Body.Close()
}

func SimulatePaymentSuccess(c *gin.Context) {
	uid := c.Param("uid")

	var participant model.Participant
	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	if err := config.DB.Model(&participant).Update("is_premium", true).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update premium status")
		return
	}

	go sendTelegramNotification(uid)

	response.OKWithMessage(c, "Payment successful", gin.H{
		"uid":        participant.UID,
		"is_premium": true,
		"paid_at":    time.Now(),
	})
}

func verifySignature(orderID, statusCode, grossAmount, signatureKey string) bool {
	hash := sha512.Sum512([]byte(orderID + statusCode + grossAmount + midtransServerKey))
	return fmt.Sprintf("%x", hash) == signatureKey
}
