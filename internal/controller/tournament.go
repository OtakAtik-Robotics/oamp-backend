package controller

import (
	"fmt"
	"math"
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// ---------- Public structs ----------

type CreateTournamentPayload struct {
	Name       string `json:"name" binding:"required"`
	MaxPlayers int    `json:"max_players" binding:"required,gte=2,lte=64"`
	EventBatchID uint `json:"event_batch_id"`
}

type JoinTournamentPayload struct {
	UID string `json:"uid" binding:"required"`
}

type SubmitResultPayload struct {
	Player1Score float64 `json:"player1_score"`
	Player2Score float64 `json:"player2_score"`
	WinnerID     uint    `json:"winner_id" binding:"required"`
}

// ---------- Helpers ----------

func nextPowerOf2(n int) int {
	if n <= 1 { return 1 }
	return int(math.Pow(2, math.Ceil(math.Log2(float64(n)))))
}

func roundName(round, totalRounds int) string {
	diff := totalRounds - round
	switch diff {
	case 0: return "Final"
	case 1: return "Semifinal"
	case 2: return "Quarterfinal"
	case 3: return "Round of 16"
	case 4: return "Round of 32"
	default: return fmt.Sprintf("Round %d", round)
	}
}

// ---------- Controllers ----------

// GetTournaments — GET /api/v1/tournaments
func GetTournaments(c *gin.Context) {
	var tournaments []model.Tournament
	if err := config.DB.Order("created_at desc").Find(&tournaments).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch tournaments")
		return
	}
	response.OKWithMessage(c, "Tournaments fetched", tournaments)
}

// CreateTournament — POST /api/v1/tournaments
func CreateTournament(c *gin.Context) {
	var req CreateTournamentPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}
	batchID := req.EventBatchID
	if batchID == 0 { batchID = 1 }
	t := model.Tournament{
		Name:         req.Name,
		MaxPlayers:   req.MaxPlayers,
		EventBatchID: batchID,
		Status:       "registration",
	}
	if err := config.DB.Create(&t).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create tournament")
		return
	}
	response.CreatedWithMessage(c, "Tournament created", t)
}

// JoinTournament — POST /api/v1/tournaments/:id/join
func JoinTournament(c *gin.Context) {
	idStr := c.Param("id")
	var tournament model.Tournament
	if err := config.DB.First(&tournament, idStr).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tournament not found")
		return
	}
	if tournament.Status != "registration" {
		response.Error(c, http.StatusConflict, "Registration closed")
		return
	}

	var req JoinTournamentPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "uid required")
		return
	}

	// Verify participant exists
	var participant model.Participant
	if err := config.DB.Where("uid = ?", req.UID).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	// Check not already joined
	var existing model.TournamentPlayer
	if err := config.DB.Where("tournament_id = ? AND participant_id = ?", tournament.ID, participant.ID).First(&existing).Error; err == nil {
		response.Error(c, http.StatusConflict, "Already joined")
		return
	}

	// Check capacity
	if tournament.PlayerCount >= tournament.MaxPlayers {
		response.Error(c, http.StatusConflict, "Tournament full")
		return
	}

	tp := model.TournamentPlayer{
		TournamentID:  tournament.ID,
		ParticipantID: participant.ID,
		Name:          participant.Name,
	}
	if err := config.DB.Create(&tp).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to join")
		return
	}

	config.DB.Model(&tournament).Update("player_count", tournament.PlayerCount+1)
	response.OKWithMessage(c, "Joined successfully", tp)
}

