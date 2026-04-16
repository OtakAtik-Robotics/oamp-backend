package config

import (
	"fmt"
	"log"
	"oamp-backend/internal/model"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"), os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}

	// Migrate EventBatch first (no FK dependencies)
	if err := db.AutoMigrate(&model.EventBatch{}); err != nil {
		log.Fatal("Database migration failed: ", err)
	}

	// Create default batch if none exists
	var count int64
	db.Model(&model.EventBatch{}).Count(&count)
	if count == 0 {
		db.Create(&model.EventBatch{Name: "Sesi Default", IsActive: true})
	}

	// Migrate other models
	if err := db.AutoMigrate(
		&model.Participant{},
		&model.GameSession{},
		&model.FaceExpressionLog{},
		&model.DatasetCapture{},
		&model.QuizResult{},
	); err != nil {
		log.Fatal("Database migration failed: ", err)
	}

	// Update existing game_sessions to use default batch (id=1) if needed
	db.Model(&model.GameSession{}).Where("event_batch_id = 0").Update("event_batch_id", 1)

	DB = db
	fmt.Println("Database Connected and Migrated successfully")
}