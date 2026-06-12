# API Reference — OAMP Backend

**Base URL (local):** `http://localhost:8080/`
**Base URL (remote):** `https://api.projectidek.dev/`

All responses follow the format:
```json
{
  "status": "success" | "error" | "fallback",
  "message": "...",
  "data": { ... } | null
}
```

---

## Table of Contents

1. [Health Check](#1-health-check)
2. [Participants](#2-participants)
3. [Robot](#3-robot)
4. [Android App](#4-android-app)
5. [Leaderboard](#5-leaderboard)
6. [Export](#6-export)
7. [Event Batches](#7-event-batches)
8. [Payment](#8-payment)
9. [AI Health Consultant](#9-ai-health-consultant)
10. [1v1 Match Rooms](#10-1v1-match-rooms)
11. [Game Client (oamp-game)](#11-game-client-oamp-game)
12. [Data Models](#12-data-models)

---

## 1. Health Check

### `GET /health`

Server liveness + database connectivity check.

**Response `200`:**
```json
{
  "status": "success",
  "message": "",
  "data": {
    "status": "healthy",
    "database": "connected"
  }
}
```

**Response `503` (database down):**
```json
{
  "status": "error",
  "message": "Database unreachable",
  "data": null
}
```

---

## 2. Participants

### `POST /api/v1/participants`

Register a new participant at the registration station.

**Request:**
```json
{
  "uid": "BCR-001",
  "name": "Budi Santoso",
  "age": 10,
  "grade": "5",
  "gender": "male",
  "height": 135.5,
  "weight": 30.2,
  "heart_rate": 85,
  "grip_strength": 12.3,
  "dexterity": 20.0
}
```

**Validation rules:**
| Field | Rules |
|-------|-------|
| `uid` | required, unique |
| `name` | required |
| `age` | required, >= 3 |
| `grade` | required, free text (e.g. "TK-A", "5", "SMP-2", "SMA-1", "Mahasiswa", "Umum") |
| `gender` | required, one of: `male`, `female` |
| `height` | optional, > 0, <= 300 (auto-filled by hardware later if empty) |
| `weight` | optional, > 0, <= 500 (auto-filled by hardware later if empty) |
| `heart_rate` | optional, 40-220 |
| `grip_strength` | optional, >= 0 |
| `dexterity` | optional, >= 0 |

**Response `201`:**
```json
{
  "status": "success",
  "message": "Participant registered successfully",
  "data": {
    "id": 1,
    "uid": "BCR-001",
    "name": "Budi Santoso",
    "age": 10,
    "grade": "5",
    "gender": "male",
    "height": 135.5,
    "weight": 30.2,
    "heart_rate": 85,
    "grip_strength": 12.3,
    "dexterity": 20.0,
    "is_premium": false,
    "created_at": "2026-04-12T10:00:00Z"
  }
}
```

**Response `400` (validation error):**
```json
{
  "status": "error",
  "message": "Name is required; Age is required",
  "data": null
}
```

---

### `GET /api/v1/participants`

List all participants, optionally filtered by event batch.

**Query parameters:**
| Param | Description |
|-------|-------------|
| `batch_id` | Filter to participants who have at least one session in this batch |

**Response `200`:**
```json
{
  "status": "success",
  "message": "Participants fetched successfully",
  "data": [
    {
      "id": 1,
      "uid": "BCR-001",
      "name": "Budi Santoso",
      "age": 10,
      "grade": "5",
      "gender": "male",
      "height": 135.5,
      "weight": 30.2,
      "heart_rate": 85,
      "grip_strength": 12.3,
      "is_premium": false,
      "created_at": "2026-04-12T10:00:00Z"
    }
  ]
}
```

---

### `GET /api/v1/participants/id/:id`

Lookup participant by numeric database ID.

**Response `200`:**
```json
{
  "status": "success",
  "message": "",
  "data": {
    "id": 1,
    "uid": "BCR-001",
    "name": "Budi Santoso",
    "age": 10,
    "grade": "5",
    "gender": "male",
    "height": 135.5,
    "weight": 30.2,
    "is_premium": true,
    "created_at": "2026-04-12T10:00:00Z"
  }
}
```

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```
```

---

### `GET /api/v1/participants/uid/:uid`

Lookup participant by barcode UID. Used by game client after bracelet scan to retrieve identity + body measurements.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Participant found",
  "data": {
    "id": 1,
    "uid": "BCR-001",
    "name": "Budi Santoso",
    "age": 10,
    "grade": "5",
    "gender": "male",
    "height": 170.5,
    "weight": 65.2,
    "heart_rate": 85,
    "grip_strength": 45.0,
    "dexterity": 20.0,
    "is_premium": false,
    "created_at": "2026-04-12T10:00:00Z"
  }
}
```

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

---

### `PUT /api/v1/participants/uid/:uid`

Update body measurements for a participant by UID. Designed for **hardware auto-fill** — a measurement station uploads height, weight, grip strength, and dexterity after registration. Only the fields you send are updated (partial update).

**Request (hardware auto-fill — all 4 fields):**
```json
{
  "height": 170.5,
  "weight": 65.2,
  "grip_strength": 45.0,
  "dexterity": 20.0
}
```

**Request (partial update — height only):**
```json
{
  "height": 172.0
}
```

**Validation rules:**
| Field | Rules |
|-------|-------|
| `height` | > 0, <= 300 |
| `weight` | > 0, <= 500 |
| `grip_strength` | >= 0, <= 200 |
| `dexterity` | >= 0, <= 500 |

**Response `200`:**
```json
{
  "status": "success",
  "message": "Participant updated",
  "data": {
    "id": 1,
    "uid": "BCR-001",
    "name": "Budi Santoso",
    "age": 10,
    "grade": "5",
    "gender": "male",
    "height": 170.5,
    "weight": 65.2,
    "heart_rate": 85,
    "grip_strength": 45.0,
    "dexterity": 20.0,
    "is_premium": false,
    "created_at": "2026-04-12T10:00:00Z"
  }
}
```

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

**Response `400` (no updatable fields):**
```json
{
  "status": "error",
  "message": "No updatable fields provided",
  "data": null
}
```

**Flow:**
```
Registrasi (keyboard)         Hardware auto-fill          Game client
─────────────────────         ──────────────────          ───────────
POST /participants      ──►   PUT /uid/:uid         ──►   GET /uid/:uid
UID, nama, gender,            height, weight,              ▶ semua data
age, grade                    grip_strength,               siap main &
                              dexterity                    cetak rapor
```

---

## 3. Robot

### `GET /api/v1/robot/auth/:uid`

Robot looks up a participant by UID (barcode) for height calibration.
Returns participant data including `height` for actuator adjustment.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Participant found",
  "data": {
    "id": 1,
    "uid": "BCR-001",
    "name": "Budi Santoso",
    "age": 10,
    "grade": "5",
    "gender": "male",
    "height": 135.5,
    "weight": 30.2,
    "heart_rate": 85,
    "grip_strength": 12.3,
    "is_premium": true,
    "created_at": "2026-04-12T10:00:00Z"
  }
}
```

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

---

### `POST /api/v1/robot/sessions`

Submit game session results after a child finishes playing.
Uses a database transaction to atomically create the session, face expression logs, and dataset captures.

**Request:**
```json
{
  "session": {
    "participant_id": 1,
    "mode": "normal",
    "level_reached": 6,
    "total_time": 18.5,
    "visuo_spatial_fit": 0.91,
    "dexterity_score": 0.0
  },
  "expressions": [
    {
      "level": 1,
      "dominant_emotion": "happy",
      "timestamp": "2026-04-12T10:05:00Z"
    },
    {
      "level": 2,
      "dominant_emotion": "surprise",
      "timestamp": "2026-04-12T10:05:15Z"
    }
  ],
  "datasets": [
    {
      "camera_source": 0,
      "image_path": "/captures/session1_frame001.jpg"
    }
  ]
}
```

**Field reference:**
| Section | Field | Required | Description |
|---------|-------|----------|-------------|
| `session` | `participant_id` | yes | From `GET /robot/auth/:uid` response |
| `session` | `mode` | no | Game mode (e.g. "normal") |
| `session` | `level_reached` | no | Highest level completed |
| `session` | `total_time` | no | Total play time in seconds |
| `session` | `visuo_spatial_fit` | no | Visuo-spatial fitness score (0-1) |
| `session` | `dexterity_score` | no | Dexterity score |
| `expressions` | `level` | no | Game level when emotion was recorded |
| `expressions` | `dominant_emotion` | no | happy, sad, angry, fear, surprise, disgust, neutral |
| `expressions` | `timestamp` | no | ISO 8601 timestamp |
| `datasets` | `camera_source` | no | Camera index (0 = game, 1 = face) |
| `datasets` | `image_path` | no | Path to captured image |

**Response `201`:**
```json
{
  "status": "success",
  "message": "Session recorded successfully",
  "data": {
    "session_id": 1
  }
}
```

**Response `400` (participant not found):**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

**Response `403` (not premium):**
```json
{
  "status": "error",
  "message": "Pay first",
  "data": null
}
```

---

### `POST /api/v1/robot/logs/face`

Submit batch face expression logs separately from the main session.
Useful for sending additional logs after the session has been recorded.

**Request:**
```json
{
  "session_id": 1,
  "logs": [
    {
      "level": 3,
      "dominant_emotion": "happy",
      "timestamp": "2026-04-12T10:06:00Z"
    },
    {
      "level": 4,
      "dominant_emotion": "neutral",
      "timestamp": "2026-04-12T10:06:15Z"
    }
  ]
}
```

**Response `201`:**
```json
{
  "status": "success",
  "message": "Face logs saved successfully",
  "data": {
    "count": 2
  }
}
```

**Response `400` (empty logs):**
```json
{
  "status": "error",
  "message": "No logs provided",
  "data": null
}
```

---

## 4. Android App

### `GET /api/v1/app/auth/:uid`

Login for the Android app. Returns participant data and all their game sessions.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Login successful",
  "data": {
    "participant": {
      "id": 1,
      "uid": "BCR-001",
      "name": "Budi Santoso",
      "age": 10,
      "grade": "5",
      "gender": "male",
      "height": 135.5,
      "weight": 30.2,
      "heart_rate": 85,
      "grip_strength": 12.3,
      "is_premium": false,
      "created_at": "2026-04-12T10:00:00Z"
    },
    "sessions": [
      {
        "id": 1,
        "participant_id": 1,
        "mode": "normal",
        "level_reached": 6,
        "total_time": 18.5,
        "visuo_spatial_fit": 0.91,
        "dexterity_score": 0.0,
        "created_at": "2026-04-12T10:10:00Z"
      }
    ]
  }
}
```

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

---

### `POST /api/v1/app/quiz`

Submit a quiz result from the Android app.

**Request:**
```json
{
  "participant_id": 1,
  "score": 85,
  "answers_data": "{\"q1\":\"A\",\"q2\":\"B\",\"q3\":\"C\"}"
}
```

**Response `201`:**
```json
{
  "status": "success",
  "message": "Quiz result saved successfully",
  "data": {
    "quiz_id": 1
  }
}
```

---

## 5. Leaderboard

### `GET /api/v1/leaderboard`

CTF-style leaderboard. Returns top 10 participants based on their best game session.
One entry per participant (uses PostgreSQL `DISTINCT ON`).

**Score formula:**
```
score = (level_reached × 1000) - (total_time × 10)
```

| Metric | Weight | Contribution |
|--------|--------|-------------|
| `level_reached` (1-8) | ×1000 | 1000 - 8000 points |
| `total_time` (seconds) | ×10 | penalty per second |
| Score capped at | minimum | 0 |

Range: 1000 - 8000. Score = (level_reached × 1000) - (total_time × 10), capped min 0.

**Query parameters:**
| Param | Description |
|-------|-------------|
| `batch_id` | Filter leaderboard to a specific event batch (defaults to active batch) |

**Response `200`:**
```json
{
  "status": "success",
  "message": "Leaderboard fetched successfully",
  "data": [
    {
      "rank": 1,
      "participant_id": 1,
      "uid": "BCR-001",
      "name": "Dina Permata",
      "grade": "6A",
      "age": 11,
      "visuo_spatial_fit": 0.95,
      "total_time": 14.2,
      "level_reached": 8,
      "dexterity_score": 95.0,
      "score": 7858
    },
    {
      "rank": 2,
      "participant_id": 2,
      "uid": "BCR-002",
      "name": "Budi Santoso",
      "grade": "5",
      "age": 10,
      "visuo_spatial_fit": 0.91,
      "total_time": 18.5,
      "level_reached": 6,
      "dexterity_score": 88.5,
      "score": 5815
    }
  ]
}
```

Returns empty array `[]` when no sessions have been recorded yet.

---

### `GET /api/v1/leaderboard/timeline`

Returns all game sessions ordered by creation time (max 200 entries). Used for timeline graph on the dashboard.

**Query parameters:**
| Param | Description |
|-------|-------------|
| `batch_id` | Filter timeline to a specific event batch (defaults to active batch) |

**Response `200`:**
```json
{
  "status": "success",
  "message": "Timeline fetched successfully",
  "data": [
    {
      "name": "Budi Santoso",
      "score": 108.7,
      "created_at": "2026-04-12T10:10:00Z"
    },
    {
      "name": "Ani Lestari",
      "score": 92.5,
      "created_at": "2026-04-12T10:15:00Z"
    }
  ]
}
```

Each entry represents one game session (not unique per participant).

---

## 6. Export

### `GET /api/v1/export/excel`

Downloads an Excel (.xlsx) file with 4 sheets:

| Sheet | Contents |
|-------|----------|
| Leaderboard | All ranked participants (best session per person) |
| Participants | All registered participant data |
| Sessions | All game session records |
| GameResults | Per-UUID game results (task01-08, cognitive_age, visuo_spatial, variant_list, cog_age_list) |

**Response:** Binary `.xlsx` file download (`Content-Disposition: attachment; filename=oamp-report.xlsx`)

---

### `GET /api/v1/export/pdf`

Downloads a PDF file with the leaderboard table.

**Response:** Binary `.pdf` file download (`Content-Disposition: attachment; filename=oamp-leaderboard.pdf`)

If no sessions exist, the PDF contains the text "No game sessions recorded yet."

---

### `GET /api/v1/export/rapor/:uid`

Downloads a PDF rapor (report card) for an individual participant.

**URL parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `uid` | string | Participant UID (barcode identifier) |

**PDF contents:**

| Section | Details |
|---------|---------|
| Header | "Rapor Peserta OAMP" + subtitle |
| Data Pribadi | UID, Kelas, Umur, Jenis Kelamin, Tinggi, Berat, Detak Jantung, Grip Strength |
| Riwayat Game | Table of all sessions: date, mode, level, time, VisuoSpatialFit, Dexterity |
| Ringkasan Performa | Total sessions, best VisuoSpatial score, highest level, average time |
| Hasil Quiz | Table of quiz results (if any): date, score |
| Footer | Print date |

**Response `200`:** Binary `.pdf` file (`Content-Disposition: attachment; filename=rapor-{name}.pdf`)

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

**Frontend download usage:**
```js
const res = await api.get(`/export/rapor/${uid}`, { responseType: "blob" });
const url = window.URL.createObjectURL(res);
const link = document.createElement("a");
link.href = url;
link.setAttribute("download", `rapor-${uid}.pdf`);
link.click();
```

---

## 7. Event Batches

### `GET /api/v1/batches`

Returns all event batches ordered by creation time (newest first).

**Response `200`:**
```json
{
  "status": "success",
  "message": "Batches fetched successfully",
  "data": [
    {
      "id": 2,
      "name": "Sesi Pameran Bandung 2026",
      "is_active": true,
      "created_at": "2026-04-15T19:12:24+07:00"
    },
    {
      "id": 1,
      "name": "Sesi Default",
      "is_active": false,
      "created_at": "2026-04-15T19:12:12+07:00"
    }
  ]
}
```

---

### `POST /api/v1/batches`

Creates a new event batch and sets it as the active batch. All previously active batches are deactivated.

**Request:**
```json
{
  "name": "Sesi Pameran Bandung 2026"
}
```

**Validation rules:**
| Field | Rules |
|-------|-------|
| `name` | required |

**Response `201`:**
```json
{
  "status": "success",
  "message": "Batch created successfully",
  "data": {
    "id": 2,
    "name": "Sesi Pameran Bandung 2026",
    "is_active": true,
    "created_at": "2026-04-15T19:12:24+07:00"
  }
}
```

---

## 8. Payment

### `POST /api/v1/payment/checkout/:uid`

Create a Midtrans Snap transaction. Returns a `snap_token` to render the payment popup and a `redirect_url` for direct payment page.

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `uid` | string | Participant UID (barcode identifier) |

**Response `200`:**
```json
{
  "status": "success",
  "message": "Checkout initiated",
  "data": {
    "token": "snap_token_string",
    "redirect_url": "https://app.sandbox.midtrans.com/snap/v2/vtweb/...",
    "order_id": "OAMP-BCR-001-1713500000000000000",
    "amount": 10000,
    "currency": "IDR"
  }
}
```

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

**Response `503`:**
```json
{
  "status": "error",
  "message": "Payment service not configured",
  "data": null
}
```

---

### `POST /api/v1/payment/webhook`

Midtrans payment notification webhook. Validates SHA512 signature before processing. Always returns HTTP 200 (per Midtrans spec).

**Signature validation:** `SHA512(order_id + status_code + gross_amount + MIDTRANS_SERVER_KEY)` must match `signature_key` field.

**Response `401` (invalid signature):**
```json
{
  "status": "invalid signature"
}
```

**Response `200` (accepted):**
```json
{
  "status": "ok"
}
```

On successful payment (`transaction_status` = `settlement` or `capture`), sets `is_premium = true` on the participant and sends a Telegram notification.

---

### `POST /api/v1/payment/simulate-success/:uid`

Internal test endpoint. Directly sets `is_premium = true` without real payment. Sends Telegram notification.

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `uid` | string | Participant UID |

**Response `200`:**
```json
{
  "status": "success",
  "message": "Payment successful",
  "data": {
    "uid": "BCR-001",
    "is_premium": true,
    "paid_at": "2026-04-19T13:45:00Z"
  }
}
```

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

---

## 9. AI Health Consultant

### `GET /api/v1/participants/analysis/:uid`

Generates an AI-powered health analysis for a participant using LLM. BMI calculation, average game performance, personalized physical activity recommendations in Markdown format. Premium-gated (`is_premium = true` required).

**URL Parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `uid` | string | Participant UID (barcode identifier) |

**Data Aggregated:**
- Participant biodata (age, gender, height, weight, heart_rate, grip_strength)
- All game sessions for average visuo-spatial fit and dexterity score
- BMI calculation: `Weight / ((Height/100)²)`

**LLM Providers Supported:** OpenAI, Gemini, Claude, Minimax (configured via `AI_PROVIDER` env var).

**Response `200` (success):**
```json
{
  "status": "success",
  "message": "Analysis generated",
  "data": {
    "analysis": "## Analisis Kesehatan\n\nBerdasarkan data yang diberikan untuk **Dina Permata (11 tahun)**:\n\n- **BMI**: 17.2 (Normal)\n- **Kekuatan Grip**: 15.2 kg\n\n### Saran Aktivitas Fisik:\n- **Latihan Motorik Kasar**: Berlari, melompat tali, bermain bola\n- **Latihan Motorik Halus**: Meronce, menyusun balok, menggambar\n- **Aktivitas Kardio**: Jalan cepat 15-20 menit"
  }
}
```

**Response `200` (fallback — AI service offline):**
```json
{
  "status": "fallback",
  "message": "AI service offline",
  "data": {
    "analysis": "Mohon maaf, layanan AI Health Analysis saat ini sedang sibuk atau tidak dapat diakses akibat gangguan jaringan. Silakan coba beberapa saat lagi."
  }
}
```

> Both success and fallback return HTTP 200 OK. The `status` field differentiates them. This is intentional graceful degradation.

**Response `403` (not premium):**
```json
{
  "status": "error",
  "message": "Pay first",
  "data": null
}
```

**Response `404` (participant not found):**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

---

## 10. 1v1 Match Rooms

In-memory room manager for real-time 1v1 matches. Rooms auto-cleanup after 5 minutes of inactivity.

### `GET /api/v1/rooms`

List all active rooms (status: `waiting`, `ready`, or `playing`).

**Response `200`:**
```json
{
  "status": "success",
  "message": "Rooms fetched",
  "data": [
    {
      "id": "AB12",
      "status": "waiting",
      "player1_name": "Budi",
      "player2_name": "",
      "player1_ready": false,
      "player2_ready": false,
      "last_activity": "2026-05-12T10:00:00Z"
    }
  ]
}
```

---

### `POST /api/v1/rooms`

Create a new room. Player 1 is assigned a 4-char room code (e.g. `AB12`).

**Request:**
```json
{ "player_name": "Budi" }
```

**Response `201`:**
```json
{
  "status": "success",
  "message": "Room created",
  "data": {
    "id": "AB12",
    "status": "waiting",
    "player1_name": "Budi",
    "player2_name": "",
    "player1_ready": false,
    "player2_ready": false,
    "last_activity": "2026-05-12T10:00:00Z"
  }
}
```

---

### `GET /api/v1/rooms/:code`

Get room details by code.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Room fetched",
  "data": {
    "id": "AB12",
    "status": "ready",
    "player1_name": "Budi",
    "player2_name": "Ani",
    "player1_ready": false,
    "player2_ready": false,
    "last_activity": "2026-05-12T10:05:00Z"
  }
}
```

**Response `404`:**
```json
{ "status": "error", "message": "Room not found", "data": null }
```

---

### `POST /api/v1/rooms/:code/join`

Join an existing room as Player 2.

**Request:**
```json
{ "player_name": "Ani" }
```

**Response `200`:**
```json
{
  "status": "success",
  "message": "Joined room",
  "data": {
    "id": "AB12",
    "status": "ready",
    "player1_name": "Budi",
    "player2_name": "Ani",
    "player1_ready": false,
    "player2_ready": false
  }
}
```

**Response `404`:**
```json
{ "status": "error", "message": "Room not found", "data": null }
```

**Response `409`:**
```json
{ "status": "error", "message": "Room is full", "data": null }
```

---

### `POST /api/v1/rooms/:code/leave`

Leave a room. If Player 1 leaves and Player 2 exists, Player 2 is promoted to Player 1.

**Request:**
```json
{ "player_name": "Ani" }
```

**Response `200`:**
```json
{
  "status": "success",
  "message": "Left room",
  "data": { "ok": true }
}
```

**Response `404`:**
```json
{ "status": "error", "message": "Room not found", "data": null }
```

---

### `POST /api/v1/rooms/:code/ready`

Mark a player as ready. When both players are ready, room status changes to `playing`.

**Request:**
```json
{ "player_name": "Budi" }
```

**Response `200`:**
```json
{
  "status": "success",
  "message": "Ready set",
  "data": {
    "id": "AB12",
    "status": "playing",
    "player1_name": "Budi",
    "player2_name": "Ani",
    "player1_ready": true,
    "player2_ready": true
  }
}
```

**Response `404`:**
```json
{ "status": "error", "message": "Room not found", "data": null }
```

---

## 11. Game Client (oamp-game)

The desktop game client (`oamp-game`) communicates with the backend via HTTP + WebSocket.

### 11.1 Submit Pure Game Session

#### `POST /api/v1/game/submit`

Submit pure game metrics after a session completes. No face/voice/emotion data — only gameplay metrics. Premium-gated.

**Request:**
```json
{
  "participant_id": 1,
  "game_score": 85,
  "blocks_hit": 12,
  "hand_tracking_status": "active",
  "play_duration": 42.5,
  "timestamp": "2026-05-13T10:05:00Z"
}
```

**Field reference:**
| Field | Required | Description |
|-------|----------|-------------|
| `participant_id` | yes | Numeric database ID |
| `game_score` | yes | Final score (0-100) |
| `blocks_hit` | yes | Total blocks hit during session |
| `hand_tracking_status` | yes | "active" if MediaPipe worked, "none" if unavailable |
| `play_duration` | yes | Total play time in seconds |
| `timestamp` | yes | ISO 8601 UTC timestamp |

**Response `201`:**
```json
{
  "status": "success",
  "message": "Session recorded",
  "data": {
    "session_id": 42
  }
}
```

**Response `400` (validation error):**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

**Response `403` (not premium):**
```json
{
  "status": "error",
  "message": "Pay first",
  "data": null
}
```

---

### 11.2 Game Event

#### `POST /api/v1/game/event`

Desktop app game event notifications. Handles `join_room` and `leave_room` types.

**Request:**
```json
{
  "type": "join_room",
  "room_id": "AB12",
  "player_name": "Budi"
}
```

**Response `200`:**
```json
{
  "status": "success",
  "message": "Event processed",
  "data": { "ok": true }
}
```

---

### 11.3 Ranking & Stats

#### `GET /api/v1/ranking`

Returns top 100 participants who have `ai_analysis` present, ordered by creation time.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Ranking fetched",
  "data": [
    { "rank": 1, "uid": "BCR-001", "name": "Dina", "age": 11, "task_avg": 0, "score": 0 }
  ]
}
```

---

#### `GET /api/v1/stats`

Aggregate statistics across all participants.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Stats fetched",
  "data": {
    "total_participants": 42,
    "avg_time": 0,
    "min_time": 0,
    "max_time": 0,
    "avg_visuo_spatial": 0
  }
}
```

---

### 11.4 WebSocket — Real-time 1v1 Match

#### `WS /ws/match/:room_id?role={player|spectator}&player_id={id}`

Real-time telemetry channel for 1v1 matches.

**Connections:**
```
# Player
ws://localhost:8080/ws/match/AB12?role=player&player_id=P1

# Spectator
ws://localhost:8080/ws/match/AB12?role=spectator&player_id=S1
```

#### Player → Server

**Throttled state updates (~5 Hz):**
```json
{ "game_score": 3, "blocks_hit": 12 }
```

**Immediate collision hit (non-throttled):**
```json
{ "type": "SCORE_UPDATE", "game_score": 3, "blocks_hit": 12 }
```

**Game over (sent exactly once at session end):**
```json
{ "type": "GAME_OVER", "game_score": 85, "blocks_hit": 12, "play_duration": 42.5 }
```

#### Server → Spectator (broadcasts)

```json
{ "type": "join", "player_id": "P1" }
{ "type": "score_update", "player_id": "P1", "game_score": 85, "blocks_hit": 12 }
{ "type": "GAME_OVER", "player_id": "P1", "game_score": 95, "blocks_hit": 15 }
{ "type": "leave", "player_id": "P1" }
```

**Rules:** Max 2 players, unlimited spectators. Room is destroyed when both players send GAME_OVER. 5-minute stale cleanup.

**LAN Master Push Mode:** For offline LAN setups, the client can push directly to a master PC:
```bash
MASTER_IP=192.168.1.100   # Master PC IP
MASTER_PORT=8080           # Master PC port
ROOM_ID=AB12               # Room code
```

---

## 12. Data Models

### Participant

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `uid` | string | Unique identifier (barcode) |
| `name` | string | Full name |
| `age` | int | Age in years (>= 3) |
| `grade` | string | Education level (e.g. "TK-A", "5", "SMP-2", "SMA-1", "Mahasiswa", "Umum") |
| `gender` | string | `male` or `female` |
| `height` | float64 | Height in cm (optional — auto-filled by hardware via PUT) |
| `weight` | float64 | Weight in kg (optional — auto-filled by hardware via PUT) |
| `heart_rate` | int | Resting heart rate (bpm) |
| `grip_strength` | float64 | Grip strength measurement (auto-filled by hardware) |
| `dexterity` | float64 | Dexterity measurement (auto-filled by hardware) |
| `is_premium` | bool | Premium access (default: false) |
| `ai_analysis` | string | Cached AI health analysis (Markdown) |
| `ai_analysis_updated_at` | timestamp | When analysis was last generated |
| `created_at` | timestamp | Auto-set by GORM |

### GameSession

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `participant_id` | uint | Foreign key to Participant |
| `event_batch_id` | uint | Foreign key to EventBatch (auto-assigned from active batch) |
| `mode` | string | Game mode (e.g. "normal") |
| `level_reached` | int | Highest level completed |
| `total_time` | float64 | Total play time in seconds |
| `visuo_spatial_fit` | float64 | Visuo-spatial fitness score |
| `dexterity_score` | float64 | Dexterity score |
| `created_at` | timestamp | Auto-set by GORM |

### EventBatch

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `name` | string | Batch/session name |
| `is_active` | bool | Only one batch is active at a time |
| `created_at` | timestamp | Auto-set by GORM |

### FaceExpressionLog

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `session_id` | uint | Foreign key to GameSession |
| `level` | int | Game level when recorded |
| `dominant_emotion` | string | happy, sad, angry, fear, surprise, disgust, neutral |
| `timestamp` | timestamp | When the emotion was recorded |

### DatasetCapture

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `session_id` | uint | Foreign key to GameSession |
| `camera_source` | int | Camera index (0 = game, 1 = face) |
| `image_path` | string | Path to captured image file |
| `created_at` | timestamp | Auto-set by GORM |

### QuizResult

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `participant_id` | uint | Foreign key to Participant |
| `score` | int | Quiz score |
| `answers_data` | string | JSON string of answers |
| `created_at` | timestamp | Auto-set by GORM |

### PureGameResult

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `participant_id` | uint | Foreign key to Participant |
| `game_score` | int | Final score from game (WS GAME_OVER) |
| `blocks_hit` | int | Total blocks hit |
| `hand_tracking_status` | string | Hand tracking state ("active" or "none") |
| `play_duration` | float64 | Play duration in seconds |
| `timestamp` | timestamp | Game timestamp |
| `created_at` | timestamp | Auto-set by GORM |

---

## Error Responses (apply to all endpoints)

**Response `429` (rate limited):**
```json
{
  "status": "error",
  "message": "Too many requests, please try again later",
  "data": null
}
```

Rate limit: 10 requests/sec per IP, burst of 30.

**Response `413` (body too large):**
Request body exceeds 2MB limit.

---

## 13. Additional Endpoints

### `GET /api/v1/participants/uid/:uid/results`

Get per-level game result data by UID. Returns all 8 task times, cognitive age, visuo-spatial score, and variant list. Used by the frontend analytics per-user page.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Participant result fetched",
  "data": {
    "uid": "BCR-001",
    "task01": 5.2, "task02": 4.1, "task03": 3.8, "task04": 5.5,
    "task05": 6.0, "task06": 4.2, "task07": 5.1, "task08": 6.3,
    "cognitive_age": 12,
    "visuo_spatial": 65.0,
    "variant_list": ["1a", "2c", "3b", "4d", "5a", "6b", "7c", "8d"],
    "cog_age_list": [10, 11, 12, 13, 14, 15, 16, 17]
  }
}
```

---

### `GET /api/v1/stats`

Aggregate statistics across all participants with game results.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Stats fetched",
  "data": {
    "total_participants": 42,
    "avg_time": 45.2,
    "min_time": 12.3,
    "max_time": 120.5,
    "avg_cognitive_age": 10.5,
    "avg_visuo_spatial": 55.0,
    "total_male": 20,
    "total_female": 22,
    "level_avgs": { "1": 5.2, "2": 4.8, "3": 5.5, "4": 6.1, "5": 6.8, "6": 7.2, "7": 7.8, "8": 8.5 },
    "timeline": [ { "name": "Budi", "score": 5815, "created_at": "..." }, ... ]
  }
}
```

---

### `GET /api/v1/stations`

Active station health — returns stations with recent heartbeat or game submissions.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Stations fetched",
  "data": [
    { "player_name": "Budi", "room_id": "AB12", "mode": "competition", "status": "playing" }
  ]
}
```

---

### `GET /api/v1/export/csv`

Download full participant + session + result data as CSV file.

**Response:** Binary `.csv` file download.

---

### `POST /api/v1/export/telegram`

Send Excel report to the configured Telegram chat. No request body needed.

**Response `200`:**
```json
{ "status": "success", "message": "Report sent to Telegram", "data": null }
```

---

### `WS /ws/event-display`

WebSocket for big-screen spectator displays. Relays real-time `score_update` and `level_start` events with `completed_levels` and `is_finished` fields.

**Connect:** `ws://localhost:8080/ws/event-display`

**Server → Client broadcasts:**
```json
{ "type": "score_update", "uid": "BCR-001", "name": "Budi", "score": 4500, "completed_levels": 5, "is_finished": false }
{ "type": "level_start", "uid": "BCR-001", "name": "Budi", "level": 6, "completed_levels": 5 }
```

---

### GameResult Model

| Field | Type | Description |
|-------|------|-------------|
| `uid` | string | Participant UID (unique, upserted) |
| `task01` .. `task08` | float64 | Per-level completion times in seconds |
| `cognitive_age` | float64 | Estimated cognitive age |
| `visuo_spatial` | float64 | Visuo-spatial score |
| `variant_list` | Float64Array | Level variant codes (e.g. ["1a","2b",...]) |
| `cog_age_list` | Float64Array | Per-level cognitive age estimates |
