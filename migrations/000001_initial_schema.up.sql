CREATE TABLE IF NOT EXISTS event_batches (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS participants (
    id SERIAL PRIMARY KEY,
    uid VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    age INTEGER NOT NULL,
    grade VARCHAR(30) NOT NULL,
    gender VARCHAR(10) NOT NULL,
    height DOUBLE PRECISION NOT NULL,
    weight DOUBLE PRECISION NOT NULL,
    heart_rate INTEGER,
    spo2 DOUBLE PRECISION,
    grip_strength DOUBLE PRECISION,
    is_premium BOOLEAN DEFAULT FALSE,
    ai_analysis TEXT,
    ai_analysis_updated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS game_sessions (
    id SERIAL PRIMARY KEY,
    participant_id INTEGER NOT NULL,
    event_batch_id INTEGER NOT NULL DEFAULT 1,
    mode VARCHAR(20),
    level_reached INTEGER,
    total_time DOUBLE PRECISION,
    cognitive_age INTEGER,
    visuo_spatial_fit DOUBLE PRECISION,
    dexterity_score DOUBLE PRECISION,
    score DOUBLE PRECISION DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_game_sessions_participant FOREIGN KEY (participant_id) REFERENCES participants(id) ON DELETE CASCADE,
    CONSTRAINT fk_game_sessions_event_batch FOREIGN KEY (event_batch_id) REFERENCES event_batches(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_game_sessions_participant_batch ON game_sessions(participant_id, event_batch_id);

CREATE TABLE IF NOT EXISTS rooms (
    id VARCHAR(4) PRIMARY KEY,
    status VARCHAR(20) DEFAULT 'waiting',
    player1_name VARCHAR(100),
    player2_name VARCHAR(100),
    player1_ready BOOLEAN DEFAULT FALSE,
    player2_ready BOOLEAN DEFAULT FALSE,
    last_activity TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS player_states (
    id SERIAL PRIMARY KEY,
    room_id VARCHAR(4) NOT NULL,
    player_name VARCHAR(100) NOT NULL,
    current_level INTEGER DEFAULT 0,
    elapsed_time DOUBLE PRECISION DEFAULT 0,
    completed_levels INTEGER DEFAULT 0,
    level_times JSONB DEFAULT '[]',
    is_finished BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_player_states_room FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_player_states_room_id ON player_states(room_id);

CREATE TABLE IF NOT EXISTS game_results (
    uid VARCHAR(255) PRIMARY KEY,
    mode VARCHAR(20) DEFAULT 'training',
    nick_name VARCHAR(100),
    gender VARCHAR(10),
    age INTEGER,
    task01 DOUBLE PRECISION,
    task02 DOUBLE PRECISION,
    task03 DOUBLE PRECISION,
    task04 DOUBLE PRECISION,
    task05 DOUBLE PRECISION,
    task06 DOUBLE PRECISION,
    task07 DOUBLE PRECISION,
    task08 DOUBLE PRECISION,
    task_avg DOUBLE PRECISION,
    cognitive_age DOUBLE PRECISION,
    visuo_spatial DOUBLE PRECISION,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tournaments (
    id SERIAL PRIMARY KEY,
    event_batch_id INTEGER NOT NULL DEFAULT 1,
    name VARCHAR(100) NOT NULL,
    status VARCHAR(20) DEFAULT 'registration',
    max_players INTEGER DEFAULT 8,
    player_count INTEGER DEFAULT 0,
    current_round INTEGER DEFAULT 0,
    current_match_id INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_tournaments_event_batch FOREIGN KEY (event_batch_id) REFERENCES event_batches(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tournaments_event_batch ON tournaments(event_batch_id);

CREATE TABLE IF NOT EXISTS tournament_players (
    id SERIAL PRIMARY KEY,
    tournament_id INTEGER NOT NULL,
    participant_id INTEGER NOT NULL,
    name VARCHAR(100),
    seed INTEGER DEFAULT 0,
    rank INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_tournament_players_tournament FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    CONSTRAINT fk_tournament_players_participant FOREIGN KEY (participant_id) REFERENCES participants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tournament_players_tournament ON tournament_players(tournament_id);
CREATE INDEX IF NOT EXISTS idx_tournament_players_participant ON tournament_players(participant_id);

CREATE TABLE IF NOT EXISTS tournament_matches (
    id SERIAL PRIMARY KEY,
    tournament_id INTEGER NOT NULL,
    round INTEGER NOT NULL,
    match_number INTEGER NOT NULL,
    player1_id INTEGER,
    player2_id INTEGER,
    player1_name VARCHAR(100),
    player2_name VARCHAR(100),
    player1_score DOUBLE PRECISION DEFAULT 0,
    player2_score DOUBLE PRECISION DEFAULT 0,
    winner_id INTEGER,
    status VARCHAR(20) DEFAULT 'scheduled',
    parent_match_id INTEGER,
    parent_slot INTEGER,
    room_id VARCHAR(4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_tournament_matches_tournament FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tournament_matches_tournament ON tournament_matches(tournament_id);
CREATE INDEX IF NOT EXISTS idx_tournament_matches_round ON tournament_matches(tournament_id, round, match_number);

-- Insert default event batch
INSERT INTO event_batches (name, is_active) VALUES ('Sesi Default', TRUE) ON CONFLICT DO NOTHING;