// StartTournament — POST /api/v1/tournaments/:id/start
func StartTournament(c *gin.Context) {
	idStr := c.Param("id")
	var tournament model.Tournament
	if err := config.DB.First(&tournament, idStr).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tournament not found")
		return
	}
	if tournament.Status != "registration" {
		response.Error(c, http.StatusConflict, "Already started")
		return
	}

	var players []model.TournamentPlayer
	config.DB.Where("tournament_id = ?", tournament.ID).Order("created_at asc").Find(&players)
	N := len(players)
	if N < 2 {
		response.Error(c, http.StatusBadRequest, "Need at least 2 players")
		return
	}

	// Randomize seeds
	for i := range players {
		players[i].Seed = i + 1
		config.DB.Save(&players[i])
	}

	// Generate bracket
	P := nextPowerOf2(N)
	totalRounds := int(math.Log2(float64(P)))

	// Build round sizes bottom-up
	roundSizes := make([]int, totalRounds)
	for r := 0; r < totalRounds; r++ {
		roundSizes[r] = P / int(math.Pow(2, float64(r+1)))
	}

	// Create matches round by round (Round 1 first, then 2, ...)
	var matches []model.TournamentMatch
	matchIDOffset := make(map[int]uint) // global match ID → sequential

	var currentID uint = 0
	for r := 0; r < totalRounds; r++ {
		for m := 0; m < roundSizes[r]; m++ {
			status := "scheduled"
			var p1ID, p2ID *uint
			var p1Name, p2Name string

			// Assign players to Round 1 matches
			if r == 0 {
				idx1 := m * 2
				idx2 := m*2 + 1
				if idx1 < N {
					p1ID = &players[idx1].ParticipantID
					p1Name = players[idx1].Name
				}
				if idx2 < N {
					p2ID = &players[idx2].ParticipantID
					p2Name = players[idx2].Name
				}
				// BYE handling
				if p1ID != nil && p2ID == nil {
					status = "bye"
					matches = append(matches, model.TournamentMatch{
						TournamentID: tournament.ID,
						Round:        1,
						MatchNumber:  m + 1,
						Player1ID:    p1ID,
						Player1Name:  p1Name,
						WinnerID:     p1ID,
						Status:       "bye",
					})
					continue
				}
				if p1ID == nil && p2ID != nil {
					status = "bye"
					matches = append(matches, model.TournamentMatch{
						TournamentID: tournament.ID,
						Round:        1,
						MatchNumber:  m + 1,
						Player2ID:    p2ID,
						Player2Name:  p2Name,
						WinnerID:     p2ID,
						Status:       "bye",
					})
					continue
				}
			}

			matches = append(matches, model.TournamentMatch{
				TournamentID: tournament.ID,
				Round:        r + 1,
				MatchNumber:  m + 1,
				Player1ID:    p1ID,
				Player2ID:    p2ID,
				Player1Name:  p1Name,
				Player2Name:  p2Name,
				Status:       status,
			})
			matchIDOffset[int(currentID)] = uint(len(matches)) // will be filled after create
			currentID++
		}
	}

	// Save all matches
	for i := range matches {
		config.DB.Create(&matches[i])
	}

	// Link parent matches
	// For each round r (0-based), match m in that round → parent in round r+1, match floor(m/2)
	var allMatches []model.TournamentMatch
	config.DB.Where("tournament_id = ?", tournament.ID).Order("id asc").Find(&allMatches)

	// Map: (round, matchNumber) → match ID
	matchKey := make(map[string]uint)
	for _, m := range allMatches {
		key := fmt.Sprintf("%d-%d", m.Round, m.MatchNumber)
		matchKey[key] = m.ID
	}

	for r := 1; r <= totalRounds; r++ {
		matchCountInRound := P / int(math.Pow(2, float64(r)))
		for m := 1; m <= matchCountInRound; m++ {
			parentKey := fmt.Sprintf("%d-%d", r, m)
			parentID := matchKey[parentKey]
			// Children are in previous round
			child1Key := fmt.Sprintf("%d-%d", r-1, m*2-1)
			child2Key := fmt.Sprintf("%d-%d", r-1, m*2)
			if child1ID, ok := matchKey[child1Key]; ok {
				config.DB.Model(&model.TournamentMatch{}).Where("id = ?", child1ID).Update("parent_match_id", parentID)
				config.DB.Model(&model.TournamentMatch{}).Where("id = ?", child1ID).Update("parent_slot", 1)
			}
			if child2ID, ok := matchKey[child2Key]; ok {
				config.DB.Model(&model.TournamentMatch{}).Where("id = ?", child2ID).Update("parent_match_id", parentID)
				config.DB.Model(&model.TournamentMatch{}).Where("id = ?", child2ID).Update("parent_slot", 2)
			}
		}
	}

	// Auto-assign unique room codes to each Round 1 scheduled match
	var round1Matches []model.TournamentMatch
	config.DB.Where("tournament_id = ? AND round = ? AND status = ?", tournament.ID, 1, "scheduled").Find(&round1Matches)
	for i := range round1Matches {
		round1Matches[i].RoomID = generateRoomCode()
		config.DB.Save(&round1Matches[i])
	}

	// Find first non-bye match as current
	var firstMatch model.TournamentMatch
	config.DB.Where("tournament_id = ? AND status = ?", tournament.ID, "scheduled").Order("round asc, match_number asc").First(&firstMatch)
	if firstMatch.ID != 0 {
		config.DB.Model(&tournament).Updates(map[string]interface{}{
			"status":             "in_progress",
			"current_round":      1,
			"current_match_id":   firstMatch.ID,
		})
		config.DB.Model(&firstMatch).Update("status", "ready")
	} else {
		config.DB.Model(&tournament).Update("status", "finished")
	}

	response.OKWithMessage(c, "Tournament started", tournament)
}

