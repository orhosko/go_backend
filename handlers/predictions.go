package handlers

import (
	"context"
	"math"
	"sort"

	"github.com/orhosko/go-backend/sqlc"
)

// TeamPrediction represents a team's championship prediction percentage.
type TeamPrediction struct {
	TeamName    string
	Probability float64 // e.g., 0.60 for 60%
}

// calculateChampionshipPredictions calculates the probability of each team winning the championship
func calculateChampionshipPredictions(ctx context.Context, repo Repository, currentSeason sqlc.Season) ([]TeamPrediction, error) {
	// Get all teams and their standings
	teams, err := repo.ListTeams(ctx)
	if err != nil {
		return nil, err
	}

	// Get current week
	currentWeek, err := repo.GetCurrentWeek(ctx, currentSeason.ID)
	if err != nil {
		return nil, err
	}

	// Calculate total weeks in the season
	totalWeeks := 2 * (len(teams) - 1)
	remainingWeeks := totalWeeks - currentWeek

	// Find current leader and their points
	var maxCurrentPoints int64
	var currentLeader int64
	for _, team := range teams {
		standing, err := repo.GetStanding(ctx, team.ID, currentSeason.ID)
		if err != nil {
			continue
		}
		if standing.Points.Int64 > maxCurrentPoints {
			maxCurrentPoints = standing.Points.Int64
			currentLeader = team.ID
		}
	}

	// Calculate predictions for each team
	var teamPredictions []TeamPrediction
	for _, team := range teams {
		standing, err := repo.GetStanding(ctx, team.ID, currentSeason.ID)
		if err != nil {
			continue
		}

		// Get remaining matches for this team
		remainingMatches, err := getRemainingMatches(ctx, repo, team.ID, currentSeason.ID, currentWeek)
		if err != nil {
			continue
		}

		// If no remaining matches and not in first place, probability is 0
		if len(remainingMatches) == 0 {
			if team.ID == currentLeader {
				teamPredictions = append(teamPredictions, TeamPrediction{
					TeamName:    team.Name,
					Probability: 1.0, // 100% chance for the leader when season is complete
				})
			} else {
				teamPredictions = append(teamPredictions, TeamPrediction{
					TeamName:    team.Name,
					Probability: 0.0,
				})
			}
			continue
		}

		// Calculate maximum possible points for this team
		maxPossiblePoints := standing.Points.Int64 + (int64(len(remainingMatches)) * 3)

		// If team can't mathematically catch up to the leader, probability is 0
		if maxPossiblePoints < maxCurrentPoints {
			teamPredictions = append(teamPredictions, TeamPrediction{
				TeamName:    team.Name,
				Probability: 0.0,
			})
			continue
		}

		// Calculate team strength based on budget and current standing
		budgetStrength := float64(team.Budget.Int64) / 1000000.0 // Normalize budget
		standingStrength := 0.0
		if currentWeek == 0 {
			standingStrength = 0.5
		} else {
			standingStrength = float64(standing.Points.Int64) / float64(currentWeek*3) // Points per available match
		}
		teamStrength := (budgetStrength + standingStrength) / 2.0

		// Calculate average opponent strength for remaining matches
		totalOpponentStrength := 0.0
		for _, match := range remainingMatches {
			var opponentID int64
			if match.HomeID == team.ID {
				opponentID = match.GuestID
			} else {
				opponentID = match.HomeID
			}

			// Get opponent's standing
			opponentStanding, err := repo.GetStanding(ctx, opponentID, currentSeason.ID)
			if err != nil {
				continue
			}

			// Get opponent's team info for budget
			opponent, err := repo.GetTeam(ctx, opponentID)
			if err != nil {
				continue
			}

			// Calculate opponent strength
			opponentBudgetStrength := float64(opponent.Budget.Int64) / 1000000.0
			opponentStandingStrength := float64(opponentStanding.Points.Int64) / float64(currentWeek*3)
			totalOpponentStrength += (opponentBudgetStrength + opponentStandingStrength) / 2.0
		}
		// averageOpponentStrength := totalOpponentStrength / float64(len(teams))

		// Calculate base probability
		probability := teamStrength //* (1.0 + (teamStrength - averageOpponentStrength))

		// Calculate points gap to leader
		pointsGap := float64(maxCurrentPoints - standing.Points.Int64)

		// Calculate season progress (0 to 1)
		seasonProgress := float64(currentWeek) / float64(totalWeeks)

		// Early season: More weight to team strength
		// Late season: More weight to current points and remaining matches
		if seasonProgress < 0.3 { // First 30% of season
			// Early season: Focus on team strength, keep probabilities close
			probability = 0.3 + (probability * 0.4) // Base 30% chance + up to 40% more
		} else if seasonProgress < 0.7 { // Middle 40% of season
			// Mid season: Points start to matter more
			pointsFactor := math.Max(0, 1.0-(pointsGap/float64(currentWeek*3)))
			probability *= (0.6 + (0.4 * pointsFactor))
		} else { // Last 30% of season
			// Late season: Points gap and remaining matches are crucial
			maxPossibleGain := float64(len(remainingMatches) * 3)
			if maxPossibleGain < pointsGap {
				probability *= 0.1 // Virtually impossible to win
			} else {
				catchupFactor := (maxPossibleGain - pointsGap) / maxPossibleGain
				probability *= math.Pow(catchupFactor, 2) // Square it to make it more extreme
			}

			// If it's the last few weeks, make it even more extreme
			if remainingWeeks <= 2 {
				probability = math.Pow(probability, float64(3-remainingWeeks))
			}
		}

		teamPredictions = append(teamPredictions, TeamPrediction{
			TeamName:    team.Name,
			Probability: probability,
		})
	}

	// Normalize probabilities to sum to 1.0
	totalProbability := 0.0
	for _, pred := range teamPredictions {
		totalProbability += pred.Probability
	}

	if totalProbability > 0 {
		for i := range teamPredictions {
			teamPredictions[i].Probability /= totalProbability
		}
	}

	// Sort by probability in descending order
	sort.Slice(teamPredictions, func(i, j int) bool {
		return teamPredictions[i].Probability > teamPredictions[j].Probability
	})

	return teamPredictions, nil
}

