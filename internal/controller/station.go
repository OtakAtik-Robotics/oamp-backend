package controller

import (
	"time"

	"oamp-backend/internal/config"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type StationInfo struct {
	PlayerName string    `json:"player_name"`
	RoomID     string    `json:"room_id"`
	Level      int       `json:"level"`
	Status     string    `json:"status"`
	LastSeen   time.Time `json:"last_seen"`
	Mode       string    `json:"mode"`
}

func GetStations(c *gin.Context) {
	cutoff := time.Now().Add(-90 * time.Second)
	var stations []StationInfo
	seenNames := make(map[string]bool)

	rows, err := config.DB.Raw(`
		SELECT ps.player_name, ps.room_id, ps.current_level, ps.is_finished, ps.updated_at
		FROM player_states ps
		WHERE ps.updated_at > ?
		ORDER BY ps.updated_at DESC
	`, cutoff).Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s StationInfo
			var isFinished bool
			rows.Scan(&s.PlayerName, &s.RoomID, &s.Level, &isFinished, &s.LastSeen)
			s.Mode = "competition"
			if isFinished {
				s.Status = "idle"
			} else {
				s.Status = "playing"
			}
			stations = append(stations, s)
			seenNames[s.PlayerName] = true
		}
	}

	rows2, err2 := config.DB.Raw(`
		SELECT p.name, gs.level_reached, gs.mode, gs.created_at
		FROM game_sessions gs
		JOIN participants p ON p.id = gs.participant_id
		WHERE gs.created_at > ?
		ORDER BY gs.created_at DESC
	`, cutoff).Rows()
	if err2 == nil {
		defer rows2.Close()
		for rows2.Next() {
			var name, mode string
			var level int
			var lastSeen time.Time
			rows2.Scan(&name, &level, &mode, &lastSeen)
			if seenNames[name] {
				continue
			}
			seenNames[name] = true
			stations = append(stations, StationInfo{
				PlayerName: name,
				RoomID:     "",
				Level:      level,
				Status:     "idle",
				LastSeen:   lastSeen,
				Mode:       mode,
			})
		}
	}

	if stations == nil {
		stations = []StationInfo{}
	}
	response.OKWithMessage(c, "Stations fetched", stations)
}