// GetTournament — GET /api/v1/tournaments/:id
func GetTournament(c *gin.Context) {
	idStr := c.Param("id")
	var tournament model.Tournament
	if err := config.DB.First(&tournament, idStr).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tournament not found")
		return
	}

	var matches []model.TournamentMatch
	config.DB.Where("tournament_id = ?", tournament.ID).Order("round asc, match_number asc").Find(&matches)

	var players []model.TournamentPlayer
	config.DB.Where("tournament_id = ?", tournament.ID).Find(&players)

	response.OKWithMessage(c, "Tournament fetched", gin.H{
		"tournament": tournament,
		"matches":    matches,
		"players":    players,
	})
}

// SubmitMatchResult — POST /api/v1/tournaments/:id/matches/:mid/result
func SubmitMatchResult(c *gin.Context) {
	var tournament model.Tournament
	if err := config.DB.First(&tournament, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tournament not found")
		return
	}
	mid := c.Param("mid")
	var match model.TournamentMatch
	if err := config.DB.First(&match, mid).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Match not found")
		return
	}
	if match.TournamentID != tournament.ID {
		response.Error(c, http.StatusBadRequest, "Match does not belong to tournament")
		return
	}
	if match.Status == "finished" || match.Status == "bye" {
		response.Error(c, http.StatusConflict, "Match already decided")
		return
	}

	var req SubmitResultPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	updates := map[string]interface{}{
		"player1_score": req.Player1Score,
		"player2_score": req.Player2Score,
		"winner_id":     req.WinnerID,
		"status":        "finished",
	}
	config.DB.Model(&match).Updates(updates)

	advanceWinner(tournament.ID, &match, req.WinnerID)

	response.OKWithMessage(c, "Result submitted", match)
}

// advanceWinner promotes the winner to the parent match and triggers next match readiness
func advanceWinner(tournamentID uint, match *model.TournamentMatch, winnerID uint) {
	// Advance winner to parent match
	if match.ParentMatchID != nil && match.ParentSlot > 0 {
		var parent model.TournamentMatch
		if err := config.DB.First(&parent, *match.ParentMatchID).Error; err == nil {
			winnerName := match.Player1Name
			if match.WinnerID != nil && match.Player2ID != nil && *match.WinnerID == *match.Player2ID {
				winnerName = match.Player2Name
			}
			if match.ParentSlot == 1 {
				config.DB.Model(&parent).Updates(map[string]interface{}{
					"player1_id":    winnerID,
					"player1_name":  winnerName,
				})
			} else {
				config.DB.Model(&parent).Updates(map[string]interface{}{
					"player2_id":    winnerID,
					"player2_name":  winnerName,
				})
			}
		}
	}

	// Find next scheduled match in tournament
	var tournament model.Tournament
	config.DB.First(&tournament, tournamentID)

	var nextMatch model.TournamentMatch
	config.DB.Where("tournament_id = ? AND status = ?", tournamentID, "scheduled").Order("round asc, match_number asc").First(&nextMatch)
	if nextMatch.ID != 0 {
		config.DB.Model(&nextMatch).Update("status", "ready")
		config.DB.Model(&tournament).Update("current_match_id", nextMatch.ID)
		config.DB.Model(&tournament).Update("current_round", nextMatch.Round)
	} else {
		// Check if tournament finished
		var unfinished int64
		config.DB.Model(&model.TournamentMatch{}).Where("tournament_id = ? AND status IN ?", tournamentID, []string{"scheduled", "ready", "playing"}).Count(&unfinished)
		if unfinished == 0 {
			config.DB.Model(&tournament).Updates(map[string]interface{}{
				"status":           "finished",
				"current_match_id": nil,
			})
			// Set ranks
			setTournamentRanks(tournamentID, match.WinnerID)
		}
	}
}

