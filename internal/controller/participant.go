package controller

import (
	"log"
	"net/http"
	"strconv"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func RegisterParticipant(c *gin.Context) {
	var participant model.Participant
	if err := c.ShouldBindJSON(&participant); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	log.Printf("[participant] registering UID=%s Name=%s", participant.UID, participant.Name)

	if err := config.DB.Create(&participant).Error; err != nil {
		log.Printf("[participant] DB insert failed: %v", err)
		response.Error(c, http.StatusInternalServerError, "Failed to register participant")
		return
	}

	response.CreatedWithMessage(c, "Participant registered successfully", participant)
}

func GetParticipants(c *gin.Context) {
	db := config.DB.Model(&model.Participant{})

	batchID := c.Query("batch_id")
	log.Printf("[participant] GET /participants batch_id=%q raw_query=%q", batchID, c.Request.URL.RawQuery)

	if batchID != "" {
		if id, err := strconv.Atoi(batchID); err == nil {
			db = db.Joins("JOIN game_sessions ON game_sessions.participant_id = participants.id").
				Where("game_sessions.event_batch_id = ?", id).
				Distinct()
		}
	}

	var participants []model.Participant
	if err := db.Find(&participants).Error; err != nil {
		log.Printf("[participant] DB fetch failed: %v", err)
		response.Error(c, http.StatusInternalServerError, "Failed to fetch participants")
		return
	}

	response.OKWithMessage(c, "Participants fetched successfully", participants)
}

// GET /api/v1/participants/id/:id — lookup by numeric DB ID
func GetParticipantByID(c *gin.Context) {
	idStr := c.Param("id")
	var id uint
	if _, err := strconv.ParseUint(idStr, 10, 64); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid ID")
		return
	}

	var participant model.Participant
	if err := config.DB.First(&participant, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	response.OKWithMessage(c, "", participant)
}
