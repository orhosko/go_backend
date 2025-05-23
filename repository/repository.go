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
	UpdateTeamStrength(ctx context.Context, arg sqlc.UpdateTeamStrenghtParams) error
	DeleteTeam(ctx context.Context, id int64) error
}

// StandingRepository defines the interface for standing-related database operations.
type StandingRepository interface {
	GetStanding(ctx context.Context, teamID int64) (sqlc.Standing, error)
	UpdateStanding(ctx context.Context, arg sqlc.UpdateStandingParams) error
}

// MatchRepository defines the interface for match-related database operations.
type MatchRepository interface {
	CreateFixture(ctx context.Context, arg sqlc.CreateFixtureParams) error
	SaveResult(ctx context.Context, arg sqlc.SaveResultParams) error
	// Add methods for listing matches, getting match details etc.
}

// Repository combines all specific repositories into a single interface.
type Repository interface {
	TeamRepository
	StandingRepository
	MatchRepository
	// Add other repositories as needed
}
