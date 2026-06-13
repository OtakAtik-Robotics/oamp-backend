package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oamp-backend/internal/config"
	"oamp-backend/internal/route"
	"oamp-backend/internal/sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	config.ConnectDB()

	// Start cloud sync engine (only if CLOUD_SYNC_URL is set)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if eng := sync.New(); eng != nil {
		go eng.Start(ctx)
	}

	r := gin.Default()
	route.SetupRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	sqlDB, err := config.DB.DB()
	if err == nil {
		sqlDB.Close()
	}

	log.Println("Server stopped")
}
