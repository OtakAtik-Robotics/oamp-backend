package controller

import (
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func parseUint(s string, result *uint) (bool, error) {
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return false, err
	}
	*result = uint(val)
	return true, nil
}

type LeaderboardEntry struct {
	Rank            int     `json:"rank"`
	ParticipantID   uint    `json:"participant_id"`
	UID             string  `json:"uid"`
	Name            string  `json:"name"`
	Grade           string  `json:"grade"`
	Age             int     `json:"age"`
	Gender          string  `json:"gender"`
	IsPremium       bool    `json:"is_premium"`
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

func fetchLeaderboard(limit int, batchID *uint) []LeaderboardEntry {
	query := `
        SELECT
            ROW_NUMBER() OVER (ORDER BY sub.score DESC) AS rank,
            sub.participant_id,
            p.uid,
            p.name,
            p.grade,
            p.age,
            p.gender,
            p.is_premium,
            sub.visuo_spatial_fit,
            sub.total_time,
            sub.level_reached,
            sub.dexterity_score,
            sub.score
        FROM (
            SELECT DISTINCT ON (participant_id)
                participant_id,
                visuo_spatial_fit,
                total_time,
                level_reached,
                dexterity_score,
                ROUND(((level_reached * 10) + (visuo_spatial_fit * 50) + (dexterity_score * 0.2)), 2) AS score
            FROM game_sessions`

	var args []any

	if batchID != nil {
		query += ` WHERE event_batch_id = ?`
		args = append(args, *batchID)
	}

	query += `
            ORDER BY participant_id, ((level_reached * 10) + (visuo_spatial_fit * 50) + (dexterity_score * 0.2)) DESC
        ) sub
        JOIN participants p ON p.id = sub.participant_id
        ORDER BY sub.score DESC
    `

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	var entries []LeaderboardEntry
	config.DB.Raw(query, args...).Scan(&entries)
	return entries
}

func GetLeaderboard(c *gin.Context) {
	var batchID *uint
	if idStr := c.Query("batch_id"); idStr != "" {
		if idStr != "all" {
			var id uint
			if _, err := parseUint(idStr, &id); err == nil {
				batchID = &id
			}
		}
	} else {
		var activeID uint
		if err := config.DB.Model(&model.EventBatch{}).Where("is_active = ?", true).Select("id").First(&activeID).Error; err == nil {
			batchID = &activeID
		}
	}
	entries := fetchLeaderboard(0, batchID)
	response.OKWithMessage(c, "Leaderboard fetched successfully", entries)
}

func GetLeaderboardTimeline(c *gin.Context) {
	idStr := c.Query("batch_id")
	var args []any
	query := `
        SELECT
            p.name,
            ROUND(((gs.level_reached * 10) + (gs.visuo_spatial_fit * 50) + (gs.dexterity_score * 0.2)), 2) AS score,
            gs.created_at
        FROM game_sessions gs
        JOIN participants p ON p.id = gs.participant_id`

	if idStr == "all" {
	} else if idStr != "" {
		var id uint
		if _, err := parseUint(idStr, &id); err == nil {
			query += ` WHERE gs.event_batch_id = ?`
			args = append(args, id)
		} else {
			response.OKWithMessage(c, "Timeline fetched successfully", []TimelineEntry{})
			return
		}
	} else {
		var activeID uint
		if err := config.DB.Model(&model.EventBatch{}).Where("is_active = ?", true).Select("id").First(&activeID).Error; err != nil {
			response.OKWithMessage(c, "Timeline fetched successfully", []TimelineEntry{})
			return
		}
		query += ` WHERE gs.event_batch_id = ?`
		args = append(args, activeID)
	}

	query += ` ORDER BY gs.created_at ASC LIMIT 200`

	var entries []TimelineEntry
	config.DB.Raw(query, args...).Scan(&entries)
	response.OKWithMessage(c, "Timeline fetched successfully", entries)
}