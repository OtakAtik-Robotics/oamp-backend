# OAMP Backend

REST API + WebSocket server for the OAMP cognitive assessment platform. Handles participant registration, Midtrans payment, game sessions, 1v1 duel matchmaking, tournaments, leaderboard, AI health analysis, quiz, and report exports.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.26+ |
| Framework | Gin (HTTP router) |
| ORM | GORM (PostgreSQL) |
| Database | PostgreSQL |
| Migrations | golang-migrate |
| AI | Multi-Provider LLM (OpenAI, Gemini, Claude, Minimax) |
| Export | excelize (Excel), gofpdf (PDF) |
| Security | golang.org/x/time (rate limiting), go-playground/validator |
| Payment | Midtrans Snap |
| Notifications | Telegram Bot API |
| Real-time | gorilla/websocket (1v1 match spectator) |

## Project Structure

```
cmd/api/main.go                 # Entry point; loads .env, connects DB, starts server
internal/
  config/database.go            # DB connection, GORM AutoMigrate + raw SQL patches
  middleware/
    ratelimit.go                # Per-IP rate limiter (10 req/sec, burst 30)
    bodylimit.go                # Request body size limit (2MB)
  controller/
    participant.go              # Register, list, lookup, get by UID, delete
    game.go                     # Game result submission (competition/training)
    room_controller.go          # Room CRUD, join, leave, ready, stale cleanup, duel result
    leaderboard.go              # CTF-style leaderboard + timeline
    export.go                   # Excel, PDF, per-participant rapor
    batches.go                  # Event batch CRUD + activate
    analysis.go                 # AI health analysis (premium-gated, cached)
    payment.go                  # Midtrans checkout, webhook, simulate
    tournament.go               # Single-elimination cup: bracket, matches, results
    health.go                   # GET /health
  websocket/
    room.go                     # WS room manager: players + spectators, GAME_OVER persistence
    handler.go                  # WS endpoint /ws/match/:room_id
  model/model.go                # GORM models: Participant, GameSession, Room, TournamentMatch, etc.
  route/route.go                # Route definitions, CORS, middleware registration
pkg/
  response/response.go          # Standardized JSON response helpers + validation formatter
  llm/
    provider.go                 # LLMProvider interface + factory
    openai.go                   # OpenAI-compatible provider
    gemini.go                   # Google Gemini provider
    claude.go                   # Anthropic Claude provider
    minimax.go                  # Minimax provider
migrations/                     # golang-migrate SQL migrations
```

## Getting Started

### Prerequisites

- Go 1.26+
- PostgreSQL (running and accessible)

### Setup

```bash
go mod tidy
createdb oamp
cp .env.example .env
# Edit .env with your database credentials and AI provider settings
go run ./cmd/api
```

Tables are created via golang-migrate + GORM AutoMigrate on startup.

### Build

```bash
go build -o bin/server ./cmd/api
./bin/server
```

### Testing

```bash
go test ./...                              # run all tests
go test -run TestName ./path/to/package    # single test
```

Controller tests use in-memory SQLite. No external DB needed.

---

## Configuration

### Database (required)

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `yourpassword` |
| `DB_NAME` | Database name | `oamp` |
| `DB_PORT` | Database port | `5432` |
| `PORT` | Server listen port | `8080` |

### Payment (required for checkout)

| Variable | Description | Example |
|----------|-------------|---------|
| `MIDTRANS_SERVER_KEY` | Midtrans server key | `SB-Mid-server-xxxxx` |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token for payment alerts | `123456:ABC-DEF-...` |
| `TELEGRAM_CHAT_ID` | Telegram chat ID for notifications | `-1001234567890` |

### AI Provider (required for health analysis)

| Variable | Description | Options |
|----------|-------------|---------|
| `AI_PROVIDER` | LLM provider name | `openai`, `gemini`, `claude`, `minimax` |
| `AI_API_KEY` | API key for the provider | — |
| `AI_MODEL` | Model identifier | Provider-specific |
| `AI_BASE_URL` | Custom API base URL (optional) | For OpenAI-compatible proxies |
| `MINIMAX_GROUP_ID` | Minimax group ID (Minimax only) | — |

### Model Reference by Provider

| Provider | Default Model | Notes |
|----------|---------------|-------|
| OpenAI | `gpt-4o-mini` | Supports `AI_BASE_URL` for compatible proxies |
| Gemini | `gemini-2.0-flash` | URL: `generativelanguage.googleapis.com` |
| Claude | `claude-sonnet-4-20250514` | URL: `api.anthropic.com` |
| Minimax | `M2-her` | Requires `MINIMAX_GROUP_ID` |

