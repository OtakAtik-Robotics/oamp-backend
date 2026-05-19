package model

import "time"

type Participant struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	UID               string    `json:"uid" binding:"required" gorm:"uniqueIndex;not null"`
	Name              string    `json:"name" binding:"required" gorm:"size:100;not null"`
	Age               int       `json:"age" binding:"required,gte=3" gorm:"not null"`
	Grade             string    `json:"grade" binding:"required" gorm:"size:30"`
	Gender            string    `json:"gender" binding:"required,oneof=male female" gorm:"size:10"`
	Height            float64   `json:"height" binding:"required,gt=0"`
	Weight            float64   `json:"weight" binding:"required,gt=0"`
	HeartRate         int       `json:"heart_rate" binding:"omitempty,gte=40,lte=220"`
	SpO2              float64   `json:"spo2" binding:"omitempty,gte=0,lte=100"`
	GripStrength      float64   `json:"grip_strength" binding:"omitempty,gte=0"`
	IsPremium         bool      `json:"is_premium" gorm:"default:false"`
	AiAnalysis        string    `json:"ai_analysis" gorm:"type:text"`
	AiAnalysisUpdatedAt *time.Time `json:"ai_analysis_updated_at"`
	CreatedAt         time.Time `json:"created_at"`
}

type EventBatch struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" binding:"required" gorm:"size:100;not null"`
	IsActive  bool      `json:"is_active" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

type GameSession struct {
	ID              uint          `json:"id" gorm:"primaryKey"`
	ParticipantID   uint          `json:"participant_id" binding:"required" gorm:"not null"`
	Participant     Participant   `json:"-" binding:"-" gorm:"foreignKey:ParticipantID"`
	EventBatchID    uint          `json:"event_batch_id" gorm:"not null;default:1"`
	EventBatch      EventBatch    `json:"-" binding:"-" gorm:"foreignKey:EventBatchID"`
	Mode            string        `json:"mode" gorm:"size:20"`
	LevelReached    int           `json:"level_reached"`
	TotalTime       float64       `json:"total_time"`
	CognitiveAge    int           `json:"cognitive_age"`
	VisuoSpatialFit float64       `json:"visuo_spatial_fit"`
	DexterityScore  float64       `json:"dexterity_score"`
	CreatedAt       time.Time     `json:"created_at"`
}

type FaceExpressionLog struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	SessionID       uint      `json:"session_id" gorm:"not null"`
	Level           int       `json:"level"`
	DominantEmotion string    `json:"dominant_emotion" gorm:"size:50"`
	Timestamp       time.Time `json:"timestamp"`
}

type DatasetCapture struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	SessionID    uint      `json:"session_id" gorm:"not null"`
	CameraSource int       `json:"camera_source"`
	ImagePath    string    `json:"image_path" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
}

type QuizResult struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ParticipantID uint      `json:"participant_id" binding:"required" gorm:"not null"`
	Score         int       `json:"score" binding:"required,gte=0"`
	AnswersData   string    `json:"answers_data" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at"`
}

type PureGameResult struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	ParticipantID       uint      `json:"participant_id" binding:"required" gorm:"not null"`
	GameScore           int       `json:"game_score" binding:"required,gte=0"`
	BlocksHit           int       `json:"blocks_hit" binding:"gte=0"`
	HandTrackingStatus  string    `json:"hand_tracking_status" gorm:"size:20"`
	PlayDuration        float64   `json:"play_duration" binding:"gte=0"`
	Timestamp           time.Time `json:"timestamp"`
	CreatedAt           time.Time `json:"created_at"`
}

// 1v1 Match Room (stored in Go memory, not DB — managed via WebSocket)
// Fields track room state for matchmaking

type MatchRoom struct {
	ID            string    `json:"id" gorm:"primaryKey"` // 4-char code e.g. "ABCD"
	Status        string    `json:"status"`               // waiting, ready, playing
	Player1Name   string    `json:"player1_name"`
	Player2Name   string    `json:"player2_name"`
	Player1Ready  bool      `json:"player1_ready"`
	Player2Ready  bool      `json:"player2_ready"`
	LastActivity  time.Time `json:"last_activity"`
}
