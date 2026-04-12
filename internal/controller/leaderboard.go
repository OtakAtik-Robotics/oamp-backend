package controller

import (
	"oamp-backend/internal/config"
	"oamp-backend/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

type LeaderboardEntry struct {
	Rank            int     `json:"rank"`
	ParticipantID   uint    `json:"participant_id"`
	UID             string  `json:"uid"`
	Name            string  `json:"name"`
	Grade           string  `json:"grade"`
	Age             int     `json:"age"`
	VisuoSpatialFit float64 `json:"visuo_spatial_fit"`
	TotalTime       float64 `json:"total_time"`
	LevelReached    int     `json:"level_reached"`
	DexterityScore  float64 `json:"dexterity_score"`
	Score           float64 `json:"score"`
}

type TimelineEntry struct {
	Name      string    `json:"name"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}

func fetchLeaderboard(limit int) []LeaderboardEntry {
	query := `
        SELECT
            ROW_NUMBER() OVER (ORDER BY sub.score DESC) AS rank,
            sub.participant_id,
            p.uid,
            p.name,
            p.grade,
            p.age,
            sub.visuo_spatial_fit,
            sub.total_time,
            sub.level_reached,
            sub.dexterity_score,
            sub.score
        FROM (
            SELECT DISTINCT ON (participant_id)
                *,
                ((level_reached * 1000) + (visuo_spatial_fit * 10) - total_time) AS score
            FROM game_sessions
            ORDER BY participant_id, ((level_reached * 1000) + (visuo_spatial_fit * 10) - total_time) DESC
        ) sub
        JOIN participants p ON p.id = sub.participant_id
        ORDER BY sub.score DESC
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

func GetLeaderboardTimeline(c *gin.Context) {
	query := `
        SELECT 
            p.name, 
            ((gs.level_reached * 1000) + (gs.visuo_spatial_fit * 10) - gs.total_time) AS score, 
            gs.created_at
        FROM game_sessions gs
        JOIN participants p ON p.id = gs.participant_id
        ORDER BY gs.created_at ASC
    `

	var entries []TimelineEntry
	config.DB.Raw(query).Scan(&entries)
	response.OKWithMessage(c, "Timeline fetched successfully", entries)
}