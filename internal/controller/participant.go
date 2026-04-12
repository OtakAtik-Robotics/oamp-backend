package controller

import (
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func RegisterParticipant(c *gin.Context) {
	var participant model.Participant
	if err := c.ShouldBindJSON(&participant); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := config.DB.Create(&participant).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to register participant")
		return
	}

	response.CreatedWithMessage(c, "Participant registered successfully", participant)
}
