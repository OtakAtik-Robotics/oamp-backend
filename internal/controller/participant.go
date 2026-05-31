package controller

import (
	"log"
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func RegisterParticipant(c *gin.Context) {
	var participant model.Participant
	if err := c.ShouldBindJSON(&participant); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	log.Printf("[participant] registering UID=%s Name=%s", participant.UID, participant.Name)

	var existing model.Participant
	if err := config.DB.Where("uid = ?", participant.UID).First(&existing).Error; err == nil {
		response.Error(c, http.StatusConflict, "UID already registered")
		return
	}

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

// GET /api/v1/participants/lookup/:nickname — find participant by name
func LookupParticipant(c *gin.Context) {
	nickname := c.Param("nickname")

	var participant model.Participant
	if err := config.DB.Where("name ILIKE ?", nickname).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uid":    participant.UID,
		"name":   participant.Name,
		"age":    participant.Age,
		"gender": participant.Gender,
	})
}

type ParticipantWithScores struct {
	ID              uint       `json:"id"`
	UID             string     `json:"uid"`
	Name            string     `json:"name"`
	Age             int        `json:"age"`
	Grade           string     `json:"grade"`
	Gender          string     `json:"gender"`
	Height          float64    `json:"height"`
	Weight          float64    `json:"weight"`
	HeartRate       int        `json:"heart_rate"`
	SpO2            float64    `json:"spo2"`
	GripStrength    float64    `json:"grip_strength"`
	IsPremium       bool       `json:"is_premium"`
	AiAnalysis      string     `json:"ai_analysis"`
	AiAnalysisUpdatedAt *time.Time `json:"ai_analysis_updated_at"`
	CreatedAt       time.Time  `json:"created_at"`
	LevelReached    int        `json:"level_reached"`
	TotalTime       float64    `json:"total_time"`
	VisuoSpatialFit float64    `json:"visuo_spatial_fit"`
	DexterityScore  float64    `json:"dexterity_score"`
	Score           float64    `json:"score"`
}

func GetParticipantsWithScores(c *gin.Context) {
	batchID := c.Query("batch_id")

	query := `
		SELECT
			p.id, p.uid, p.name, p.age, p.grade, p.gender,
			p.height, p.weight, p.heart_rate, p.spo2, p.grip_strength,
			p.is_premium, p.ai_analysis, p.ai_analysis_updated_at, p.created_at,
			COALESCE(best.level_reached, 0) AS level_reached,
			COALESCE(best.total_time, 0) AS total_time,
			COALESCE(best.visuo_spatial_fit, 0) AS visuo_spatial_fit,
			COALESCE(best.dexterity_score, 0) AS dexterity_score,
			COALESCE(best.score, 0) AS score
		FROM participants p
		LEFT JOIN LATERAL (
			SELECT participant_id, level_reached, total_time, visuo_spatial_fit, dexterity_score, score
			FROM game_sessions gs
			WHERE gs.participant_id = p.id`

	var args []any
	if batchID != "" && batchID != "all" {
		if id, err := strconv.Atoi(batchID); err == nil {
			query += ` AND gs.event_batch_id = ?`
			args = append(args, id)
		}
	}

	query += ` ORDER BY gs.score DESC LIMIT 1
		) best ON true
		ORDER BY best.score DESC NULLS LAST, p.name ASC`

	var results []ParticipantWithScores
	config.DB.Raw(query, args...).Scan(&results)
	response.OKWithMessage(c, "Participants fetched successfully", results)
}

func GetParticipantSessions(c *gin.Context) {
	uid := c.Param("uid")

	var participant model.Participant
	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, 404, "Participant not found")
		return
	}

	var sessions []model.GameSession
	config.DB.Where("participant_id = ?", participant.ID).Order("created_at desc").Find(&sessions)

	response.OKWithMessage(c, "Sessions fetched successfully", sessions)
}
