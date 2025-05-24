CREATE TABLE season (
    id          INTEGER     PRIMARY KEY,
    year        INTEGER     NOT NULL,
    is_current  BOOLEAN     DEFAULT FALSE,
    is_complete BOOLEAN     DEFAULT FALSE
);

CREATE TABLE team (
    id          INTEGER     PRIMARY KEY,
    name        text        NOT NULL,
    strength    INTEGER,
    budget      INTEGER     DEFAULT 1000000
);

CREATE TABLE standing (
    id          INTEGER     PRIMARY KEY,
    team_id     INTEGER     NOT NULL,
    season_id   INTEGER     NOT NULL,
    points      INTEGER     DEFAULT 0,
    wins        INTEGER     DEFAULT 0,
    draws       INTEGER     DEFAULT 0,
    losses      INTEGER     DEFAULT 0,
    goal_diff   INTEGER     DEFAULT 0,
    FOREIGN KEY (team_id) REFERENCES team(id),
    FOREIGN KEY (season_id) REFERENCES season(id)
);

CREATE TABLE match (
    id          INTEGER     PRIMARY KEY,
    season_id   INTEGER     NOT NULL,
    home_id     INTEGER     NOT NULL,
    guest_id    INTEGER     NOT NULL,
    played      BOOLEAN     DEFAULT FALSE,
    week        INTEGER     NOT NULL,
    FOREIGN KEY (home_id) REFERENCES team(id),
    FOREIGN KEY (guest_id) REFERENCES team(id),
    FOREIGN KEY (season_id) REFERENCES season(id)
);

CREATE TABLE match_result (
    id          INTEGER     PRIMARY KEY,
    match_id    INTEGER     NOT NULL,
    home_score  INTEGER     NOT NULL,
    guest_score INTEGER     NOT NULL,
    winner_id   INTEGER,
    FOREIGN KEY (match_id) REFERENCES match(id),
    FOREIGN KEY (winner_id) REFERENCES team(id)
);

CREATE TABLE game_state (
    id          INTEGER     PRIMARY KEY,
    current_week INTEGER    DEFAULT 1,
    season_id   INTEGER     NOT NULL,
    FOREIGN KEY (season_id) REFERENCES season(id)
);

-- Initialize first season (2025)
INSERT INTO season (year, is_current, is_complete) VALUES (2025, TRUE, FALSE);

-- Initialize game state with week 1 and current season
INSERT INTO game_state (current_week, season_id) 
SELECT 1, id FROM season WHERE is_current = TRUE;

-- Initialize teams with budgets
INSERT INTO team (name, strength, budget) VALUES ('Manchester City', 10, 1000000000);
INSERT INTO team (name, strength, budget) VALUES ('Manchester United', 8, 800000000);
INSERT INTO team (name, strength, budget) VALUES ('Chelsea', 7, 700000000);
INSERT INTO team (name, strength, budget) VALUES ('Arsenal', 6, 600000000);
INSERT INTO team (name, strength, budget) VALUES ('Liverpool', 9, 900000000);

CREATE TABLE teamStats (
    id          INTEGER     PRIMARY KEY,
    team        team,
    value       integer,
    lastSeasonStanding integer
);
