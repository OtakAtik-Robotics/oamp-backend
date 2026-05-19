package model

import (
	"testing"
)

func TestParticipant_IsPremium_DefaultFalse(t *testing.T) {
	p := Participant{}
	if p.IsPremium {
		t.Error("IsPremium should default to false")
	}
}

func TestParticipant_Fields(t *testing.T) {
	p := Participant{
		UID:          "BCR-001",
		Name:         "Test User",
		Age:          10,
		Grade:        "5",
		Gender:       "male",
		Height:       135.0,
		Weight:       30.0,
		HeartRate:    85,
		SpO2:         98.5,
		GripStrength: 12.3,
	}

	if p.UID != "BCR-001" {
		t.Errorf("expected UID BCR-001, got %s", p.UID)
	}
	if p.Age != 10 {
		t.Errorf("expected Age 10, got %d", p.Age)
	}
	if p.IsPremium {
		t.Error("IsPremium should be false by default")
	}
}

func TestGameSession_ParticipantBinding(t *testing.T) {
	s := GameSession{
		ParticipantID:   1,
		LevelReached:    5,
		VisuoSpatialFit: 0.85,
		DexterityScore:  75.0,
	}

	if s.ParticipantID != 1 {
		t.Errorf("expected ParticipantID 1, got %d", s.ParticipantID)
	}
	// Participant field should be zero-value (json:"-" binding:"-")
	if s.Participant.ID != 0 {
		t.Error("Participant embedded struct should be zero-value")
	}
}

func TestEventBatch_Defaults(t *testing.T) {
	b := EventBatch{Name: "Test Batch"}
	if b.IsActive {
		t.Error("IsActive should default to false")
	}
}

func TestQuizResult_Fields(t *testing.T) {
	q := QuizResult{
		ParticipantID: 1,
		Score:         85,
		AnswersData:   `{"q1":"A"}`,
	}
	if q.Score != 85 {
		t.Errorf("expected Score 85, got %d", q.Score)
	}
}
