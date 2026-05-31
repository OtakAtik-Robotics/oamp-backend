package route

import (
	"os"
	"strings"
	"oamp-backend/internal/controller"
	"oamp-backend/internal/middleware"
	"oamp-backend/internal/websocket"

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

	// WebSocket — outside API group (no body limit, no rate limit)
	wsManager := websocket.NewManager()
	r.GET("/ws/match/:room_id", websocket.HandleWebSocket(wsManager))

	// Game client compatibility routes (mirrors web-server-api, no v1 prefix)
	r.POST("/api/game/event", controller.GameEvent)
	r.POST("/api/game/submit", controller.SubmitGameResult)
	r.GET("/api/participants/uid/:uid", controller.GetParticipantByUID)
	r.GET("/api/rooms", controller.GetRooms)
	r.POST("/api/rooms", controller.CreateRoom)
	r.GET("/api/rooms/:code", controller.GetRoom)
	r.POST("/api/rooms/:code/join", controller.JoinRoom)
	r.POST("/api/rooms/:code/leave", controller.LeaveRoom)
	r.POST("/api/rooms/:code/ready", controller.SetReady)

	// Tournament active-match lookup for desktop client (compat, no v1 prefix)
	r.GET("/api/tournaments/active-match/:uid", controller.GetActiveMatchByUID)
	// Tournament event from desktop client (match_started, match_finished)
	r.POST("/api/tournaments/event", controller.HandleTournamentEvent)

	api := r.Group("/api/v1")
	{
		// Participant registration
		api.POST("/participants", controller.RegisterParticipant)
		api.GET("/participants", controller.GetParticipants)
		api.GET("/participants/stats", controller.GetParticipantsWithScores)
		api.GET("/participants/id/:id", controller.GetParticipantByID)
		api.DELETE("/participants/all", controller.DeleteAllParticipants)
		api.DELETE("/participants/:id", controller.DeleteParticipant)
		api.GET("/participants/lookup/:nickname", controller.LookupParticipant)
		api.GET("/participants/uid/:uid/sessions", controller.GetParticipantSessions)

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
		api.PUT("/batches/:id", controller.RenameBatch)
		api.DELETE("/batches/:id", controller.DeleteBatch)
		api.POST("/batches/:id/activate", controller.ActivateBatch)

		// Payment (Midtrans Snap)
		api.POST("/payment/checkout/:uid", controller.Checkout)
		api.POST("/payment/webhook", controller.PaymentWebhook)
		api.POST("/payment/simulate-success/:uid", controller.SimulatePaymentSuccess)

		// Game result from desktop client (bracelet UID scan)
		api.POST("/game/submit", controller.SubmitGameResult)
		api.GET("/participants/uid/:uid", controller.GetParticipantByUID)

		// 1v1 Match rooms (database-backed)
		api.GET("/rooms", controller.GetRooms)
		api.POST("/rooms", controller.CreateRoom)
		api.GET("/rooms/:code", controller.GetRoom)
		api.POST("/rooms/:code/join", controller.JoinRoom)
		api.POST("/rooms/:code/leave", controller.LeaveRoom)
		api.POST("/rooms/:code/ready", controller.SetReady)

		// Game event (desktop app — join_room, level_start, level_complete, leave_room)
		api.POST("/game/event", controller.GameEvent)

		// Participant analysis (AI Health Consultant, premium-gated)
		api.GET("/participants/analysis/:uid", controller.GetParticipantAnalysis)

		// Tournaments — single elimination cup
		api.GET("/tournaments", controller.GetTournaments)
		api.POST("/tournaments", controller.CreateTournament)
		api.GET("/tournaments/:id", controller.GetTournament)
		api.DELETE("/tournaments/:id", controller.DeleteTournament)
		api.POST("/tournaments/:id/join", controller.JoinTournament)
		api.POST("/tournaments/:id/register", controller.RegisterTournamentPlayers)
		api.POST("/tournaments/:id/start", controller.StartTournament)
		api.GET("/tournaments/:id/current-match", controller.GetCurrentMatch)
		api.POST("/tournaments/:id/matches/:mid/create-room", controller.CreateMatchRoom)
		api.POST("/tournaments/:id/matches/:mid/result", controller.SubmitMatchResult)

		// Game client — check active cup match by UID
		api.GET("/tournaments/active-match/:uid", controller.GetActiveMatchByUID)
	}
}
