-- name: CreateFixture :exec
INSERT INTO match (
  home, guest, played, week
) VALUES (
  ?, ?, ?, ?
);

-- name: SaveResult :exec
UPDATE match
SET played = ?
WHERE id = ?;

-- name: ListTeams :many
SELECT * FROM team
ORDER BY name;

-- name: standing :exec
UPDATE standing
SET points = ?,
wins = ?,
draws = ?,
losses = ?,
goal_diff = ?
WHERE id = ?;

-- name: GetTeamStanding :one
SELECT * FROM standing
WHERE id = ?
LIMIT 1;

-- name: CreateTeam :one
INSERT INTO team (
  name, strength
) VALUES (
  ?, ?
)
RETURNING *;

-- name: GetTeam :one
SELECT * FROM team
WHERE id = ?
LIMIT 1;

-- name: GetTeamByName :one
SELECT * FROM team
WHERE name = ?
LIMIT 1;

-- name: UpdateTeamStrenght :exec
UPDATE team
SET strength = ?
WHERE id = ?;

-- name: DeleteTeam :exec
DELETE FROM team
WHERE id = ?;
