package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
)

type Engine struct {
	cloudURL string
	apiKey   string
	interval time.Duration
	client   *http.Client
}

func New() *Engine {
	url := os.Getenv("CLOUD_SYNC_URL")
	if url == "" {
		return nil
	}
	interval := 60
	if v := os.Getenv("CLOUD_SYNC_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			interval = i
		}
	}
	return &Engine{
		cloudURL: url,
		apiKey:   os.Getenv("CLOUD_API_KEY"),
		interval: time.Duration(interval) * time.Second,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *Engine) syncTable(name string, modelType interface{}, idsCol string, ids interface{}) {
	data, err := json.Marshal(modelType)
	if err != nil {
		log.Printf("[sync] %s marshal error: %v", name, err)
		return
	}

	req, err := http.NewRequest("POST", e.cloudURL+"/api/v1/sync/"+name, bytes.NewReader(data))
	if err != nil {
		log.Printf("[sync] %s request error: %v", name, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("X-Sync-Key", e.apiKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		log.Printf("[sync] %s http error: %v", name, err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("[sync] %s rejected (status %d)", name, resp.StatusCode)
		return
	}
	log.Printf("[sync] %s synced successfully", name)

	// Mark as synced
	if idsCol != "" && ids != nil {
		config.DB.Model(modelType).Where(idsCol+" IN ?", ids).Update("synced_at", time.Now())
	}
}

func (e *Engine) syncParticipants() {
	var rows []model.Participant
	config.DB.Where("synced_at IS NULL").Limit(100).Find(&rows)
	if len(rows) == 0 {
		return
	}
	ids := make([]uint, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	e.syncTable("participants", rows, "id", ids)
}

func (e *Engine) syncGameSessions() {
	var rows []model.GameSession
	config.DB.Where("synced_at IS NULL").Limit(100).Find(&rows)
	if len(rows) == 0 {
		return
	}
	ids := make([]uint, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	e.syncTable("game-sessions", rows, "id", ids)
}

func (e *Engine) syncGameResults() {
	var rows []model.GameResult
	config.DB.Where("synced_at IS NULL").Limit(100).Find(&rows)
	if len(rows) == 0 {
		return
	}
	uids := make([]string, len(rows))
	for i := range rows {
		uids[i] = rows[i].UID
	}
	data, _ := json.Marshal(rows)
	req, _ := http.NewRequest("POST", e.cloudURL+"/api/v1/sync/game-results", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("X-Sync-Key", e.apiKey)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		log.Printf("[sync] game-results http error: %v", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		log.Printf("[sync] game-results rejected (status %d)", resp.StatusCode)
		return
	}
	config.DB.Model(&model.GameResult{}).Where("uid IN ?", uids).Update("synced_at", time.Now())
	log.Printf("[sync] game-results: %d rows synced", len(rows))
}

func (e *Engine) Start(ctx context.Context) {
	log.Printf("[sync] engine started → %s (every %v)", e.cloudURL, e.interval)

	// Sync immediately on startup
	e.syncParticipants()
	e.syncGameSessions()
	e.syncGameResults()

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.syncParticipants()
			e.syncGameSessions()
			e.syncGameResults()
		case <-ctx.Done():
			log.Println("[sync] engine stopped")
			return
		}
	}
}
