package controller

import (
	"fmt"
	"net/http"
	"time"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/llm"

	"github.com/gin-gonic/gin"
)

const fallbackAnalysis = "Mohon maaf, layanan AI Health Analysis saat ini sedang sibuk atau tidak dapat diakses akibat gangguan jaringan. Silakan coba beberapa saat lagi."

func GetParticipantAnalysis(c *gin.Context) {
	uid := c.Param("uid")

	// Fetch participant
	var participant model.Participant
	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Participant not found",
			"data":    nil,
		})
		return
	}

	// Return cached analysis if exists
	if participant.AiAnalysis != "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Analysis from cache",
			"data": gin.H{
				"analysis": participant.AiAnalysis,
			},
		})
		return
	}

	// Fetch all game sessions for this participant
	var sessions []model.GameSession
	if err := config.DB.Where("participant_id = ?", participant.ID).Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to fetch session data",
			"data":    nil,
		})
		return
	}

	// Calculate averages
	var avgVisuo, avgDex float64
	if len(sessions) > 0 {
		for _, s := range sessions {
			avgVisuo += s.VisuoSpatialFit
			avgDex += s.DexterityScore
		}
		avgVisuo /= float64(len(sessions))
		avgDex /= float64(len(sessions))
	}

	// Calculate BMI: Weight / ((Height/100) * (Height/100))
	heightM := participant.Height / 100
	bmi := 0.0
	if heightM > 0 {
		bmi = participant.Weight / (heightM * heightM)
	}

	// Build structured prompt
	prompt := fmt.Sprintf(`Analisis data kesehatan anak berikut dan berikan rekomendasi:

**Data Peserta:**
| Field | Value |
|-------|-------|
| Umur | %d tahun |
| Gender | %s |
| BMI | %.1f |
| SpO2 | %.0f%% |
| Heart Rate | %d bpm |
| Grip Strength | %.1f kg |

**Data Motorik (rata-rata %d sesi):**
| Metric | Score |
|--------|-------|
| Visuo-Spatial | %.2f |
| Dexterity | %.2f |

Ikuti format output yang sudah ditentukan di system prompt. Maksimal 150 kata.`, participant.Age, participant.Gender, bmi, participant.SpO2, participant.HeartRate, participant.GripStrength, len(sessions), avgVisuo, avgDex)

	// Call LLM provider
	analysis, err := callLLMProvider(prompt)
	if err != nil {
		// Graceful degradation: HTTP 200 OK with status "fallback"
		c.JSON(http.StatusOK, gin.H{
			"status":  "fallback",
			"message": "AI service offline",
			"data": gin.H{
				"analysis": fallbackAnalysis,
			},
		})
		return
	}

	// Save analysis to cache
	now := time.Now()
	config.DB.Model(&participant).Updates(map[string]interface{}{
		"ai_analysis":            analysis,
		"ai_analysis_updated_at": &now,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Analysis generated",
		"data": gin.H{
			"analysis": analysis,
		},
	})
}

// callLLMProvider initializes the configured LLM provider and generates analysis
func callLLMProvider(prompt string) (string, error) {
	provider, err := llm.NewProvider()
	if err != nil {
		return "", err
	}

	return provider.GenerateHealthAnalysis(prompt)
}
