package controller

import (
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite DB, migrates models, and overrides config.DB.
// Call defer cleanupTestDB() in the test function.
func setupTestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Migrate in order: EventBatch first (FKs depend on it)
	if err := db.AutoMigrate(&model.EventBatch{}); err != nil {
		t.Fatalf("AutoMigrate EventBatch failed: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Participant{},
		&model.GameSession{},
		&model.Room{},
		&model.PlayerState{},
		&model.GameResult{},
	); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Tournament{},
		&model.TournamentPlayer{},
		&model.TournamentMatch{},
	); err != nil {
		t.Fatalf("AutoMigrate tournament models failed: %v", err)
	}

	// Default batch
	var count int64
	db.Model(&model.EventBatch{}).Count(&count)
	if count == 0 {
		db.Create(&model.EventBatch{Name: "Test Batch", IsActive: true})
	}

	config.DB = db
}

// cleanupTestDB nils out config.DB to prevent test leakage.
func cleanupTestDB() {
	config.DB = nil
}
