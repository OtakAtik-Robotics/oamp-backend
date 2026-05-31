package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// Float64Array implements sql.Scanner and driver.Valuer for JSONB float64 arrays
type Float64Array []float64

func (a *Float64Array) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, a)
}

func (a Float64Array) Value() (driver.Value, error) {
	if a == nil {
		return "[]", nil
	}
	return json.Marshal(a)
}

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
	EventBatchID    uint          `json:"event_batch_id" gorm:"not null;default:1;index:idx_participant_batch"`
	EventBatch      EventBatch    `json:"-" binding:"-" gorm:"foreignKey:EventBatchID"`
	Mode            string        `json:"mode" gorm:"size:20"`
	LevelReached    int           `json:"level_reached"`
	TotalTime       float64       `json:"total_time"`
	CognitiveAge    int           `json:"cognitive_age"`
	VisuoSpatialFit float64       `json:"visuo_spatial_fit"`
	DexterityScore  float64       `json:"dexterity_score"`
	Score           float64       `json:"score"`
	CreatedAt       time.Time     `json:"created_at"`
}

// Room — database-backed 1v1 match room
type Room struct {
	ID           string    `json:"id" gorm:"primaryKey;size:4"` // 4-char code, e.g. "ABCD"
	Status       string    `json:"status" gorm:"size:20;default:waiting"` // waiting|ready|playing|finished
	Player1Name  string    `json:"player1_name" gorm:"size:100"`
	Player2Name  string    `json:"player2_name" gorm:"size:100"`
	Player1Ready bool      `json:"player1_ready" gorm:"default:false"`
	Player2Ready bool      `json:"player2_ready" gorm:"default:false"`
	LastActivity time.Time `json:"last_activity" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at"`
}

// PlayerState — per-player state within a room
type PlayerState struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	RoomID          string    `json:"room_id" gorm:"size:4;index;not null"`
	PlayerName      string    `json:"player_name" gorm:"size:100;not null"`
	CurrentLevel    int       `json:"current_level" gorm:"default:0"`
	ElapsedTime     float64   `json:"elapsed_time" gorm:"default:0"`
	CompletedLevels int       `json:"completed_levels" gorm:"default:0"`
	LevelTimes      Float64Array `json:"level_times" gorm:"type:jsonb"`
	IsFinished      bool      `json:"is_finished" gorm:"default:false"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GameEvent — event payload from desktop game client (not a DB model)
type GameEvent struct {
	Type       string  `json:"type"`
	RoomID     string  `json:"room_id"`
	PlayerName string  `json:"player_name"`
	Level      *int    `json:"level,omitempty"`
	TimeSec    float64 `json:"time_sec,omitempty"`
}

// GameResult — result from desktop game client (bracelet UID scan)
type GameResult struct {
	UID          string    `json:"uid" gorm:"primaryKey"` // one result per participant UID
	Mode         string    `json:"mode" gorm:"size:20;default:'training'"` // "training" | "competition"
	NickName     string    `json:"nick_name" gorm:"size:100"`
	Gender       string    `json:"gender" gorm:"size:10"`
	Age          int       `json:"age"`
	Task01       float64   `json:"task01"`
	Task02       float64   `json:"task02"`
	Task03       float64   `json:"task03"`
	Task04       float64   `json:"task04"`
	Task05       float64   `json:"task05"`
	Task06       float64   `json:"task06"`
	Task07       float64   `json:"task07"`
	Task08       float64   `json:"task08"`
	TaskAvg      float64   `json:"task_avg"`
	CognitiveAge float64   `json:"cognitive_age"`
	VisuoSpatial float64   `json:"visuo_spatial"`
	CreatedAt    time.Time `json:"created_at"`
}

// Tournament — single elimination cup
type Tournament struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	EventBatchID uint      `json:"event_batch_id" gorm:"not null;default:1;index"`
	Name         string    `json:"name" gorm:"size:100;not null"`
	Status       string    `json:"status" gorm:"size:20;default:'registration'"` // registration | ready | in_progress | finished
	MaxPlayers   int       `json:"max_players" gorm:"default:8"`
	PlayerCount  int       `json:"player_count" gorm:"default:0"`
	CurrentRound int       `json:"current_round" gorm:"default:0"`
	CurrentMatchID *uint   `json:"current_match_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// TournamentPlayer — participant in a tournament
type TournamentPlayer struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	TournamentID  uint      `json:"tournament_id" gorm:"index"`
	ParticipantID uint      `json:"participant_id" gorm:"index"`
	Name          string    `json:"name" gorm:"size:100"`
	Seed          int       `json:"seed" gorm:"default:0"`
	Rank          int       `json:"rank" gorm:"default:0"`
	CreatedAt     time.Time `json:"created_at"`
}

// TournamentMatch — single elimination bracket node
type TournamentMatch struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	TournamentID uint       `json:"tournament_id" gorm:"index"`
	Round        int        `json:"round"` // 1=Round1, 2=Quarterfinal, 3=Semifinal, 4=Final
	MatchNumber  int        `json:"match_number"`
	Player1ID    *uint      `json:"player1_id"`
	Player2ID    *uint      `json:"player2_id"`
	Player1Name  string     `json:"player1_name" gorm:"size:100"`
	Player2Name  string     `json:"player2_name" gorm:"size:100"`
	Player1Score float64    `json:"player1_score"`
	Player2Score float64    `json:"player2_score"`
	WinnerID     *uint      `json:"winner_id"`
	Status       string     `json:"status" gorm:"size:20;default:'scheduled'"` // scheduled | ready | playing | finished | bye
	ParentMatchID *uint     `json:"parent_match_id"`
	ParentSlot   int        `json:"parent_slot"` // 1 or 2
	RoomID       string     `json:"room_id" gorm:"size:4"`
	CreatedAt    time.Time  `json:"created_at"`
}
