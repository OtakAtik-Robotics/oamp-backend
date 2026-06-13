package controller

import (
	"net/http"

	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

// SyncParticipants — batch insert participants from local server
func SyncParticipants(c *gin.Context) {
	var rows []model.Participant
	if err := c.ShouldBindJSON(&rows); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if len(rows) == 0 {
		response.OK(c, gin.H{"synced": 0})
		return
	}
	config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	response.OK(c, gin.H{"synced": len(rows)})
}

// SyncGameSessions — batch insert game sessions from local server
func SyncGameSessions(c *gin.Context) {
	var rows []model.GameSession
	if err := c.ShouldBindJSON(&rows); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if len(rows) == 0 {
		response.OK(c, gin.H{"synced": 0})
		return
	}
	config.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	response.OK(c, gin.H{"synced": len(rows)})
}

// SyncGameResults — batch upsert game results (keyed by uid)
func SyncGameResults(c *gin.Context) {
	var rows []model.GameResult
	if err := c.ShouldBindJSON(&rows); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if len(rows) == 0 {
		response.OK(c, gin.H{"synced": 0})
		return
	}
	for _, r := range rows {
		config.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "uid"}},
			DoUpdates: clause.AssignmentColumns([]string{"mode", "task01", "task02", "task03", "task04", "task05", "task06", "task07", "task08", "task_avg", "cognitive_age", "visuo_spatial", "cog_age_list", "variant_list", "client_ts"}),
		}).Create(&r)
	}
	response.OK(c, gin.H{"synced": len(rows)})
}
