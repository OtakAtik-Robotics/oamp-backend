package model

import "time"

type Participant struct {
	ID        uint      `gorm:"primaryKey"`
	UID       string    `gorm:"uniqueIndex;not null"`
	Name      string    `gorm:"size:100;not null"`
	Age       int       `gorm:"not null"`
	Gender    string    `gorm:"size:10"`
	Height    float64
	Weight    float64
	HeartRate int
	SpO2      float64
	CreatedAt time.Time
}

type GameSession struct {
	ID              uint      `gorm:"primaryKey"`
	ParticipantID   uint      `gorm:"not null"`
	Mode            string    `gorm:"size:20"`
	LevelReached    int
	TotalTime       float64
	CognitiveAge    int
	VisuoSpatialFit float64
	GripStrength    float64
	DexterityScore  float64
	CreatedAt       time.Time
}

type FaceExpressionLog struct {
	ID              uint      `gorm:"primaryKey"`
	SessionID       uint      `gorm:"not null"`
	Level           int
	DominantEmotion string    `gorm:"size:50"`
	Timestamp       time.Time
}

type DatasetCapture struct {
	ID           uint      `gorm:"primaryKey"`
	SessionID    uint      `gorm:"not null"`
	CameraSource int
	ImagePath    string    `gorm:"not null"`
	CreatedAt    time.Time
}

type QuizResult struct {
	ID            uint      `gorm:"primaryKey"`
	ParticipantID uint      `gorm:"not null"`
	Score         int
	AnswersData   string    `gorm:"type:text"`
	CreatedAt     time.Time
}