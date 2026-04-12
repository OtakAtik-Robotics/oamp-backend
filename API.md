# API Reference — OAMP Backend

Base URL: `http://localhost:8080/api/v1`

All responses follow the format:
```json
{
  "status": "success" | "error",
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
7. [Data Models](#7-data-models)

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
  "uid": "RFID-001",
  "name": "Budi Santoso",
  "age": 10,
  "grade": "5A",
  "gender": "male",
  "height": 135.5,
  "weight": 30.2,
  "heart_rate": 85,
  "spo2": 98.5,
  "grip_strength": 12.3
}
```

**Validation rules:**
| Field | Rules |
|-------|-------|
| `uid` | required, unique |
| `name` | required |
| `age` | required, 3-18 |
| `grade` | required |
| `gender` | required, one of: `male`, `female` |
| `height` | required, > 0 |
| `weight` | required, > 0 |
| `heart_rate` | optional, 40-220 |
| `spo2` | optional, 0-100 |
| `grip_strength` | optional, >= 0 |

**Response `201`:**
```json
{
  "status": "success",
  "message": "Participant registered successfully",
  "data": {
    "id": 1,
    "uid": "RFID-001",
    "name": "Budi Santoso",
    "age": 10,
    "grade": "5A",
    "gender": "male",
    "height": 135.5,
    "weight": 30.2,
    "heart_rate": 85,
    "spo2": 98.5,
    "grip_strength": 12.3,
    "created_at": "2026-04-12T10:00:00Z"
  }
}
```

**Response `400` (validation error):**
```json
{
  "status": "error",
  "message": "Key: 'Participant.Name' Error:Field validation for 'Name' failed on the 'required' tag",
  "data": null
}
```

---

## 3. Robot

### `GET /api/v1/robot/auth/{uid}`

