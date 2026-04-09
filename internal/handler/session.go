package handler

import (
	"bdt-server/internal/config"
	"bdt-server/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SessionPayload struct {
	Session     model.GameSession         `json:"session"`
	Expressions []model.FaceExpressionLog `json:"expressions"`
	Datasets    []model.DatasetCapture    `json:"datasets"`
}

func SubmitGameSession(c *gin.Context) {
	var payload SessionPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := config.DB.Begin()

	if err := tx.Create(&payload.Session).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}

	for i := range payload.Expressions {
		payload.Expressions[i].SessionID = payload.Session.ID
	}
	if len(payload.Expressions) > 0 {
		if err := tx.Create(&payload.Expressions).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save expressions"})
			return
		}
	}

	for i := range payload.Datasets {
		payload.Datasets[i].SessionID = payload.Session.ID
	}
	if len(payload.Datasets) > 0 {
		if err := tx.Create(&payload.Datasets).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save datasets"})
			return
		}
	}

	tx.Commit()
	c.JSON(http.StatusCreated, gin.H{
		"message":    "Session recorded successfully",
		"session_id": payload.Session.ID,
	})
}