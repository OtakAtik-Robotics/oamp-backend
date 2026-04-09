# OAMP Backend 

> REST API and Database service for the Block Design Test (BDT) cognitive research project. 
> Maintained by the Backend Developers of OtakAtik-Robotics.

This repository serves as the central hub for the OAMP project, handling participant registration, cognitive evaluation data processing from the robot's AI, and serving quiz data to the mobile application.

## 🛠️ Tech Stack
* **Language:** Golang (Go)
* **Framework:** Gin Web Framework
* **ORM:** GORM
* **Database:** PostgreSQL
* **Environment:** Godotenv

##  Project Structure
Following Standard Go Project Layout:
* `cmd/api/` - Main application entry point
* `internal/config/` - Database and environment configurations
* `internal/handler/` - HTTP route handlers (Controllers)
* `internal/model/` - GORM database schemas

##  Getting Started

### Prerequisites
* Go (1.20 or newer)
* PostgreSQL database server

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/OtakAtik-Robotics/oamp-backend.git
   cd oamp-backend
    ```
2. Install dependencies:
    ```bash
    go mod tidy
    ```
3. Create a .env file in the root directory based on .env.example:
    ```bash
    DB_HOST=localhost
    DB_USER=postgres
    DB_PASSWORD=yourpassword
    DB_NAME=bdt_db
    DB_PORT=5432
    PORT=8080
    ```
4. Run the development server:
    ```bash
    go run cmd/api/main.go
    ```

## API Overview
Base URL: `http://localhost:8080/api/v1`

### Hardware / Registration

* `POST /participants` - Register new participant
* `GET /participants/:uid` - Fetch participant data (used by Robot for height auto-adjustment)

### Robot AI

* `POST /sessions` - Submit game session results (Cognitive score, dexterity, face expressions)

### Mobile App

* `GET /mobile/results/:uid` - Fetch participant and game session data

* `POST /mobile/quiz` - Submit post-game quiz score