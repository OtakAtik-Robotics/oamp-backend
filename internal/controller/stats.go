package controller

import (
	"oamp-backend/internal/config"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type StatsResponse struct {
	TotalParticipants int64   `json:"total_participants"`
	AvgTime           float64 `json:"avg_time"`
	MinTime           float64 `json:"min_time"`
	MaxTime           float64 `json:"max_time"`
	AvgCognitiveAge   float64 `json:"avg_cognitive_age" gorm:"column:avg_cognitive_age"`
	AvgVisuoSpatial   float64 `json:"avg_visuo_spatial" gorm:"column:avg_visuo_spatial"`
	TotalMale         int64   `json:"total_male"`
	TotalFemale       int64   `json:"total_female"`
}

type LevelAvg struct {
	Level string  `json:"level"`
	Avg   float64 `json:"avg"`
}

type TimelinePoint struct {
	Name    string  `json:"name"`
	AvgTime float64 `json:"avg_time"`
}

// GetStats — GET /api/v1/stats
func GetStats(c *gin.Context) {
	var stats StatsResponse

	// Total participants
	config.DB.Raw("SELECT COUNT(*) FROM participants").Scan(&stats.TotalParticipants)

	// Time stats from game_results
	config.DB.Raw(`
		SELECT
			COALESCE(ROUND(AVG(task_avg)::numeric, 2), 0) AS avg_time,
			COALESCE(ROUND(MIN(task_avg)::numeric, 2), 0) AS min_time,
			COALESCE(ROUND(MAX(task_avg)::numeric, 2), 0) AS max_time
		FROM game_results WHERE task_avg > 0
	`).Scan(&stats)

	// Cognitive + visuo-spatial averages
	config.DB.Raw(`
		SELECT
			COALESCE(ROUND(AVG(cognitive_age)::numeric, 1), 0) AS avg_cognitive_age,
			COALESCE(ROUND(AVG(visuo_spatial)::numeric, 1), 0) AS avg_visuo_spatial
		FROM game_results WHERE cognitive_age > 0
	`).Scan(&stats)

	// Gender distribution
	config.DB.Raw(`SELECT COUNT(*) FROM participants WHERE gender = 'male'`).Scan(&stats.TotalMale)
	config.DB.Raw(`SELECT COUNT(*) FROM participants WHERE gender = 'female'`).Scan(&stats.TotalFemale)

	// Per-level averages
	var levelAvgs []LevelAvg
	config.DB.Raw(`
		SELECT 'L1' AS level, COALESCE(ROUND(AVG(task01)::numeric, 2), 0) AS avg FROM game_results WHERE task01 > 0
		UNION ALL SELECT 'L2', COALESCE(ROUND(AVG(task02)::numeric, 2), 0) FROM game_results WHERE task02 > 0
		UNION ALL SELECT 'L3', COALESCE(ROUND(AVG(task03)::numeric, 2), 0) FROM game_results WHERE task03 > 0
		UNION ALL SELECT 'L4', COALESCE(ROUND(AVG(task04)::numeric, 2), 0) FROM game_results WHERE task04 > 0
		UNION ALL SELECT 'L5', COALESCE(ROUND(AVG(task05)::numeric, 2), 0) FROM game_results WHERE task05 > 0
		UNION ALL SELECT 'L6', COALESCE(ROUND(AVG(task06)::numeric, 2), 0) FROM game_results WHERE task06 > 0
		UNION ALL SELECT 'L7', COALESCE(ROUND(AVG(task07)::numeric, 2), 0) FROM game_results WHERE task07 > 0
		UNION ALL SELECT 'L8', COALESCE(ROUND(AVG(task08)::numeric, 2), 0) FROM game_results WHERE task08 > 0
	`).Scan(&levelAvgs)

	// Timeline — last 20 participants
	var timeline []TimelinePoint
	config.DB.Raw(`
		SELECT nick_name AS name, ROUND(task_avg::numeric, 2) AS avg_time
		FROM game_results
		WHERE task_avg > 0
		ORDER BY created_at DESC
		LIMIT 20
	`).Scan(&timeline)

	response.OKWithMessage(c, "Stats fetched", gin.H{
		"stats":       stats,
		"level_avgs":  levelAvgs,
		"timeline":    timeline,
	})
}
