package repository

import (
	"context"
	"database/sql"

	"github.com/orhosko/go-backend/sqlc"
)

// SQLCRepository implements the Repository interface using SQLC-generated code.
type SQLCRepository struct {
	queries *sqlc.Queries
	Conn    *sql.DB
}

// NewSQLCRepository creates a new SQLCRepository instance.
func NewSQLCRepository(queries *sqlc.Queries, conn *sql.DB) *SQLCRepository {
	return &SQLCRepository{
		queries: queries,
		Conn:    conn,
	}
}

func (r *SQLCRepository) GetCurrentSeason(ctx context.Context) (sqlc.Season, error) {
	return r.queries.GetCurrentSeason(ctx)
}

func (r *SQLCRepository) CreateNewSeason(ctx context.Context, year int64) (sqlc.Season, error) {
	return r.queries.CreateNewSeason(ctx, year)
}

func (r *SQLCRepository) SetCurrentSeason(ctx context.Context, id int64) error {
	return r.queries.SetCurrentSeason(ctx, id)
}

func (r *SQLCRepository) CompleteSeason(ctx context.Context, id int64) error {
	return r.queries.CompleteSeason(ctx, id)
}

func (r *SQLCRepository) ResetToYear(ctx context.Context, year int64) error {
	// Execute each statement in a transaction
	tx, err := r.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete match results
	if _, err := tx.ExecContext(ctx, "DELETE FROM match_result"); err != nil {
		return err
	}

	// Delete matches
	if _, err := tx.ExecContext(ctx, "DELETE FROM match"); err != nil {
		return err
	}

	// Delete standings
	if _, err := tx.ExecContext(ctx, "DELETE FROM standing"); err != nil {
		return err
	}

	// Delete game state
	if _, err := tx.ExecContext(ctx, "DELETE FROM game_state"); err != nil {
		return err
	}

	// Delete seasons except the target year
	if _, err := tx.ExecContext(ctx, "DELETE FROM season WHERE year != ?", year); err != nil {
		return err
	}

	// Reset the target season
	if _, err := tx.ExecContext(ctx, "UPDATE season SET is_complete = FALSE, is_current = TRUE WHERE year = ?", year); err != nil {
		return err
	}

	// Initialize game state for the reset season
	if _, err := tx.ExecContext(ctx, "INSERT INTO game_state (current_week, season_id) SELECT 1, id FROM season WHERE is_current = TRUE"); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SQLCRepository) GetCurrentWeek(ctx context.Context, seasonID int64) (int, error) {
	week, err := r.queries.GetCurrentWeek(ctx, seasonID)
	if err != nil {
		return 0, err
	}
	return int(week.Int64), nil
}

func (r *SQLCRepository) IncrementWeek(ctx context.Context, seasonID int64) error {
	return r.queries.IncrementWeek(ctx, seasonID)
}

func (r *SQLCRepository) GetAllMatchesPlayedForWeek(ctx context.Context, week int64, seasonID int64) (bool, error) {
	return r.queries.GetAllMatchesPlayedForWeek(ctx, sqlc.GetAllMatchesPlayedForWeekParams{
		Week:     week,
		SeasonID: seasonID,
	})
}

func (r *SQLCRepository) GetMatchesByWeek(ctx context.Context, week int64, seasonID int64) ([]sqlc.GetMatchesByWeekRow, error) {
	return r.queries.GetMatchesByWeek(ctx, sqlc.GetMatchesByWeekParams{
		Week:     week,
		SeasonID: seasonID,
	})
}

func (r *SQLCRepository) GetUnplayedMatchesByWeek(ctx context.Context, week int64, seasonID int64) ([]sqlc.GetUnplayedMatchesByWeekRow, error) {
	return r.queries.GetUnplayedMatchesByWeek(ctx, sqlc.GetUnplayedMatchesByWeekParams{
		Week:     week,
		SeasonID: seasonID,
	})
}

func (r *SQLCRepository) GetStanding(ctx context.Context, teamID int64, seasonID int64) (sqlc.Standing, error) {
	return r.queries.GetStanding(ctx, sqlc.GetStandingParams{
		TeamID:   teamID,
		SeasonID: seasonID,
	})
}

func (r *SQLCRepository) UpdateStanding(ctx context.Context, arg sqlc.UpdateStandingParams) error {
	return r.queries.UpdateStanding(ctx, arg)
}

func (r *SQLCRepository) CreateStanding(ctx context.Context, arg sqlc.CreateStandingParams) error {
	return r.queries.CreateStanding(ctx, arg)
}

func (r *SQLCRepository) CreateFixture(ctx context.Context, arg sqlc.CreateFixtureParams) error {
	return r.queries.CreateFixture(ctx, arg)
}

func (r *SQLCRepository) SaveResult(ctx context.Context, arg sqlc.SaveResultParams) error {
	return r.queries.SaveResult(ctx, arg)
}

func (r *SQLCRepository) GetMatchResult(ctx context.Context, matchID int64) (sqlc.GetMatchResultRow, error) {
	return r.queries.GetMatchResult(ctx, matchID)
}

func (r *SQLCRepository) MarkMatchAsPlayed(ctx context.Context, id int64) error {
	return r.queries.MarkMatchAsPlayed(ctx, id)
}

func (r *SQLCRepository) CreateTeam(ctx context.Context, arg sqlc.CreateTeamParams) (sqlc.Team, error) {
	return r.queries.CreateTeam(ctx, arg)
}

func (r *SQLCRepository) GetTeam(ctx context.Context, id int64) (sqlc.Team, error) {
	return r.queries.GetTeam(ctx, id)
}

func (r *SQLCRepository) GetTeamByName(ctx context.Context, name string) (sqlc.Team, error) {
	return r.queries.GetTeamByName(ctx, name)
}

func (r *SQLCRepository) ListTeams(ctx context.Context) ([]sqlc.Team, error) {
	return r.queries.ListTeams(ctx)
}

func (r *SQLCRepository) UpdateTeamStrength(ctx context.Context, arg sqlc.UpdateTeamStrengthParams) error {
	return r.queries.UpdateTeamStrength(ctx, arg)
}

func (r *SQLCRepository) DeleteTeam(ctx context.Context, id int64) error {
	// Implement if needed
	return nil
}

func (r *SQLCRepository) InitializeGameState(ctx context.Context, seasonID int64) error {
	_, err := r.Conn.ExecContext(ctx, "INSERT INTO game_state (current_week, season_id) VALUES (1, ?)", seasonID)
	return err
}
