package controller

import (
	"oamp-backend/internal/config"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type LeaderboardEntry struct {
	Rank            int     `json:"rank"`
	ParticipantID   uint    `json:"participant_id"`
	Name            string  `json:"name"`
	Grade           string  `json:"grade"`
	Age             int     `json:"age"`
	VisuoSpatialFit float64 `json:"visuo_spatial_fit"`
	TotalTime       float64 `json:"total_time"`
	LevelReached    int     `json:"level_reached"`
	DexterityScore  float64 `json:"dexterity_score"`
}

func fetchLeaderboard(limit int) []LeaderboardEntry {
	query := `
		SELECT
			ROW_NUMBER() OVER (ORDER BY sub.visuo_spatial_fit DESC, sub.total_time ASC) AS rank,
			sub.participant_id,
			p.name,
			p.grade,
			p.age,
			sub.visuo_spatial_fit,
			sub.total_time,
			sub.level_reached,
			sub.dexterity_score
		FROM (
			SELECT DISTINCT ON (participant_id) *
			FROM game_sessions
			ORDER BY participant_id, visuo_spatial_fit DESC, total_time ASC
		) sub
		JOIN participants p ON p.id = sub.participant_id
		ORDER BY sub.visuo_spatial_fit DESC, sub.total_time ASC
	`

	var entries []LeaderboardEntry
	if limit > 0 {
		config.DB.Raw(query+" LIMIT ?", limit).Scan(&entries)
	} else {
		config.DB.Raw(query).Scan(&entries)
	}
	return entries
}

func GetLeaderboard(c *gin.Context) {
	entries := fetchLeaderboard(10)
	response.OKWithMessage(c, "Leaderboard fetched successfully", entries)
}
