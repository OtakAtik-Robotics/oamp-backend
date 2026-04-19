package controller

import (
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func RobotAuth(c *gin.Context) {
	uid := c.Param("uid")
	var participant model.Participant

	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	response.OKWithMessage(c, "Participant found", participant)
}

type SessionPayload struct {
	Session     model.GameSession         `json:"session"`
	Expressions []model.FaceExpressionLog `json:"expressions"`
	Datasets    []model.DatasetCapture    `json:"datasets"`
}

func SubmitSession(c *gin.Context) {
	var payload SessionPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	// Verify participant exists
	var participant model.Participant
	if err := config.DB.First(&participant, payload.Session.ParticipantID).Error; err != nil {
		response.Error(c, http.StatusBadRequest, "Participant not found")
		return
	}

	if !participant.IsPremium {
		response.Error(c, http.StatusForbidden, "Pay first")
		return
	}

	// Auto-assign EventBatchID: find active batch or create default
	var batch model.EventBatch
	if err := config.DB.Where("is_active = ?", true).First(&batch).Error; err != nil {
		// No active batch, create default
		batch = model.EventBatch{Name: "Sesi Default", IsActive: true}
		if err := config.DB.Create(&batch).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "Failed to create default event batch")
			return
		}
	}
	payload.Session.EventBatchID = batch.ID

	tx := config.DB.Begin()

	if err := tx.Create(&payload.Session).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "Failed to save session")
		return
	}

	for i := range payload.Expressions {
		payload.Expressions[i].SessionID = payload.Session.ID
	}
	if len(payload.Expressions) > 0 {
		if err := tx.Create(&payload.Expressions).Error; err != nil {
			tx.Rollback()
			response.Error(c, http.StatusInternalServerError, "Failed to save face expressions")
			return
		}
	}

	for i := range payload.Datasets {
		payload.Datasets[i].SessionID = payload.Session.ID
	}
	if len(payload.Datasets) > 0 {
		if err := tx.Create(&payload.Datasets).Error; err != nil {
			tx.Rollback()
			response.Error(c, http.StatusInternalServerError, "Failed to save dataset captures")
			return
		}
	}

	tx.Commit()
	response.CreatedWithMessage(c, "Session recorded successfully", gin.H{"session_id": payload.Session.ID})
}

type FaceLogBatch struct {
	SessionID uint                      `json:"session_id"`
	Logs      []model.FaceExpressionLog `json:"logs"`
}

func SubmitFaceLogs(c *gin.Context) {
	var batch FaceLogBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	if len(batch.Logs) == 0 {
		response.Error(c, http.StatusBadRequest, "No logs provided")
		return
	}

	for i := range batch.Logs {
		batch.Logs[i].SessionID = batch.SessionID
	}

	if err := config.DB.Create(&batch.Logs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save face expression logs")
		return
	}

	response.CreatedWithMessage(c, "Face logs saved successfully", gin.H{"count": len(batch.Logs)})
}
