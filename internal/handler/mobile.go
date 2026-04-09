package handler

import (
	"net/http"
	"server/internal/config"
	"server/internal/model"

	"github.com/gin-gonic/gin"
)

func GetParticipantResult(c *gin.Context) {
	uid := c.Param("uid")
	var participant model.Participant

	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found"})
		return
	}

	var session model.GameSession
	if err := config.DB.Where("participant_id = ?", participant.ID).Order("created_at desc").First(&session).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"participant": participant,
			"session":     nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"participant": participant,
		"session":     session,
	})
}

func SubmitQuizResult(c *gin.Context) {
	var quizResult model.QuizResult
	if err := c.ShouldBindJSON(&quizResult); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := config.DB.Create(&quizResult).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save quiz result"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Quiz result saved successfully",
		"quiz_id": quizResult.ID,
	})
}