Robot looks up a participant by UID (RFID/QR) for height calibration.
Returns `height` so the robot can adjust its actuator.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Participant found",
  "data": {
    "id": 1,
    "uid": "RFID-001",
    "name": "Budi Santoso",
    "age": 10,
    "grade": "5A",
    "gender": "male",
    "height": 135.5,
    "weight": 30.2,
    "heart_rate": 85,
    "spo2": 98.5,
    "grip_strength": 12.3,
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
    "cognitive_age": 11,
    "visuo_spatial_fit": 0.91,
    "dexterity_score": 88.5
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
| `session` | `participant_id` | yes | From `GET /robot/auth/{uid}` response |
| `session` | `mode` | no | Game mode (e.g. "normal") |
| `session` | `level_reached` | no | Highest level completed |
| `session` | `total_time` | no | Total play time in seconds |
| `session` | `cognitive_age` | no | Estimated cognitive age |
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

### `GET /api/v1/app/auth/{uid}`

Login for the Android app. Returns participant data and all their game sessions.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Login successful",
  "data": {
    "participant": {
      "id": 1,
      "uid": "RFID-001",
      "name": "Budi Santoso",
      "age": 10,
      "grade": "5A",
      "gender": "male",
      "height": 135.5,
      "weight": 30.2,
      "heart_rate": 85,
      "spo2": 98.5,
      "grip_strength": 12.3,
      "created_at": "2026-04-12T10:00:00Z"
    },
    "sessions": [
      {
        "id": 1,
        "participant_id": 1,
        "mode": "normal",
        "level_reached": 6,
        "total_time": 18.5,
        "cognitive_age": 11,
        "visuo_spatial_fit": 0.91,
        "dexterity_score": 88.5,
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
One entry per participant (uses `DISTINCT ON`), ranked by `visuo_spatial_fit` descending, then `total_time` ascending as tiebreaker.

**Response `200`:**
```json
{
  "status": "success",
  "message": "Leaderboard fetched successfully",
  "data": [
    {
      "rank": 1,
      "participant_id": 1,
      "uid": "RFID-001",
      "name": "Budi Santoso",
      "grade": "5A",
      "age": 10,
      "visuo_spatial_fit": 0.91,
      "total_time": 18.5,
      "level_reached": 6,
      "dexterity_score": 88.5
    },
    {
      "rank": 2,
      "participant_id": 2,
      "uid": "RFID-002",
      "name": "Ani Lestari",
      "grade": "4B",
      "age": 9,
      "visuo_spatial_fit": 0.85,
      "total_time": 22.3,
      "level_reached": 5,
      "dexterity_score": 82.0
    }
  ]
}
```

Returns empty array `[]` when no sessions have been recorded yet.

---

## 6. Export

### `GET /api/v1/export/excel`

Downloads an Excel (.xlsx) file with 3 sheets:

| Sheet | Contents |
|-------|----------|
| Leaderboard | All ranked participants (best session per person) |
| Participants | All registered participant data |
| Sessions | All game session records |

**Response:** Binary `.xlsx` file download (`Content-Disposition: attachment; filename=oamp-report.xlsx`)

---

### `GET /api/v1/export/pdf`

Downloads a PDF file with the leaderboard table.

**Response:** Binary `.pdf` file download (`Content-Disposition: attachment; filename=oamp-leaderboard.pdf`)

If no sessions exist, the PDF contains the text "No game sessions recorded yet."

---

### `GET /api/v1/export/rapor/{uid}`

Downloads a PDF rapor (report card) for an individual participant.

**URL parameters:**
| Param | Type | Description |
|-------|------|-------------|
| `uid` | string | Participant UID (RFID tag / QR code) |

**Response `200`:** Binary `.pdf` file (`Content-Disposition: attachment; filename=rapor-{name}.pdf`)

**PDF contents:**

| Section | Details |
|---------|---------|
| Header | "Rapor Peserta OAMP" + subtitle |
| Data Pribadi | UID, Kelas, Umur, Jenis Kelamin, Tinggi, Berat, Detak Jantung, SpO2, Grip Strength |
| Riwayat Game | Tabel semua sesi: tanggal, mode, level, waktu, VisuoSpatialFit, Dexterity |
| Ringkasan Performa | Total sesi, skor VisuoSpatial terbaik, level tertinggi, rata-rata waktu |
| Hasil Quiz | Tabel quiz (jika ada): tanggal, skor |
| Footer | Tanggal cetak |

If the participant has no sessions yet, the rapor still generates with participant data only (no session table).

**Response `404`:**
```json
{
  "status": "error",
  "message": "Participant not found",
  "data": null
}
```

**Frontend usage:**
```js
const res = await api.get(`/export/rapor/${uid}`, { responseType: "blob" });
const url = window.URL.createObjectURL(res);
const link = document.createElement("a");
link.href = url;
link.setAttribute("download", `rapor-${uid}.pdf`);
link.click();
```

---

## 7. Data Models

### Participant

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `uid` | string | Unique identifier (RFID tag / QR code) |
| `name` | string | Full name |
| `age` | int | Age in years (3-18) |
| `grade` | string | School grade / class |
| `gender` | string | `male` or `female` |
| `height` | float | Height in cm |
| `weight` | float | Weight in kg |
| `heart_rate` | int | Resting heart rate (bpm) |
| `spo2` | float | Blood oxygen saturation (%) |
| `grip_strength` | float | Grip strength measurement |
| `created_at` | timestamp | Auto-set by GORM |

### GameSession

| Field | Type | Description |
|-------|------|-------------|
| `id` | uint | Auto-generated primary key |
| `participant_id` | uint | Foreign key to Participant |
| `mode` | string | Game mode (e.g. "normal") |
| `level_reached` | int | Highest level completed |
| `total_time` | float | Total play time in seconds |
| `cognitive_age` | int | Estimated cognitive age |
| `visuo_spatial_fit` | float | Visuo-spatial fitness score |
| `dexterity_score` | float | Dexterity score |
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
