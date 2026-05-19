package controller

import (
	"log"
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func SubmitPureGame(c *gin.Context) {
	var result model.PureGameResult
	if err := c.ShouldBindJSON(&result); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	// Verify participant exists
	var participant model.Participant
	if err := config.DB.First(&participant, result.ParticipantID).Error; err != nil {
		response.Error(c, http.StatusBadRequest, "Participant not found")
		return
	}

	if !participant.IsPremium {
		response.Error(c, http.StatusForbidden, "Pay first")
		return
	}

	if err := config.DB.Create(&result).Error; err != nil {
		log.Printf("[game] DB insert failed: %v", err)
		response.Error(c, http.StatusInternalServerError, "Failed to save game result")
		return
	}

	response.CreatedWithMessage(c, "Session recorded", gin.H{"session_id": result.ID})
}
