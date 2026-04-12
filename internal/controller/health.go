package controller

import (
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func HealthCheck(c *gin.Context) {
	sqlDB, err := config.DB.DB()
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Database connection error")
		return
	}

	if err := sqlDB.Ping(); err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Database unreachable")
		return
	}

	response.OK(c, gin.H{"status": "healthy", "database": "connected"})
}