#### OpenAI-Compatible Example (DeepSeek)

```env
AI_PROVIDER=openai
AI_API_KEY=your-key
AI_MODEL=deepseek-chat
AI_BASE_URL=https://api.deepseek.com
```

---

## API Endpoints

### Core (v1)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Server + DB health check |
| POST | `/api/v1/participants` | Register participant |
| GET | `/api/v1/participants` | List participants (filter: `?batch_id=N`) |
| GET | `/api/v1/participants/stats` | Participants with scores |
| GET | `/api/v1/participants/id/:id` | Get participant by DB ID |
| GET | `/api/v1/participants/uid/:uid` | Get participant by UID |
| GET | `/api/v1/participants/uid/:uid/sessions` | Get participant sessions by UID |
| GET | `/api/v1/participants/lookup/:nickname` | Lookup participant by nickname |
| DELETE | `/api/v1/participants/:id` | Delete participant (cascade) |

### Payment

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/payment/checkout/:uid` | Midtrans Snap token |
| POST | `/api/v1/payment/webhook` | Midtrans notification (SHA512 validated) |
| POST | `/api/v1/payment/simulate-success/:uid` | Test premium without payment |

### Game

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/game/submit` | Submit game result (validation: task times 0-600s, visuo_spatial 0-100, cognitive_age 0-120) |
| POST | `/api/v1/game/event` | Desktop game event (join_room, level_start, heartbeat, etc.) |

### Rooms & Match

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/rooms` | List active rooms |
| POST | `/api/v1/rooms` | Create room |
| GET | `/api/v1/rooms/:code` | Get room by code |
| POST | `/api/v1/rooms/:code/join` | Join room as player 2 |
| POST | `/api/v1/rooms/:code/leave` | Leave room |
| POST | `/api/v1/rooms/:code/ready` | Mark player ready |
| POST | `/api/v1/rooms/:code/result` | Submit duel result (server-side score computation, dual submission) |
| GET | `/api/v1/rooms/:code/result` | Get duel result (winner + scores) |
| WS | `/ws/match/:room_id` | WebSocket match spectator |

### Leaderboard & Stats

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/leaderboard` | CTF-style top 10 leaderboard (`?mode=training\|competition&batch_id=N`) |
| GET | `/api/v1/leaderboard/timeline` | Timeline data (max 200 entries) |

### Event Batches

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/batches` | List all event batches |
| POST | `/api/v1/batches` | Create new event batch |
| PUT | `/api/v1/batches/:id` | Rename batch |
| DELETE | `/api/v1/batches/:id` | Delete batch |
| POST | `/api/v1/batches/:id/activate` | Activate batch |

### Tournaments

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tournaments` | List all tournaments |
| POST | `/api/v1/tournaments` | Create tournament |
| GET | `/api/v1/tournaments/:id` | Get tournament detail |
| DELETE | `/api/v1/tournaments/:id` | Delete tournament |
| POST | `/api/v1/tournaments/:id/join` | Participant joins tournament |
| POST | `/api/v1/tournaments/:id/register` | Register players to tournament |
| POST | `/api/v1/tournaments/:id/start` | Start tournament (generate bracket) |
| GET | `/api/v1/tournaments/:id/current-match` | Get current active match |
| POST | `/api/v1/tournaments/:id/matches/:mid/create-room` | Create room for match |
| POST | `/api/v1/tournaments/:id/matches/:mid/result` | Submit match result |
| GET | `/api/v1/tournaments/active-match/:uid` | Check active cup match by UID |

### Analysis & Export

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/participants/analysis/:uid` | AI health analysis (premium-gated, cached) |
| GET | `/api/v1/export/excel` | Download .xlsx report |
| GET | `/api/v1/export/pdf` | Download .pdf leaderboard |
| GET | `/api/v1/export/rapor/:uid` | Download per-participant .pdf rapor |

### Compat Routes (no v1 prefix, for desktop client)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/participants/uid/:uid` | Get participant by UID |
| POST | `/api/game/submit` | Submit game result |
| POST | `/api/game/event` | Game event |
| GET | `/api/rooms` | List rooms |
| POST | `/api/rooms` | Create room |
| GET | `/api/rooms/:code` | Get room |
| POST | `/api/rooms/:code/join` | Join room |
| POST | `/api/rooms/:code/leave` | Leave room |
| POST | `/api/rooms/:code/ready` | Set ready |
| POST | `/api/rooms/:code/result` | Submit duel result |
| GET | `/api/rooms/:code/result` | Get duel result |
| GET | `/api/tournaments/active-match/:uid` | Active cup match by UID |
| POST | `/api/tournaments/event` | Tournament event (match_started / match_finished) |

