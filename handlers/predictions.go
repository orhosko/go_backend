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

	// Calculate total league budget
	totalBudget := float64(0)
	for _, team := range teams {
		totalBudget += float64(team.Budget.Int64)
	}

	// Get current week
	currentWeek, err := repo.GetCurrentWeek(ctx, currentSeason.ID)
	if err != nil {
		return nil, err
	}

	// Calculate total weeks in the season
	totalWeeks := 2 * (len(teams) - 1)
	remainingWeeks := totalWeeks - currentWeek

	// Calculate how far we are into the season (0.0 to 1.0)
	seasonProgress := float64(currentWeek) / float64(totalWeeks)

	// Get standings for all teams
	var teamPredictions []TeamPrediction
	maxPoints := int64(0)
	maxPossiblePoints := make(map[int64]int64)
	currentLeader := int64(0)
	maxCurrentPoints := int64(0)
	leaderRemainingMatches := 0

	// First pass: calculate max points and find current leader
	for _, team := range teams {
		standing, err := repo.GetStanding(ctx, team.ID, currentSeason.ID)
		if err != nil {
			continue
		}

		// Track current leader
		if standing.Points.Int64 > maxCurrentPoints {
			maxCurrentPoints = standing.Points.Int64
			currentLeader = team.ID

			// Get remaining matches for leader
			leaderMatches, err := getRemainingMatches(ctx, repo, team.ID, currentSeason.ID, currentWeek)
			if err == nil {
				leaderRemainingMatches = len(leaderMatches)
			}
		}

		// Calculate remaining matches for this team
		remainingMatches, err := getRemainingMatches(ctx, repo, team.ID, currentSeason.ID, currentWeek)
		if err != nil {
			continue
		}

		// Calculate maximum possible points
		maxPossible := standing.Points.Int64 + (int64(len(remainingMatches)) * 3)
		maxPossiblePoints[team.ID] = maxPossible

		if maxPossible > maxPoints {
			maxPoints = maxPossible
		}
	}

	// Calculate minimum points leader can achieve (assuming they lose all remaining matches)
	minLeaderFinalPoints := maxCurrentPoints

	// Calculate predictions based on current points, remaining matches, and team budget
	totalProbability := 0.0
	for _, team := range teams {
		standing, err := repo.GetStanding(ctx, team.ID, currentSeason.ID)
		if err != nil {
			continue
		}

		// Calculate remaining matches for this team
		remainingMatches, err := getRemainingMatches(ctx, repo, team.ID, currentSeason.ID, currentWeek)
		if err != nil {
			continue
		}

		// Check if team is mathematically eliminated
		maxPossibleTeamPoints := standing.Points.Int64 + (int64(len(remainingMatches)) * 3)

		// Team is eliminated if:
		// 1. They can't reach the current maximum possible points, OR
		// 2. They can't reach the minimum points the leader will have
		if maxPossibleTeamPoints < maxPoints || maxPossibleTeamPoints <= minLeaderFinalPoints {
			teamPredictions = append(teamPredictions, TeamPrediction{
				TeamName:    team.Name,
				Probability: 0,
			})
			continue
		}

		// Calculate base probability using current points and team budget
		currentPoints := float64(standing.Points.Int64)
		maxPossible := float64(maxPossiblePoints[team.ID])
		budgetFactor := float64(team.Budget.Int64) / totalBudget // Team's budget as a proportion of total league budget

		// Weighted factors that change based on season progress
		pointsFactor := currentPoints / float64(maxPoints)
		remainingPotentialFactor := float64(maxPossible-currentPoints) / float64(remainingWeeks*3)

		// Adjust weights based on season progress
		var probability float64
		if seasonProgress >= 0.75 { // Last quarter of the season
			// Give much more weight to current points in the final stages
			pointsWeight := 0.8 + (seasonProgress * 0.2) // Increases from 0.8 to 1.0
			budgetWeight := (1.0 - seasonProgress) * 0.2 // Decreases from 0.2 to 0
			probability = (pointsFactor * pointsWeight) + (budgetFactor * budgetWeight)

			// Boost probability for the current leader in the final weeks
			if team.ID == currentLeader {
				leaderBoost := seasonProgress * 0.4 // Up to 40% boost at the end
				probability *= (1.0 + leaderBoost)
			}

			// If we're in the final weeks (>85% through season), make predictions more decisive
			if seasonProgress > 0.85 {
				probability = math.Pow(probability, 0.5) // Make high probabilities higher
			}
		} else {
			// Earlier in the season, use the regular calculation
			probability = (pointsFactor*0.5 + budgetFactor*0.3 + remainingPotentialFactor*0.2)
		}

		// Add a small random factor that decreases as the season progresses
		randomFactor := 1.0 + ((1.0 - seasonProgress) * math.Sin(float64(team.ID+currentSeason.ID)) * 0.1)
		probability *= randomFactor

		totalProbability += probability

		teamPredictions = append(teamPredictions, TeamPrediction{
			TeamName:    team.Name,
			Probability: probability,
		})
	}

	// Normalize probabilities to sum to 1.0
	if totalProbability > 0 {
		for i := range teamPredictions {
			teamPredictions[i].Probability /= totalProbability
		}

		// In the final weeks, ensure the leader has at least 80% probability if they're clearly ahead
		if seasonProgress >= 0.85 {
			leaderPoints := float64(maxCurrentPoints)
			secondPlace := float64(0)

			// Find second place points
			for _, team := range teams {
				standing, err := repo.GetStanding(ctx, team.ID, currentSeason.ID)
				if err != nil {
					continue
				}
				if float64(standing.Points.Int64) > secondPlace && float64(standing.Points.Int64) < leaderPoints {
					secondPlace = float64(standing.Points.Int64)
				}
			}

			// Calculate points needed to guarantee championship
			pointsToGuarantee := maxCurrentPoints + int64(leaderRemainingMatches*3)

			// If the leader is clearly ahead (more than 6 points) or can clinch mathematically
			pointsGap := leaderPoints - secondPlace
			if pointsGap >= 6 || pointsToGuarantee > maxPoints {
				for i := range teamPredictions {
					if teamPredictions[i].Probability >= 0.5 { // This is likely the leader
						teamPredictions[i].Probability = math.Max(0.8, teamPredictions[i].Probability)

						// Redistribute remaining probability among others
						remainingProb := 1.0 - teamPredictions[i].Probability
						for j := range teamPredictions {
							if i != j && teamPredictions[j].Probability > 0 { // Only redistribute to teams not mathematically eliminated
								teamPredictions[j].Probability *= remainingProb / (1.0 - teamPredictions[i].Probability)
							}
						}
						break
					}
				}
			}
		}
	}

	// Sort by probability in descending order
	sort.Slice(teamPredictions, func(i, j int) bool {
		return teamPredictions[i].Probability > teamPredictions[j].Probability
	})

	return teamPredictions, nil
}

// getRemainingMatches returns the number of remaining matches for a team
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
	GetStanding(ctx context.Context, teamID int64, seasonID int64) (sqlc.Standing, error)
	GetCurrentWeek(ctx context.Context, seasonID int64) (int, error)
	GetUnplayedMatchesByWeek(ctx context.Context, week int64, seasonID int64) ([]sqlc.GetUnplayedMatchesByWeekRow, error)
}
