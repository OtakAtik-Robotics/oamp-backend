package route

import (
	"os"
	"strings"
	"oamp-backend/internal/controller"
	"oamp-backend/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Configure CORS from environment
	origins := os.Getenv("CORS_ORIGINS")
	allowAll := origins == "*" || origins == ""

	var corsConfig cors.Config
	if allowAll {
		corsConfig = cors.Config{
			AllowAllOrigins:  true,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
		}
	} else {
		originList := strings.Split(origins, ",")
		corsConfig = cors.Config{
			AllowOrigins:     originList,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
		}
	}
	r.Use(cors.New(corsConfig))
	r.Use(middleware.RateLimit())
	r.Use(middleware.BodyLimit(2 * 1024 * 1024)) // 2 MB max body size

	// Health check
	r.GET("/health", controller.HealthCheck)

	api := r.Group("/api/v1")
	{
		// Participant registration
		api.POST("/participants", controller.RegisterParticipant)

		// Robot endpoints
		robot := api.Group("/robot")
		{
			robot.GET("/auth/:uid", controller.RobotAuth)
			robot.POST("/sessions", controller.SubmitSession)
			robot.POST("/logs/face", controller.SubmitFaceLogs)
		}

		// Android app endpoints
		app := api.Group("/app")
		{
			app.GET("/auth/:uid", controller.AppAuth)
			app.POST("/quiz", controller.SubmitQuiz)
		}

		// Leaderboard
		api.GET("/leaderboard", controller.GetLeaderboard)
		api.GET("/leaderboard/timeline", controller.GetLeaderboardTimeline)

		// Export
		api.GET("/export/excel", controller.ExportExcel)
		api.GET("/export/pdf", controller.ExportPDF)
		api.GET("/export/rapor/:uid", controller.ExportRapor)

		// Event Batch management
		api.GET("/batches", controller.GetBatches)
		api.POST("/batches", controller.CreateBatch)

		// Payment (Midtrans Snap)
		api.POST("/payment/checkout/:uid", controller.Checkout)
		api.POST("/payment/webhook", controller.PaymentWebhook)
		api.POST("/payment/simulate-success/:uid", controller.SimulatePaymentSuccess)

		// Participant analysis (AI Health Consultant, premium-gated)
		api.GET("/participants/analysis/:uid", controller.GetParticipantAnalysis)
	}
}
