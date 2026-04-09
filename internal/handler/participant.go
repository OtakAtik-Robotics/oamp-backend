package handler

import (
	"bdt-server/internal/config"
	"bdt-server/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterParticipant(c *gin.Context) {
	var participant model.Participant
	if err := c.ShouldBindJSON(&participant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Create(&participant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register participant"})
		return
	}

	c.JSON(http.StatusCreated, participant)
}

func GetParticipantByUID(c *gin.Context) {
	uid := c.Param("uid")
	var participant model.Participant

	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found"})
		return
	}

	c.JSON(http.StatusOK, participant)
}