---

## Application Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│  1. REGISTRATION                                                      │
│                                                                       │
│  POST /api/v1/participants → PostgreSQL participants table            │
│  { uid, name, age, gender, height, weight, grip_strength, ... }       │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌──────────────────────────────────────────────────────────────────────┐
│  2. PAYMENT (Pay-First Model)                                         │
│                                                                       │
│  POST /api/v1/payment/checkout/:uid → Midtrans Snap token             │
│  POST /api/v1/payment/webhook → SHA512 validated → is_premium=true    │
│  POST /api/v1/payment/simulate-success/:uid → dev testing shortcut    │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌──────────────────────────────────────────────────────────────────────┐
│  3. GAME PLAY (3 paths)                                               │
│                                                                       │
│  Desktop Client              1v1 Match              Tournament Cup    │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │ POST /game/submit│  │ WS /ws/match/:id │  │ POST /tournaments│  │
│  │ POST /game/event │  │ POST /rooms/...  │  │   /:id/start     │  │
│  │ GET /...uid/:uid│  │ GAME_OVER → DB   │  │   /:id/matches/  │  │
│  └──────────────────┘  └──────────────────┘  │   /:mid/result   │  │
│                                               └──────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌──────────────────────────────────────────────────────────────────────┐
│  4. RESULT & ANALYSIS                                                 │
│                                                                       │
│  GET /api/v1/leaderboard    GET /api/v1/participants/analysis/:uid    │
│  CTF top 10                 AI Health Report (LLM, premium-gated)     │
│                                                                       │
│  GET /api/v1/export/excel   GET /api/v1/export/rapor/:uid             │
│  3-sheet workbook           Per-participant PDF rapor                 │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Duel Result Flow (Anti-Cheat)

```
Desktop Client (P1)                     Server                              Desktop Client (P2)
     │                                      │                                      │
     │── POST /game/submit ────────────────▶│  Saves GameSession                   │
     │   { uid, mode, task01-08, ... }     │  Computes server-side score          │
     │                                      │  using same formula                  │
     │                                      │                                      │
     │── POST /rooms/AB12/result ──────────▶│  Looks up GameSession               │
     │   { player_uid, player_num, score } │  Uses server score if > 0            │
     │                                      │  Sets player1_submitted = true       │
     │                                      │                                      │
     │                                      │◀── POST /rooms/AB12/result ────────│
     │                                      │  Sets player2_submitted = true       │
     │                                      │  Both submitted → determines winner   │
     │                                      │  Broadcasts match_result via WS      │
     │◀──── WS match_result ───────────────│────── WS match_result ─────────────▶│
     │  { winner, p1_score, p2_score }      │                                      │
```

---

## Leaderboard Score Formula

```
score = (level_reached × 10) + (visuo_spatial_fit × 50) + (dexterity_score × 0.2)
```

| Metric | Weight | Range |
|--------|--------|-------|
| `level_reached` (1-8) | ×10 | 10-80 |
| `visuo_spatial_fit` (0-1) | ×50 | 0-50 |
| `dexterity_score` (0-2, capped) | ×0.2 | 0-20 |

**Total range: 10-150**

Server computes this identically in `saveGameSession` and the leaderboard SQL query.

---

## Security

- **Rate limiting:** Per-IP, 10 requests/sec with burst of 30
- **Body size limit:** 2MB max request body
- **Input validation:** All endpoints validated via Gin binding tags + go-playground/validator. Game results: task times 0-600s, visuo_spatial 0-100, cognitive_age 0-120
- **Clean error messages:** Validation errors formatted without leaking internal struct details
- **Filename sanitization:** Export filenames stripped of special characters
- **Graceful degradation:** AI analysis returns HTTP 200 with fallback message on failure (never 500)
- **CORS:** AllowAllOrigins
- **Database transactions:** Game session submission uses `tx.Begin()` with rollback
- **Webhook signature validation:** Midtrans notifications verified via SHA512
- **Payment gate:** Game submission and AI analysis require `is_premium = true`
- **Dual submission:** Room and tournament results require both players to submit before determining winner
- **Server-side score computation:** Duel results use server-computed scores from GameSession, not client-provided values
- **Identity verification:** `SubmitDuelResultDB` verifies player UID matches registered participant + room player name
- **Midtrans logs suppressed:** ServerKey not leaked in debug output

## Related Repositories

- **`oamp-frontend/`** — React admin dashboard (this monorepo)
- **`oamp-bdt-dekstop-app-python/`** — Python desktop game client (this monorepo)