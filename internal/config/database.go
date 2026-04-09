package config

import (
	"bdt-server/internal/model"
	"fmt"
	"log"
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

	err = db.AutoMigrate(
		&model.Participant{},
		&model.GameSession{},
		&model.FaceExpressionLog{},
		&model.DatasetCapture{},
		&model.QuizResult{},
	)
	if err != nil {
		log.Fatal("Database migration failed: ", err)
	}

	DB = db
	fmt.Println("Database Connected and Migrated successfully")
}