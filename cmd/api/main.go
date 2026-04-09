package main

import (
	"log"
	"os"
	"server/internal/config"
	"server/internal/handler"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	config.ConnectDB()

	r := gin.Default()

	api := r.Group("/api/v1")
	{
		api.POST("/participants", handler.RegisterParticipant)
		api.GET("/participants/:uid", handler.GetParticipantByUID)
		api.POST("/sessions", handler.SubmitGameSession)

		api.GET("/mobile/results/:uid", handler.GetParticipantResult)
		api.POST("/mobile/quiz", handler.SubmitQuizResult)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}