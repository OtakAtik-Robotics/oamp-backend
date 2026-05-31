package config

import (
	"fmt"
	"log"
	"oamp-backend/internal/model"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	// Run schema migrations using golang-migrate
	runMigrations(dsn)

	// AutoMigrate as safety net for dev (skips if tables already exist)
	// In production, disable by setting DISABLE_AUTO_MIGRATE=true
	if os.Getenv("DISABLE_AUTO_MIGRATE") != "true" {
		if err := db.AutoMigrate(&model.EventBatch{}); err != nil {
			log.Fatal("AutoMigrate failed: ", err)
		}
		if err := db.AutoMigrate(
			&model.Participant{},
			&model.GameSession{},
			&model.Room{},
			&model.PlayerState{},
			&model.GameResult{},
		); err != nil {
			log.Fatal("AutoMigrate failed: ", err)
		}
		if err := db.AutoMigrate(
			&model.Tournament{},
			&model.TournamentPlayer{},
			&model.TournamentMatch{},
		); err != nil {
			log.Fatal("AutoMigrate failed: ", err)
		}
	}

	// Create default batch if none exists
	var count int64
	db.Model(&model.EventBatch{}).Count(&count)
	if count == 0 {
		db.Create(&model.EventBatch{Name: "Sesi Default", IsActive: true})
	}

	// Update existing game_sessions to use default batch (id=1) if needed
	db.Model(&model.GameSession{}).Where("event_batch_id = 0").Update("event_batch_id", 1)

	// Raw SQL migrations for columns AutoMigrate may miss
	db.Exec(`ALTER TABLE game_sessions ADD COLUMN IF NOT EXISTS score DOUBLE PRECISION DEFAULT 0`)
	db.Exec(`ALTER TABLE participants ADD COLUMN IF NOT EXISTS spo2 DOUBLE PRECISION`)

	DB = db

	// Connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get underlying sql.DB: ", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	fmt.Println("Database Connected and Migrated successfully")
}

func runMigrations(dsn string) {
	m, err := migrate.New(
		"file://migrations",
		fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		),
	)
	if err != nil {
		log.Printf("Migration init failed (skipping): %v", err)
		return
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Printf("Migration up failed (skipping): %v", err)
		return
	}
	fmt.Println("Schema migrations applied successfully")
}
