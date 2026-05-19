# OAMP Backend

REST API server for the OtakAtik-Robotics cognitive measurement platform. Handles participant registration, Midtrans payment, robot game sessions, leaderboard with CTF-style scoring, AI health analysis, quiz, and report exports (Excel/PDF).

**Pay-first model:** Participants must complete payment before accessing robot sessions and AI analysis. Targets all ages: TK, SD, SMP, SMA, Mahasiswa, Umum (age 3+).

## Tech Stack

| Layer | Technology |
|-------|------------|
| Language | Go 1.26+ |
| Framework | Gin (HTTP router) |
| ORM | GORM (PostgreSQL) |
| Database | PostgreSQL |
| AI | Multi-Provider LLM (OpenAI, Gemini, Claude, Minimax) |
| Export | excelize (Excel), gofpdf (PDF) |
| Security | golang.org/x/time (rate limiting), go-playground/validator |
| Payment | Midtrans Snap (Sandbox) |
| Notifications | Telegram Bot API |
| Real-time | gorilla/websocket (1v1 match spectator) |

## Project Structure

```
cmd/api/main.go                    # Entry point; loads .env, connects DB, starts server
internal/
  config/database.go               # DB connection + GORM AutoMigrate
  middleware/
    ratelimit.go                   # Per-IP rate limiter (10 req/sec, burst 30)
    bodylimit.go                   # Request body size limit (2MB)
  controller/
    participant.go                 # POST/GET /api/v1/participants, filter by batch_id
    robot.go                       # Robot auth, sessions, face logs (premium-gated)
    app.go                         # Android auth, quiz submission
    leaderboard.go                # GET /api/v1/leaderboard, /leaderboard/timeline
    export.go                     # Excel, PDF, per-participant rapor
    batches.go                    # GET/POST /api/v1/batches (event batch management)
    analysis.go                   # GET /api/v1/participants/analysis/{uid} (AI, premium-gated)
    payment.go                    # Midtrans checkout, webhook, simulate
    match.go                      # Rooms CRUD, ranking, stats, game/event (Next.js migration)
    room_manager.go               # In-memory room manager for 1v1 matchmaking
    game.go                       # Pure game result submission (oamp-game client)
    health.go                     # GET /health
  websocket/
    room.go                      # WS room manager: players + spectators, GAME_OVER persistence
    handler.go                   # WS endpoint /ws/match/:room_id
  model/model.go                  # GORM models: Participant, GameSession, PureGameResult, etc.
  route/route.go                  # Route definitions, CORS, middleware registration
pkg/
  response/response.go             # Standardized JSON response helpers + validation formatter
  llm/
    provider.go                   # LLMProvider interface + factory (sync.Once caching)
    openai.go                     # OpenAI-compatible provider
    gemini.go                     # Google Gemini provider
    claude.go                     # Anthropic Claude provider
    minimax.go                    # Minimax provider
```

## Getting Started

### Prerequisites

- Go 1.22+
- PostgreSQL (running and accessible)

### Setup

1. **Clone and install dependencies:**
   ```bash
   git clone https://github.com/OtakAtik-Robotics/oamp-backend.git
   cd oamp-backend
   go mod tidy
   ```

2. **Create database:**
   ```bash
   createdb oamp
   ```

3. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your database credentials and AI provider settings
   ```

4. **Run the server:**
   ```bash
   go run ./cmd/api
   ```

   Tables are auto-created via GORM AutoMigrate on startup.

### Build

```bash
go build -o bin/server ./cmd/api
./bin/server
```

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
| `MIDTRANS_SERVER_KEY` | Midtrans Sandbox server key | `SB-Mid-server-xxxxx` |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token for payment alerts | `123456:ABC-DEF-...` |
| `TELEGRAM_CHAT_ID` | Telegram chat ID for notifications | `-1001234567890` |

### AI Provider (required for health analysis)

| Variable | Description | Options |
|----------|-------------|---------|
| `AI_PROVIDER` | LLM provider name | `openai`, `gemini`, `claude`, `minimax` |
| `AI_API_KEY` | API key for the provider | — |
| `AI_MODEL` | Model identifier | Provider-specific (see below) |
| `AI_BASE_URL` | Custom API base URL (optional) | For OpenAI-compatible proxies (DeepSeek, Kimi, Ollama) |
| `MINIMAX_GROUP_ID` | Minimax group ID (required for Minimax only) | — |

#### Model Reference by Provider

| Provider | Default Model | Notes |
|----------|---------------|-------|
| OpenAI | `gpt-4o-mini` | Supports `AI_BASE_URL` for compatible proxies |
| Gemini | `gemini-2.0-flash` | URL: `generativelanguage.googleapis.com` |
| Claude | `claude-sonnet-4-20250514` | URL: `api.anthropic.com` |
| Minimax | `M2-her` | Requires `MINIMAX_GROUP_ID` |

#### OpenAI-Compatible Example (DeepSeek, Kimi, Ollama)

```env
AI_PROVIDER=openai
AI_API_KEY=your-key
AI_MODEL=deepseek-chat
AI_BASE_URL=https://api.deepseek.com
```

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Server + DB health check |
| POST | `/api/v1/participants` | Register participant |
| GET | `/api/v1/participants` | List participants (filter: `?batch_id=N`) |
| POST | `/api/v1/payment/checkout/:uid` | Midtrans Snap token (premium gate) |
| POST | `/api/v1/payment/webhook` | Midtrans notification (SHA512 validated) |
| POST | `/api/v1/payment/simulate-success/:uid` | Test premium without payment |
| GET | `/api/v1/robot/auth/:uid` | Robot look up participant by UID |
| POST | `/api/v1/robot/sessions` | Submit game session (premium-gated) |
| POST | `/api/v1/robot/logs/face` | Submit batch face expression logs |
| POST | `/api/v1/game/submit` | Pure game result (premium-gated, no face/emotion) |
| WS | `/ws/match/:room_id?role={player\|spectator}&player_id={id}` | 1v1 real-time match |
| POST | `/api/v1/rooms` | Create 1v1 match room |
| GET | `/api/v1/rooms` | List active rooms |
| POST | `/api/v1/rooms/:code/join` | Join room as player 2 |
| POST | `/api/v1/rooms/:code/leave` | Leave room |
| POST | `/api/v1/rooms/:code/ready` | Mark player ready (both ready → "playing") |
| GET | `/api/v1/ranking` | Leaderboard (CTF-style) |
| GET | `/api/v1/stats` | Aggregate statistics |
| POST | `/api/v1/game/event` | Desktop game event (join_room, leave_room) |
| GET | `/api/v1/app/auth/:uid` | Android app login |
| POST | `/api/v1/app/quiz` | Submit quiz result |
| GET | `/api/v1/leaderboard` | CTF-style top 10 leaderboard |
| GET | `/api/v1/leaderboard/timeline` | Timeline data (max 200 entries) |
| GET | `/api/v1/batches` | List all event batches |
| POST | `/api/v1/batches` | Create new event batch (auto-activates) |
| GET | `/api/v1/export/excel` | Download .xlsx report |
| GET | `/api/v1/export/pdf` | Download .pdf leaderboard |
| GET | `/api/v1/export/rapor/:uid` | Download per-participant .pdf rapor |
| GET | `/api/v1/participants/analysis/:uid` | AI health analysis (premium-gated) |

---

## Application Flow

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  1. REGISTRATION                                                                │
│                                                                                  │
│  Station PC                                                                        │
│  ┌─────────────────────────────────┐                                             │
│  │ POST /api/v1/participants       │                                             │
│  │ { uid, name, age, gender,       │                                             │
│  │   height, weight, ... }          │                                             │
│  └────────────┬────────────────────┘                                             │
│               ▼                                                                  │
│          PostgreSQL                                                              │
│       participants table                                                         │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  2. PAYMENT (Pay-First Model)                                                    │
│                                                                                  │
│  React Frontend                          Go Backend                              │
│  ┌──────────────────────┐              ┌──────────────────────┐                 │
│  │ POST /payment/       │ ───checkout──▶│ Creates Snap token  │                 │
│  │   checkout/:uid      │              │ Returns snap_token   │                 │
│  └──────────────────────┘              └──────────────────────┘                 │
│                                            │                                      │
│           ┌────────────────────────────────┼────────────────────────────┐        │
│           ▼                                ▼                                ▼        │
│  ┌──────────────────┐            ┌──────────────────┐         ┌─────────────────┐ │
│  │ Midtrans Web UI  │            │ POST /payment/    │         │ Telegram Alert  │ │
│  │ (QRIS/GoPay/CC)  │            │ webhook           │         │ (async)         │ │
│  └────────┬─────────┘            │ SHA512 validated  │         └─────────────────┘ │
│           │                      │ → is_premium=true │                          │
│           ▼                      └──────────────────┘                           │
└─────────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  3. GAME PLAY (3 paths)                                                         │
│                                                                                  │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                           │
│  │ Robot Client │   │ oamp-game    │   │ 1v1 Match   │                           │
│  │ (YOLO/etc)  │   │ (Hand Track) │   │ (WebSocket) │                           │
│  └──────┬───────┘   └──────┬───────┘   └──────┬───────┘                           │
│         │                 │                 │                                   │
│         ▼                 ▼                 ▼                                   │
│  POST /robot/sessions  POST /game/submit   WS /ws/match/:id                       │
│  {session, expressions │ {game_score,     Player sends GAME_OVER                  │
│   datasets}           blocks_hit, ...}    → broadcast to spectators               │
│                              │            → persist PureGameResult to DB         │
│                              └──────────────┘                                     │
└─────────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  4. RESULT & ANALYSIS                                                           │
│                                                                                  │
│  GET /api/v1/leaderboard        GET /api/v1/participants/analysis/:uid           │
│  CTF top 10 unique              AI Health Report (LLM)                           │
│  participants                   (premium-gated → 403 "Pay first" if unpaid)     │
│                                                                                  │
│  GET /api/v1/export/excel       GET /api/v1/export/rapor/:uid                   │
│  3-sheet workbook             Per-participant PDF rapor                        │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 1v1 Match Architecture (WebSocket)

```
Player 1 (P1)                           Player 2 (P2)
  │                                        │
  │──── POST /rooms ──────────────────────▶│  Create room, get 4-char code
  │                                        │
  │◀──── code: "AB12" ─────────────────────│
  │                                        │
  │──── WS /ws/match/AB12?role=player ────▶│  Both players connect WS
  │──── &player_id=P1 ─────────────────────▶│
  │                                        │
  │◀──── /rooms/AB12/join ────────────────▶│  P2 joins with player_name
  │                                        │
  │──── POST /rooms/AB12/ready ───────────▶│  Both call ready → status="playing"
  │                                        │
  │◀──── status="playing" ────────────────▶│
  │                                        │
  │   Real-time score broadcast via WS      │
  │   ┌─────────────────────────────────┐ │
  │   │ {type:"score_update",            │ │
  │   │  player_id:"P1",                 │ │
  │   │  game_score:85, blocks_hit:12}   │ │
  │   └──────────────┬──────────────────┘ │
  │                  ▼ (to all spectators)│
  │            Spectator dashboard         │
  │                                        │
  │   P1 sends GAME_OVER ───────────────▶│
  │   ┌─────────────────────────────────┐ │
  │   │ {type:"GAME_OVER", game_score:X, │ │
  │   │  blocks_hit:Y, play_duration:Z}  │ │
  │   └──────────────┬──────────────────┘ │
  │                  ▼                   │
  │          DB: PureGameResult         │
  │          Room destroyed when both   │
  │          players finish             │
  └─────────────────────────────────────┘
