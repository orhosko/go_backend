package repository

import (
	"context"

	"github.com/orhosko/go-backend/sqlc"
)

// TeamRepository defines the interface for team-related database operations.
type TeamRepository interface {
	CreateTeam(ctx context.Context, arg sqlc.CreateTeamParams) (sqlc.Team, error)
	GetTeam(ctx context.Context, id int64) (sqlc.Team, error)
	GetTeamByName(ctx context.Context, name string) (sqlc.Team, error)
	ListTeams(ctx context.Context) ([]sqlc.Team, error)
	UpdateTeamStrength(ctx context.Context, arg sqlc.UpdateTeamStrengthParams) error
	DeleteTeam(ctx context.Context, id int64) error
}

// StandingRepository defines the interface for standing-related database operations.
type StandingRepository interface {
	GetStanding(ctx context.Context, teamID int64, seasonID int64) (sqlc.Standing, error)
	UpdateStanding(ctx context.Context, arg sqlc.UpdateStandingParams) error
	CreateStanding(ctx context.Context, arg sqlc.CreateStandingParams) error
}

// MatchRepository defines the interface for match-related database operations.
type MatchRepository interface {
	CreateFixture(ctx context.Context, arg sqlc.CreateFixtureParams) error
	SaveResult(ctx context.Context, arg sqlc.SaveResultParams) error
	GetMatchesByWeek(ctx context.Context, week int64, seasonID int64) ([]sqlc.GetMatchesByWeekRow, error)
	GetUnplayedMatchesByWeek(ctx context.Context, week int64, seasonID int64) ([]sqlc.GetUnplayedMatchesByWeekRow, error)
	GetMatchResult(ctx context.Context, matchID int64) (sqlc.GetMatchResultRow, error)
	MarkMatchAsPlayed(ctx context.Context, id int64) error
	GetCurrentWeek(ctx context.Context, seasonID int64) (int, error)
	IncrementWeek(ctx context.Context, seasonID int64) error
	GetAllMatchesPlayedForWeek(ctx context.Context, week int64, seasonID int64) (bool, error)
}

// SeasonRepository defines the interface for season-related database operations.
type SeasonRepository interface {
	GetCurrentSeason(ctx context.Context) (sqlc.Season, error)
	CreateNewSeason(ctx context.Context, year int64) (sqlc.Season, error)
	SetCurrentSeason(ctx context.Context, id int64) error
	CompleteSeason(ctx context.Context, id int64) error
	ResetToYear(ctx context.Context, year int64) error
	InitializeGameState(ctx context.Context, seasonID int64) error
}

// Repository combines all repository interfaces.
type Repository interface {
	TeamRepository
	StandingRepository
	MatchRepository
	SeasonRepository
}
