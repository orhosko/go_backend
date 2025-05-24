package handlers

import (
	"context"
	"database/sql"

	"github.com/orhosko/go-backend/repository"
	"github.com/orhosko/go-backend/sqlc"
)

// recalculateTeamStanding recalculates the standing for a specific team in a season
func recalculateTeamStanding(ctx context.Context, repo repository.Repository, seasonID, teamID int64) error {
	// Get all teams to calculate total weeks
	teams, err := repo.ListTeams(ctx)
	if err != nil {
		return err
	}

	// Calculate total weeks in the season (each team plays against every other team twice)
	totalWeeks := 2 * (len(teams) - 1)

	// Initialize standings data
	stats := struct {
		points   int64
		wins     int64
		draws    int64
		losses   int64
		goalDiff int64
	}{}

	// Get matches from all weeks
	for week := 1; week <= totalWeeks; week++ {
		matches, err := repo.GetMatchesByWeek(ctx, int64(week), seasonID)
		if err != nil {
			return err
		}

		// Calculate standings based on matches
		for _, match := range matches {
			if !match.Played.Bool {
				continue
			}

			// Skip if team is not involved in this match
			if match.HomeID != teamID && match.GuestID != teamID {
				continue
			}

			result, err := repo.GetMatchResult(ctx, match.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				return err
			}

			if match.HomeID == teamID {
				stats.goalDiff += result.HomeScore - result.GuestScore
				if result.HomeScore == result.GuestScore {
					stats.draws++
					stats.points++
				} else if result.HomeScore > result.GuestScore {
					stats.wins++
					stats.points += 3
				} else {
					stats.losses++
				}
			} else if match.GuestID == teamID {
				stats.goalDiff += result.GuestScore - result.HomeScore
				if result.HomeScore == result.GuestScore {
					stats.draws++
					stats.points++
				} else if result.HomeScore < result.GuestScore {
					stats.wins++
					stats.points += 3
				} else {
					stats.losses++
				}
			}
		}
	}

	// Update team standing
	return repo.UpdateStanding(ctx, sqlc.UpdateStandingParams{
		TeamID:   teamID,
		SeasonID: seasonID,
		Points:   sql.NullInt64{Int64: stats.points, Valid: true},
		Wins:     sql.NullInt64{Int64: stats.wins, Valid: true},
		Draws:    sql.NullInt64{Int64: stats.draws, Valid: true},
		Losses:   sql.NullInt64{Int64: stats.losses, Valid: true},
		GoalDiff: sql.NullInt64{Int64: stats.goalDiff, Valid: true},
	})
}
