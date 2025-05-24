-- name: CreateFixture :exec
INSERT INTO match (
  home_id, guest_id, played, week, season_id
) VALUES (
  ?, ?, ?, ?, ?
);

-- name: SaveResult :exec
INSERT INTO match_result (
  match_id, home_score, guest_score, winner_id
) VALUES (
  ?, ?, ?, ?
)
ON CONFLICT(match_id) DO UPDATE SET
  home_score = excluded.home_score,
  guest_score = excluded.guest_score,
  winner_id = excluded.winner_id;

-- name: ListTeams :many
SELECT * FROM team
ORDER BY name;

-- name: GetStanding :one
SELECT * FROM standing
WHERE team_id = ? AND season_id = ?
LIMIT 1;

-- name: UpdateStanding :exec
UPDATE standing
SET points = ?,
    wins = ?,
    draws = ?,
    losses = ?,
    goal_diff = ?
WHERE team_id = ? AND season_id = ?;

-- name: GetTeam :one
SELECT * FROM team
WHERE id = ?
LIMIT 1;

-- name: GetTeamByName :one
SELECT * FROM team
WHERE name = ?
LIMIT 1;

-- name: CreateTeam :one
INSERT INTO team (
  name, strength, budget
) VALUES (
  ?, ?, ?
)
RETURNING *;

-- name: UpdateTeamStrength :exec
UPDATE team
SET strength = ?
WHERE id = ?;

-- name: DeleteTeam :exec
DELETE FROM team
WHERE id = ?;

-- name: GetMatchesByWeek :many
SELECT m.*, 
       ht.name as home_team_name, 
       gt.name as guest_team_name,
       ht.strength as home_team_strength,
       gt.strength as guest_team_strength
FROM match m
JOIN team ht ON m.home_id = ht.id
JOIN team gt ON m.guest_id = gt.id
WHERE m.week = ? AND m.season_id = ?;

-- name: GetMatchResult :one
SELECT mr.*, 
       ht.name as home_team_name, 
       gt.name as guest_team_name
FROM match_result mr
JOIN match m ON mr.match_id = m.id
JOIN team ht ON m.home_id = ht.id
JOIN team gt ON m.guest_id = gt.id
WHERE mr.match_id = ?;

-- name: GetUnplayedMatchesByWeek :many
SELECT m.*, 
       ht.name as home_team_name, 
       gt.name as guest_team_name,
       ht.strength as home_team_strength,
       gt.strength as guest_team_strength
FROM match m
JOIN team ht ON m.home_id = ht.id
JOIN team gt ON m.guest_id = gt.id
WHERE m.week = ? AND m.played = FALSE AND m.season_id = ?;

-- name: MarkMatchAsPlayed :exec
UPDATE match SET played = TRUE WHERE id = ?;

-- name: CreateStanding :exec
INSERT INTO standing (
  team_id, season_id, points, wins, draws, losses, goal_diff
) VALUES (
  ?, ?, ?, ?, ?, ?, ?
);

-- name: GetCurrentWeek :one
SELECT current_week FROM game_state WHERE season_id = ? LIMIT 1;

-- name: IncrementWeek :exec
UPDATE game_state SET current_week = current_week + 1 WHERE season_id = ?;

-- name: GetAllMatchesPlayedForWeek :one
SELECT COUNT(*) = 0 as all_played FROM match WHERE week = ? AND played = FALSE AND season_id = ?;

-- name: GetCurrentSeason :one
SELECT * FROM season WHERE is_current = TRUE LIMIT 1;

-- name: CreateNewSeason :one
INSERT INTO season (year, is_current, is_complete) VALUES (?, FALSE, FALSE) RETURNING *;

-- name: SetCurrentSeason :exec
UPDATE season SET is_current = (season.id = ?) WHERE season.id IN (SELECT id FROM season);

-- name: CompleteSeason :exec
UPDATE season SET is_complete = TRUE WHERE id = ?;

-- name: ResetToYear :exec
DELETE FROM match_result;
DELETE FROM match;
DELETE FROM standing;
DELETE FROM game_state;
DELETE FROM season WHERE year != ?;
UPDATE season SET is_complete = FALSE, is_current = TRUE WHERE year = ?;
INSERT INTO game_state (current_week, season_id) SELECT 1, id FROM season WHERE is_current = TRUE;
