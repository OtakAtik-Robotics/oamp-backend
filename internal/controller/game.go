package controller

import (
	"net/http"

	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// SubmitGameResult — POST /api/v1/game/submit
// Accepts game client payload (bracelet UID scan → store task results)
// mode: "training" → game_results + game_sessions (leaderboard training)
// mode: "competition" → game_results + game_sessions (leaderboard competition)
func SubmitGameResult(c *gin.Context) {
	var result model.GameResult
	if err := c.ShouldBindJSON(&result); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	if result.UID == "" {
		response.Error(c, http.StatusBadRequest, "uid is required")
		return
	}

	// Default to training if not specified
	if result.Mode == "" {
		result.Mode = "training"
	}

	// Find participant by UID from bracelet
	var participant model.Participant
	if err := config.DB.Where("uid = ?", result.UID).First(&participant).Error; err != nil {
		response.Error(c, http.StatusBadRequest, "Participant not found. Please register first.")
		return
	}

	// Always save to game_results (AI analysis)
	saveGameResult(&result)

	// Save to game_sessions for both training and competition (leaderboard)
	if err := saveGameSession(&result, participant.ID); err != nil {
		// log but don't fail the request — game_results was saved successfully
	}

	response.CreatedWithMessage(c, "Game result recorded", gin.H{
		"uid":      result.UID,
		"mode":     result.Mode,
		"task_avg": result.TaskAvg,
	})
}

// saveGameResult upserts into game_results (AI analysis)
func saveGameResult(result *model.GameResult) {
	config.DB.Exec(
		`INSERT INTO game_results (uid, mode, nick_name, gender, age, task01, task02, task03, task04, task05, task06, task07, task08, task_avg, cognitive_age, visuo_spatial, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT (uid) DO UPDATE SET
		 	mode = EXCLUDED.mode,
		 	nick_name = EXCLUDED.nick_name,
		 	gender = EXCLUDED.gender,
		 	age = EXCLUDED.age,
		 	task01 = EXCLUDED.task01,
		 	task02 = EXCLUDED.task02,
		 	task03 = EXCLUDED.task03,
		 	task04 = EXCLUDED.task04,
		 	task05 = EXCLUDED.task05,
		 	task06 = EXCLUDED.task06,
		 	task07 = EXCLUDED.task07,
		 	task08 = EXCLUDED.task08,
		 	task_avg = EXCLUDED.task_avg,
		 	cognitive_age = EXCLUDED.cognitive_age,
		 	visuo_spatial = EXCLUDED.visuo_spatial,
		 	created_at = EXCLUDED.created_at`,
		result.UID, result.Mode, result.NickName, result.Gender, result.Age,
		result.Task01, result.Task02, result.Task03, result.Task04,
		result.Task05, result.Task06, result.Task07, result.Task08,
		result.TaskAvg, result.CognitiveAge, result.VisuoSpatial,
	)
}

// saveGameSession computes leaderboard fields and inserts into game_sessions
func saveGameSession(result *model.GameResult, participantID uint) error {
	// Count completed levels (non-zero tasks)
	levelReached := 0
	if result.Task01 > 0 {
		levelReached++
	}
	if result.Task02 > 0 {
		levelReached++
	}
	if result.Task03 > 0 {
		levelReached++
	}
	if result.Task04 > 0 {
		levelReached++
	}
	if result.Task05 > 0 {
		levelReached++
	}
	if result.Task06 > 0 {
		levelReached++
	}
	if result.Task07 > 0 {
		levelReached++
	}
	if result.Task08 > 0 {
		levelReached++
	}

	// visuo_spatial_fit: normalize 0-100 → 0.0-1.0
	visuoSpatialFit := result.VisuoSpatial / 100.0

	// dexterity_score: ratio cognitive_age / real_age
	dexterityScore := 0.0
	if result.Age > 0 && result.CognitiveAge > 0 {
		dexterityScore = result.CognitiveAge / float64(result.Age)
		if dexterityScore > 2.0 {
			dexterityScore = 2.0
		}
	}

	// Compute score using same formula as leaderboard
	score := float64(levelReached)*10 + visuoSpatialFit*50 + dexterityScore*0.2

	// Get active batch
	var batchID uint = 1
	config.DB.Model(&model.EventBatch{}).Where("is_active = ?", true).Select("id").Scan(&batchID)

	session := model.GameSession{
		ParticipantID:   participantID,
		EventBatchID:    batchID,
		Mode:            result.Mode,
		LevelReached:    levelReached,
		TotalTime:       result.TaskAvg,
		CognitiveAge:    int(result.CognitiveAge),
		VisuoSpatialFit: visuoSpatialFit,
		DexterityScore:  dexterityScore,
		Score:           score,
	}

	// Upsert: delete existing + insert new (same as debug endpoint)
	config.DB.Exec(`DELETE FROM game_sessions WHERE participant_id = ? AND event_batch_id = ?`, participantID, batchID)
	return config.DB.Create(&session).Error
}

// GetParticipantByUID — GET /api/v1/participants/uid/:uid
func GetParticipantByUID(c *gin.Context) {
	uid := c.Param("uid")
	var participant model.Participant
	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}
	response.OKWithMessage(c, "Participant found", participant)
}