func setTournamentRanks(tournamentID uint, winnerID *uint) {
	if winnerID == nil { return }
	// Winner rank 1
	config.DB.Model(&model.TournamentPlayer{}).Where("tournament_id = ? AND participant_id = ?", tournamentID, *winnerID).Update("rank", 1)
	// Simple: others get rank by how far they advanced (reverse order of elimination)
	// For now just leave rank 0 for others
}

// GetCurrentMatch — GET /api/v1/tournaments/:id/current-match
func GetCurrentMatch(c *gin.Context) {
	var tournament model.Tournament
	if err := config.DB.First(&tournament, c.Param("id")).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tournament not found")
		return
	}
	if tournament.CurrentMatchID == nil {
		response.OKWithMessage(c, "No active match", nil)
		return
	}
	var match model.TournamentMatch
	if err := config.DB.First(&match, *tournament.CurrentMatchID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Match not found")
		return
	}
	response.OKWithMessage(c, "Current match", match)
}

// RegisterTournamentPlayers — POST /api/v1/tournaments/:id/register
// Admin bulk-registers multiple UIDs into a tournament (replaces self-join flow)
type RegisterPlayersPayload struct {
	UIDs []string `json:"uids" binding:"required,min=2,max=64"`
}

func RegisterTournamentPlayers(c *gin.Context) {
	idStr := c.Param("id")
	var tournament model.Tournament
	if err := config.DB.First(&tournament, idStr).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tournament not found")
		return
	}
	if tournament.Status != "registration" {
		response.Error(c, http.StatusConflict, "Registration closed")
		return
	}

	var req RegisterPlayersPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	var added int
	var errors []string
	for _, uid := range req.UIDs {
		var participant model.Participant
		if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
			errors = append(errors, fmt.Sprintf("UID %s not found", uid))
			continue
		}
		var existing model.TournamentPlayer
		if err := config.DB.Where("tournament_id = ? AND participant_id = ?", tournament.ID, participant.ID).First(&existing).Error; err == nil {
			errors = append(errors, fmt.Sprintf("UID %s already joined", uid))
			continue
		}
		if tournament.PlayerCount+added >= tournament.MaxPlayers {
			errors = append(errors, fmt.Sprintf("UID %s skipped (tournament full)", uid))
			continue
		}
		config.DB.Create(&model.TournamentPlayer{
			TournamentID:  tournament.ID,
			ParticipantID: participant.ID,
			Name:          participant.Name,
		})
		added++
	}

	config.DB.Model(&tournament).Update("player_count", tournament.PlayerCount+added)
	response.OKWithMessage(c, fmt.Sprintf("Registered %d players", added), gin.H{
		"added":  added,
		"errors": errors,
	})
}

// GetActiveMatchByUID — GET /api/v1/tournaments/active-match/:uid
// Game client calls this after scanning UID to see if player has an active cup match
func GetActiveMatchByUID(c *gin.Context) {
	uid := c.Param("uid")

	var participant model.Participant
	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	// Find an in-progress tournament where this player is registered
	var tp model.TournamentPlayer
	if err := config.DB.Where("participant_id = ?", participant.ID).First(&tp).Error; err != nil {
		response.OKWithMessage(c, "No active tournament", gin.H{"has_match": false})
		return
	}

	var tournament model.Tournament
	if err := config.DB.First(&tournament, tp.TournamentID).Error; err != nil {
		response.OKWithMessage(c, "No active tournament", gin.H{"has_match": false})
		return
	}
	if tournament.Status != "in_progress" {
		response.OKWithMessage(c, "Tournament not in progress", gin.H{"has_match": false, "status": tournament.Status})
		return
	}

	// Find the next scheduled/ready match where this player participates
	var matches []model.TournamentMatch
	config.DB.Where("tournament_id = ? AND status IN ?", tournament.ID, []string{"scheduled", "ready", "playing"}).
		Order("round asc, match_number asc").Find(&matches)

	for _, m := range matches {
		isP1 := m.Player1ID != nil && *m.Player1ID == participant.ID
		isP2 := m.Player2ID != nil && *m.Player2ID == participant.ID
		if isP1 || isP2 {
			opponent := m.Player1Name
			if isP1 {
				opponent = m.Player2Name
			}
			response.OKWithMessage(c, "Active match found", gin.H{
				"has_match":     true,
				"tournament_id": tournament.ID,
				"tournament_name": tournament.Name,
				"match":         m,
				"room_code":     m.RoomID,
				"opponent":      opponent,
				"is_player1":    isP1,
			})
			return
		}
	}

	response.OKWithMessage(c, "No upcoming match", gin.H{"has_match": false})
}

