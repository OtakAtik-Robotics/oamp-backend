package controller

import (
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func AppAuth(c *gin.Context) {
	uid := c.Param("uid")
	var participant model.Participant

	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	var sessions []model.GameSession
	config.DB.Where("participant_id = ?", participant.ID).Order("created_at desc").Find(&sessions)

	response.OKWithMessage(c, "Login successful", gin.H{
		"participant": participant,
		"sessions":    sessions,
	})
}

func SubmitQuiz(c *gin.Context) {
	var quizResult model.QuizResult
	if err := c.ShouldBindJSON(&quizResult); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := config.DB.Create(&quizResult).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save quiz result")
		return
	}

	response.CreatedWithMessage(c, "Quiz result saved successfully", gin.H{"quiz_id": quizResult.ID})
}
