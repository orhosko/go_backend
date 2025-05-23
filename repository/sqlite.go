package repository

import (
	"context"

	"github.com/orhosko/go-backend/sqlc"
)

// SQLCRepository implements the Repository interface using sqlc.
type SQLCRepository struct {
	queries *sqlc.Queries
}

// NewSQLCRepository creates a new SQLCRepository.
func NewSQLCRepository(queries *sqlc.Queries) *SQLCRepository {
	return &SQLCRepository{queries: queries}
}

// --- TeamRepository implementation ---

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

func (r *SQLCRepository) UpdateTeamStrength(ctx context.Context, arg sqlc.UpdateTeamStrenghtParams) error {
	return r.queries.UpdateTeamStrenght(ctx, arg)
}

func (r *SQLCRepository) DeleteTeam(ctx context.Context, id int64) error {
	return r.queries.DeleteTeam(ctx, id)
}

// --- StandingRepository implementation ---

func (r *SQLCRepository) GetStanding(ctx context.Context, teamID int64) (sqlc.Standing, error) {
	return r.queries.GetStanding(ctx, teamID)
}

func (r *SQLCRepository) UpdateStanding(ctx context.Context, arg sqlc.UpdateStandingParams) error {
	return r.queries.UpdateStanding(ctx, arg)
}

// --- MatchRepository implementation ---

func (r *SQLCRepository) CreateFixture(ctx context.Context, arg sqlc.CreateFixtureParams) error {
	return r.queries.CreateFixture(ctx, arg)
}

func (r *SQLCRepository) SaveResult(ctx context.Context, arg sqlc.SaveResultParams) error {
	return r.queries.SaveResult(ctx, arg)
}