// CreateMatchRoom — POST /api/v1/tournaments/:id/matches/:mid/create-room
// Admin (or auto-start) assigns a room code to a match
func CreateMatchRoom(c *gin.Context) {
	var match model.TournamentMatch
	if err := config.DB.First(&match, c.Param("mid")).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Match not found")
		return
	}
	if match.RoomID != "" {
		response.OKWithMessage(c, "Room already assigned", match)
		return
	}
	roomCode := generateRoomCode()
	config.DB.Model(&match).Update("room_id", roomCode)
	match.RoomID = roomCode
	response.OKWithMessage(c, "Room assigned", match)
}

// HandleTournamentEvent — POST /api/tournaments/event (called by desktop game client)
type TournamentEventPayload struct {
	RoomID    string  `json:"room_id" binding:"required"`
	EventType string  `json:"event_type" binding:"required,oneof=match_started match_finished"`
	PlayerNum int     `json:"player_num" binding:"omitempty,oneof=1 2"` // 1 or 2 — which player is reporting
	Score     float64 `json:"score"`
}

func HandleTournamentEvent(c *gin.Context) {
	var req TournamentEventPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	// Find match by room_id
	var match model.TournamentMatch
	if err := config.DB.Where("room_id = ?", req.RoomID).First(&match).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Match not found for this room")
		return
	}

	var tournament model.Tournament
	if err := config.DB.First(&tournament, match.TournamentID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Tournament not found")
		return
	}

	switch req.EventType {
	case "match_started":
		if match.Status == "scheduled" || match.Status == "ready" {
			config.DB.Model(&match).Update("status", "playing")
		}
		response.OKWithMessage(c, "Match marked as playing", match)

	case "match_finished":
		if match.Status == "finished" || match.Status == "bye" {
			response.Error(c, http.StatusConflict, "Match already decided")
			return
		}
		if req.PlayerNum != 1 && req.PlayerNum != 2 {
			response.Error(c, http.StatusBadRequest, "player_num must be 1 or 2")
			return
		}

		// Update only the submitting player's score
		scoreField := "player1_score"
		if req.PlayerNum == 2 {
			scoreField = "player2_score"
		}
		config.DB.Model(&match).Update(scoreField, req.Score)

		// Refresh match to see both scores
		config.DB.First(&match, match.ID)

		p1Score := match.Player1Score
		p2Score := match.Player2Score

		// If either score is still zero, wait for the other player
		if p1Score == 0 || p2Score == 0 {
			response.OKWithMessage(c, "Score recorded — waiting for opponent", gin.H{
				"match_status":    "waiting",
				"player1_score":   p1Score,
				"player2_score":   p2Score,
			})
			return
		}

		// Both scores present — determine winner
		var winnerID uint
		if p1Score > p2Score {
			if match.Player1ID == nil {
				response.Error(c, http.StatusBadRequest, "Player 1 not assigned")
				return
			}
			winnerID = *match.Player1ID
		} else if p2Score > p1Score {
			if match.Player2ID == nil {
				response.Error(c, http.StatusBadRequest, "Player 2 not assigned")
				return
			}
			winnerID = *match.Player2ID
		} else {
			// Tie — default to Player 1 for single elimination
			if match.Player1ID == nil {
				response.Error(c, http.StatusBadRequest, "Player 1 not assigned")
				return
			}
			winnerID = *match.Player1ID
		}

		config.DB.Model(&match).Updates(map[string]interface{}{
			"winner_id": winnerID,
			"status":    "finished",
		})

		// Refresh match from DB to get latest values for advancing
		config.DB.First(&match, match.ID)
		advanceWinner(tournament.ID, &match, winnerID)

		response.OKWithMessage(c, "Match finished and advanced", match)
	}
}

// DeleteTournament — DELETE /api/v1/tournaments/:id
func DeleteTournament(c *gin.Context) {
	idStr := c.Param("id")
	var tournament model.Tournament
	if err := config.DB.First(&tournament, idStr).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Tournament not found")
		return
	}
	// Cascade delete
	config.DB.Where("tournament_id = ?", tournament.ID).Delete(&model.TournamentMatch{})
	config.DB.Where("tournament_id = ?", tournament.ID).Delete(&model.TournamentPlayer{})
	config.DB.Delete(&tournament)
	response.OKWithMessage(c, "Tournament deleted", nil)
}
