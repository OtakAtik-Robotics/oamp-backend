# OAMP Backend

REST API server for the OtakAtik-Robotics event platform.
Handles participant registration, robot game sessions, leaderboard, quiz, and report exports.

Maintained by the Backend Developers of OtakAtik-Robotics.

## Tech Stack

- **Language:** Go
- **Framework:** Gin
- **ORM:** GORM
- **Database:** PostgreSQL
- **Export:** Excelize (Excel), gofpdf (PDF)

## Project Structure

```
cmd/api/main.go            # Entry point
internal/
  config/database.go       # DB connection + AutoMigrate
  controller/              # Route handlers
    participant.go         #   POST /participants
    robot.go               #   Robot auth, sessions, face logs
    app.go                 #   Android auth, quiz
    leaderboard.go         #   GET /leaderboard
    export.go              #   Excel & PDF export
    health.go              #   GET /health
  model/model.go           # GORM models
  route/route.go           # Route definitions + CORS
pkg/response/response.go   # Standardized JSON response helper
```

## Getting Started

### Prerequisites

- Go 1.22+
- PostgreSQL

### Setup

1. Clone and install dependencies:
   ```bash
   git clone https://github.com/OtakAtik-Robotics/oamp-backend.git
   cd oamp-backend
   go mod tidy
   ```

2. Create database:
   ```bash
   createdb oamp
   ```

3. Copy `.env.example` to `.env` and fill in your database credentials:
   ```bash
   cp .env.example .env
   ```

4. Run the server:
   ```bash
   go run ./cmd/api
   ```

Tables are auto-created via GORM AutoMigrate on startup.

### Build

```bash
go build -o bin/server ./cmd/api
./bin/server
```

## API Documentation

See [API.md](API.md) for the full endpoint reference with request/response examples.

## Related Repositories

- **oamp-ai** — Python robot client (YOLO, MediaPipe, DeepFace, Wav2Vec2, ESP32 serial). Communicates with this server via `api_client.py`.
- **oamp-android** — Android app for quiz and participant results.