// getRemainingMatches returns the remaining matches for a team
func getRemainingMatches(ctx context.Context, repo Repository, teamID int64, seasonID int64, currentWeek int) ([]sqlc.GetUnplayedMatchesByWeekRow, error) {
	var allRemainingMatches []sqlc.GetUnplayedMatchesByWeekRow
	teams, err := repo.ListTeams(ctx)
	if err != nil {
		return nil, err
	}

	totalWeeks := 2 * (len(teams) - 1)

	// Get remaining matches for each week
	for week := currentWeek; week <= totalWeeks; week++ {
		matches, err := repo.GetUnplayedMatchesByWeek(ctx, int64(week), seasonID)
		if err != nil {
			continue
		}

		// Filter matches for the specific team
		for _, match := range matches {
			if match.HomeID == teamID || match.GuestID == teamID {
				allRemainingMatches = append(allRemainingMatches, match)
			}
		}
	}

	return allRemainingMatches, nil
}

// Repository interface for predictions
type Repository interface {
	ListTeams(ctx context.Context) ([]sqlc.Team, error)
	GetTeam(ctx context.Context, id int64) (sqlc.Team, error)
	GetStanding(ctx context.Context, teamID int64, seasonID int64) (sqlc.Standing, error)
	GetCurrentWeek(ctx context.Context, seasonID int64) (int, error)
	GetUnplayedMatchesByWeek(ctx context.Context, week int64, seasonID int64) ([]sqlc.GetUnplayedMatchesByWeekRow, error)
}