```

---

## Security

- **Rate limiting:** Per-IP, 10 requests/sec with burst of 30.
- **Body size limit:** 2MB max request body.
- **Input validation:** All endpoints validated via Gin binding tags + go-playground/validator.
- **Clean error messages:** Validation errors formatted without leaking internal struct details.
- **Filename sanitization:** Export filenames stripped of special characters via regex.
- **Graceful degradation:** AI analysis endpoint returns HTTP 200 with fallback message on provider failure (never crashes, never 500).
- **CORS:** AllowAllOrigins enabled (suitable for internal network; restrict to specific origins before production deployment).
- **Database transactions:** Game session submission uses `tx.Begin()` with rollback on any failure.
- **Webhook signature validation:** Midtrans notifications verified via SHA512 before processing. Spoofed webhooks rejected with HTTP 401.
- **Payment gate:** Robot sessions and AI analysis require `is_premium = true`. Unpaid participants get HTTP 403.
- **Telegram alerts:** Real-time payment notifications sent async on successful settlement/capture.

---

## Leaderboard Score Formula

```
score = (level_reached × 10) + (visuo_spatial_fit × 50) + (dexterity_score × 0.2)
```

| Metric | Weight | Range |
|--------|--------|-------|
| `level_reached` (1-8) | ×10 | 10–80 |
| `visuo_spatial_fit` (0-1) | ×50 | 0–50 |
| `dexterity_score` (0-100) | ×0.2 | 0–20 |

**Total range: 10–150** (always positive, level-weighted but not dominant)

---

## Related Repositories

- **[oamp-ai](https://github.com/OtakAtik-Robotics/oamp-ai)** — Python robot client (YOLO, MediaPipe, DeepFace, Wav2Vec2, ESP32 serial)
- **[oamp-android](https://github.com/OtakAtik-Robotics/oamp-android)** — Android app for quiz and participant results
