# OAMP Backend

REST API server for the OtakAtik-Robotics cognitive measurement platform. Handles participant registration, robot game sessions, leaderboard with CTF-style scoring, AI health analysis, quiz, and report exports (Excel/PDF).

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

## Project Structure

```
cmd/api/main.go                    # Entry point; loads .env, connects DB, starts server
internal/
  config/database.go               # DB connection + GORM AutoMigrate
  middleware/
    ratelimit.go                   # Per-IP rate limiter (10 req/sec, burst 30)
    bodylimit.go                   # Request body size limit (2MB)
  controller/
    participant.go                 # POST /api/v1/participants
    robot.go                       # Robot auth, sessions, face logs
    app.go                         # Android auth, quiz submission
    leaderboard.go                # GET /api/v1/leaderboard, /leaderboard/timeline
    export.go                     # Excel, PDF, per-participant rapor
    batches.go                    # GET/POST /api/v1/batches (event batch management)
    analysis.go                   # GET /api/v1/participants/analysis/{uid} (AI)
    health.go                     # GET /health
  model/model.go                  # GORM models: Participant, GameSession, EventBatch, etc.
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
| GET | `/api/v1/robot/auth/{uid}` | Robot look up participant by UID |
| POST | `/api/v1/robot/sessions` | Submit game session |
| POST | `/api/v1/robot/logs/face` | Submit batch face expression logs |
| GET | `/api/v1/app/auth/{uid}` | Android app login |
| POST | `/api/v1/app/quiz` | Submit quiz result |
| GET | `/api/v1/leaderboard` | CTF-style top 10 leaderboard |
| GET | `/api/v1/leaderboard/timeline` | Timeline data (max 200 entries) |
| GET | `/api/v1/batches` | List all event batches |
| POST | `/api/v1/batches` | Create new event batch (auto-activates) |
| GET | `/api/v1/export/excel` | Download .xlsx report |
| GET | `/api/v1/export/pdf` | Download .pdf leaderboard |
| GET | `/api/v1/export/rapor/{uid}` | Download per-participant .pdf rapor |
| GET | `/api/v1/participants/analysis/{uid}` | AI health analysis (Markdown) |

Full API reference: [API.md](API.md)

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
