package model

import "time"

type Participant struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UID          string    `json:"uid" binding:"required" gorm:"uniqueIndex;not null"`
	Name         string    `json:"name" binding:"required" gorm:"size:100;not null"`
	Age          int       `json:"age" binding:"required,gte=3,lte=18" gorm:"not null"`
	Grade        string    `json:"grade" binding:"required" gorm:"size:20"`
	Gender       string    `json:"gender" binding:"required,oneof=male female" gorm:"size:10"`
	Height       float64   `json:"height" binding:"required,gt=0"`
	Weight       float64   `json:"weight" binding:"required,gt=0"`
	HeartRate    int       `json:"heart_rate" binding:"omitempty,gte=40,lte=220"`
	SpO2         float64   `json:"spo2" binding:"omitempty,gte=0,lte=100"`
	GripStrength float64   `json:"grip_strength" binding:"omitempty,gte=0"`
	CreatedAt    time.Time `json:"created_at"`
}

type GameSession struct {
	ID              uint        `json:"id" gorm:"primaryKey"`
	ParticipantID   uint        `json:"participant_id" binding:"required" gorm:"not null"`
	Participant     Participant `json:"-" gorm:"foreignKey:ParticipantID"`
	Mode            string      `json:"mode" gorm:"size:20"`
	LevelReached    int         `json:"level_reached"`
	TotalTime       float64     `json:"total_time"`
	CognitiveAge    int         `json:"cognitive_age"`
	VisuoSpatialFit float64     `json:"visuo_spatial_fit"`
	DexterityScore  float64     `json:"dexterity_score"`
	CreatedAt       time.Time   `json:"created_at"`
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
