# AGENTS.md

## Commands

```bash
go build -o bin/server ./cmd/api        # build
go run ./cmd/api                         # run (needs PostgreSQL + .env)
go test ./...                            # run all tests
go test -run TestFunctionName ./path/to/package  # single test
```

No Makefile. No `go vet` or lint step configured.

## Environment

Requires `.env` (or env vars): `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`, `PORT`.
Optional: `CORS_ORIGINS` (default `"*"`), `MIDTRANS_SERVER_KEY`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`, `AI_PROVIDER`, `AI_API_KEY`, `AI_MODEL`, `AI_BASE_URL`, `MINIMAX_GROUP_ID`.

See `.env.example` for full list.

## Architecture Gotchas

- **Dual route sets.** Two route groups serve the same rooms/game endpoints: `/api/game/...`/`/api/rooms/...` (no v1 prefix, for desktop client compat) and `/api/v1/game/...`/`/api/v1/rooms/...`. When adding room/game endpoints, register in **both** groups in `internal/route/route.go`.
- **Dual response styles.** v1 routes use `pkg/response` helpers (`OK()`, `Error()`, etc.) producing `{"status","message","data"}`. Compat routes (no v1 prefix) and `analysis.go` return raw `gin.H{"error":...}` or `gin.H{"ok":true}`. Match the convention of the route group you're editing.
- **Global DB.** `config.DB` is a package-level `*gorm.DB` singleton. Controllers are standalone functions that read it directly — no dependency injection.
- **`GameSession.Participant`** and **`GameSession.EventBatch`** use `json:"-" binding:"-"` to prevent leaking empty structs and triggering nested validation. Never remove these tags or bind the nested struct.
- **Schema migrations use golang-migrate.** Migration files live in `migrations/`. `database.go` runs `golang-migrate` on startup, then falls back to GORM `AutoMigrate` for dev. Set `DISABLE_AUTO_MIGRATE=true` in production.
- **AutoMigrate order matters.** `EventBatch` migrates first (other models have FKs to it). A default batch is created if table is empty. Adding new FK-dependent models requires updating migration files and `database.go` migration order.
- **Raw SQL migration.** `ALTER TABLE game_sessions ADD COLUMN IF NOT EXISTS score` runs after AutoMigrate. Column additions that GORM can't handle go here.
- **`GripStrength`** lives on `Participant` (measured at registration), not `GameSession`.
- **Leaderboard score formula:** `(level_reached * 10) + (visuo_spatial_fit * 50) + (dexterity_score * 0.2)`. Uses PostgreSQL `DISTINCT ON` for one entry per participant.
- **`GameResult` upserts by UID** via raw SQL `ON CONFLICT (uid) DO UPDATE`, not GORM's clause builder. One result per participant UID.
- **Competition mode in `SubmitGameResult`** does DELETE+INSERT for `game_sessions` (not true upsert). `DexterityScore` = `cognitive_age / real_age`, capped at 2.0.
- **WebSocket route bypasses middleware.** `/ws/match/:room_id` is registered outside the API group — no rate limit, no body limit.
- **WS rooms are in-memory only.** Destroyed on server restart. `Room` model in DB is separate (used by REST room endpoints).
- **Stale room cleanup** runs on every `GetRooms` call: waiting/ready rooms idle >5min are deleted, playing rooms idle >30min are marked finished.
- **`Float64Array`** is a custom `sql.Scanner`/`driver.Valuer` type for JSONB `[]float64`. Use it (or copy the pattern) for any new JSONB array fields on models.
- **AI analysis is cached.** `Participant.AiAnalysis` is populated once; subsequent calls return the cached value. AI failures return HTTP 200 with `status:"fallback"` (graceful degradation, never 500).
- **Room code charset** excludes confusable chars (I, O, 0, 1): `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`.
- **Midtrans logs suppressed.** `initMidtrans()` sets `midtrans.DefaultLoggerLevel = LogError` to prevent leaking the Base64-encoded `ServerKey` in debug output. Do not downgrade.

## Testing

- Tests run **without a database**. `payment_test.go` and `response_test.go` use `gin.TestMode` with `httptest.NewRecorder()`. Model tests are pure struct checks.
- Payment tests reset `midtransServerKey` and `midtransOnce` (`sync.Once`) between cases — copy this pattern if testing other `sync.Once`-initialized globals.
- Don't add integration tests that need PostgreSQL without mocking `config.DB`.

## Response Format

v1 routes use `pkg/response` helpers: `OK()`, `Error()`, `CreatedWithMessage()`. Format is `{"status", "message", "data"}`. Validation errors use `FormatBindError()` to produce clean messages — never return raw Gin binding errors. Compat routes use raw `gin.H` (see "Dual response styles" above).

## Key Dependencies

- **HTTP:** `gin-gonic/gin`
- **ORM:** `gorm.io/gorm` + `gorm.io/driver/postgres`
- **Export:** `xuri/excelize/v2` (Excel), `jung-kurt/gofpdf` (PDF)
- **Payment:** `midtrans/midtrans-go`
- **WebSocket:** `gorilla/websocket` (1v1 match spectator)
- **LLM:** Multi-provider via `pkg/llm` (OpenAI, Gemini, Claude, Minimax; OpenAI-compatible proxies like DeepSeek via `AI_BASE_URL`)
- **Notifications:** Telegram Bot API (async on payment settlement)

Module name: `oamp-backend` (no namespace prefix).