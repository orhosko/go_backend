package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/orhosko/go-backend/repository"
	"github.com/orhosko/go-backend/sqlc"
)

// RegisterFixtureRoutes registers all fixture related routes
func RegisterFixtureRoutes(router *gin.Engine, repo repository.Repository) {
	router.POST("/generate-fixtures", handleGenerateFixtures(repo))
	router.POST("/play-week", handlePlayWeek(repo))
	router.POST("/next-week", handleNextWeek(repo))
	router.POST("/play-all", handlePlayAll(repo))
}

func handleGenerateFixtures(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Check if fixtures already exist for the current season
		currentSeason, err := repo.GetCurrentSeason(reqCtx)
		if err != nil {
			if err == sql.ErrNoRows {
				// If no season exists, create one starting from 2025
				currentSeason, err = repo.CreateNewSeason(reqCtx, 2025)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create initial season"})
					return
				}
				// Set it as the current season
				err = repo.SetCurrentSeason(reqCtx, currentSeason.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set current season"})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current season"})
				return
			}
		}

		// Get current week
		currentWeek, err := repo.GetCurrentWeek(reqCtx, currentSeason.ID)
		if err != nil {
			currentWeek = 1 // Default to week 1 if not set
		}

		// Check if there are any existing matches for this week
		matches, err := repo.GetMatchesByWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing fixtures"})
			return
		}

		// Only generate new fixtures if none exist for the current week
		if err == sql.ErrNoRows || len(matches) == 0 {
			err = generateRoundRobinFixtures(repo, reqCtx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate fixtures"})
				return
			}
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}

func handlePlayWeek(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Get current season
		currentSeason, err := repo.GetCurrentSeason(reqCtx)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{"error": "No active season. Please generate fixtures first."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current season"})
			return
		}

		currentWeek, err := repo.GetCurrentWeek(reqCtx, currentSeason.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current week"})
			return
		}

		// Get unplayed matches for current week
		matches, err := repo.GetUnplayedMatchesByWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
			return
		}

		if err == sql.ErrNoRows || len(matches) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No matches available to play. Please generate fixtures first."})
			return
		}

		// Play each match
		for _, match := range matches {
			log.Printf("Playing match: Home(%s) vs Guest(%s)", match.HomeTeamName, match.GuestTeamName)

			// Calculate win probabilities based on team strengths
			homeStrength := float64(match.HomeTeamStrength.Int64)
			guestStrength := float64(match.GuestTeamStrength.Int64)
			totalStrength := homeStrength + guestStrength

			log.Printf("Team strengths - Home: %.2f, Guest: %.2f", homeStrength, guestStrength)

			// Add some randomness
			randomFactor := rand.Float64() // 0.0 to 1.0

			// Home team has a slight advantage (1.1x)
			homeWinProb := (homeStrength * 1.1) / totalStrength

			var homeScore, guestScore int64
			var winnerID sql.NullInt64

			if randomFactor < homeWinProb {
				// Home team wins
				homeScore = 1 + rand.Int63n(3)      // 1-3 goals
				guestScore = rand.Int63n(homeScore) // 0 to homeScore-1 goals
				winnerID = sql.NullInt64{Int64: match.HomeID, Valid: true}
				log.Printf("Home team wins! Score: %d-%d", homeScore, guestScore)
			} else if randomFactor < homeWinProb+((1-homeWinProb)/2) {
				// Draw
				homeScore = rand.Int63n(3) // 0-2 goals
				guestScore = homeScore
				winnerID = sql.NullInt64{Valid: false}
				log.Printf("Draw! Score: %d-%d", homeScore, guestScore)
			} else {
				// Guest team wins
				guestScore = 1 + rand.Int63n(3)     // 1-3 goals
				homeScore = rand.Int63n(guestScore) // 0 to guestScore-1 goals
				winnerID = sql.NullInt64{Int64: match.GuestID, Valid: true}
				log.Printf("Guest team wins! Score: %d-%d", homeScore, guestScore)
			}

			// Save match result
			err = repo.SaveResult(reqCtx, sqlc.SaveResultParams{
				MatchID:    match.ID,
				HomeScore:  homeScore,
				GuestScore: guestScore,
				WinnerID:   winnerID,
			})
			if err != nil {
				log.Printf("Failed to save match result: %v", err)
				continue
			}
			log.Printf("Match result saved successfully")

			// Mark match as played
			err = repo.MarkMatchAsPlayed(reqCtx, match.ID)
			if err != nil {
				log.Printf("Failed to mark match as played: %v", err)
				continue
			}
			log.Printf("Match marked as played")

			// Get or create standings for both teams
			homeStanding, err := repo.GetStanding(reqCtx, match.HomeID, currentSeason.ID)
			if err != nil && err != sql.ErrNoRows {
				log.Printf("Failed to get home team standing: %v", err)
				continue
			}
			if err == sql.ErrNoRows {
				log.Printf("Creating new standing for home team %d", match.HomeID)
				homeStanding = sqlc.Standing{
					TeamID:   match.HomeID,
					SeasonID: currentSeason.ID,
					Points:   sql.NullInt64{Int64: 0, Valid: true},
					Wins:     sql.NullInt64{Int64: 0, Valid: true},
					Draws:    sql.NullInt64{Int64: 0, Valid: true},
					Losses:   sql.NullInt64{Int64: 0, Valid: true},
					GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
				}
				err = repo.CreateStanding(reqCtx, sqlc.CreateStandingParams{
					TeamID:   match.HomeID,
					SeasonID: currentSeason.ID,
					Points:   sql.NullInt64{Int64: 0, Valid: true},
					Wins:     sql.NullInt64{Int64: 0, Valid: true},
					Draws:    sql.NullInt64{Int64: 0, Valid: true},
					Losses:   sql.NullInt64{Int64: 0, Valid: true},
					GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
				})
				if err != nil {
					log.Printf("Failed to create home team standing: %v", err)
					continue
				}
				log.Printf("Created new standing for home team")
			}

			guestStanding, err := repo.GetStanding(reqCtx, match.GuestID, currentSeason.ID)
			if err != nil && err != sql.ErrNoRows {
				log.Printf("Failed to get guest team standing: %v", err)
				continue
			}
			if err == sql.ErrNoRows {
				log.Printf("Creating new standing for guest team %d", match.GuestID)
				guestStanding = sqlc.Standing{
					TeamID:   match.GuestID,
					SeasonID: currentSeason.ID,
					Points:   sql.NullInt64{Int64: 0, Valid: true},
					Wins:     sql.NullInt64{Int64: 0, Valid: true},
					Draws:    sql.NullInt64{Int64: 0, Valid: true},
					Losses:   sql.NullInt64{Int64: 0, Valid: true},
					GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
				}
				err = repo.CreateStanding(reqCtx, sqlc.CreateStandingParams{
					TeamID:   match.GuestID,
					SeasonID: currentSeason.ID,
					Points:   sql.NullInt64{Int64: 0, Valid: true},
					Wins:     sql.NullInt64{Int64: 0, Valid: true},
					Draws:    sql.NullInt64{Int64: 0, Valid: true},
					Losses:   sql.NullInt64{Int64: 0, Valid: true},
					GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
				})
				if err != nil {
					log.Printf("Failed to create guest team standing: %v", err)
					continue
				}
				log.Printf("Created new standing for guest team")
			}

			// Update standings based on match result
			if homeScore == guestScore {
				// Draw
				homeStanding.Points.Int64++
				homeStanding.Draws.Int64++
				guestStanding.Points.Int64++
				guestStanding.Draws.Int64++
			} else if homeScore > guestScore {
				// Home win
				homeStanding.Points.Int64 += 3
				homeStanding.Wins.Int64++
				guestStanding.Losses.Int64++
			} else {
				// Guest win
				guestStanding.Points.Int64 += 3
				guestStanding.Wins.Int64++
				homeStanding.Losses.Int64++
			}

			// Update goal differences
			homeStanding.GoalDiff.Int64 += homeScore - guestScore
			guestStanding.GoalDiff.Int64 += guestScore - homeScore

			// Save updated standings
			err = repo.UpdateStanding(reqCtx, sqlc.UpdateStandingParams{
				TeamID:   homeStanding.TeamID,
				SeasonID: currentSeason.ID,
				Points:   homeStanding.Points,
				Wins:     homeStanding.Wins,
				Draws:    homeStanding.Draws,
				Losses:   homeStanding.Losses,
				GoalDiff: homeStanding.GoalDiff,
			})
			if err != nil {
				log.Printf("Failed to update home team standing: %v", err)
				continue
			}

			err = repo.UpdateStanding(reqCtx, sqlc.UpdateStandingParams{
				TeamID:   guestStanding.TeamID,
				SeasonID: currentSeason.ID,
				Points:   guestStanding.Points,
				Wins:     guestStanding.Wins,
				Draws:    guestStanding.Draws,
				Losses:   guestStanding.Losses,
				GoalDiff: guestStanding.GoalDiff,
			})
			if err != nil {
				log.Printf("Failed to update guest team standing: %v", err)
				continue
			}
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}

func handleNextWeek(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Get current season
		currentSeason, err := repo.GetCurrentSeason(reqCtx)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{"error": "No active season. Please generate fixtures first."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current season"})
			return
		}

		// Check if all matches for current week are played
		currentWeek, err := repo.GetCurrentWeek(reqCtx, currentSeason.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current week"})
			return
		}

		// Check if there are any matches for the current week
		matches, err := repo.GetMatchesByWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check matches status"})
			return
		}

		if err == sql.ErrNoRows || len(matches) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No matches found for current week. Please generate fixtures first."})
			return
		}

		allPlayed, err := repo.GetAllMatchesPlayedForWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check matches status"})
			return
		}

		if !allPlayed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot proceed to next week until all matches are played"})
			return
		}

		// Check if this is the last week
		teams, err := repo.ListTeams(reqCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}
		totalWeeks := 2 * (len(teams) - 1)

		if currentWeek >= totalWeeks {
			c.JSON(http.StatusOK, gin.H{"message": "Season is complete!"})
			return
		}

		// Increment the week
		err = repo.IncrementWeek(reqCtx, currentSeason.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment week"})
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}

func handlePlayAll(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Get current season
		currentSeason, err := repo.GetCurrentSeason(reqCtx)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{"error": "No active season"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current season"})
			return
		}

		// Get teams to calculate total weeks
		teams, err := repo.ListTeams(reqCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}

		totalWeeks := 2 * (len(teams) - 1)
		currentWeek, err := repo.GetCurrentWeek(reqCtx, currentSeason.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current week"})
			return
		}

		// Play all remaining weeks
		for week := currentWeek; week <= totalWeeks; week++ {
			// Get unplayed matches for current week
			matches, err := repo.GetUnplayedMatchesByWeek(reqCtx, int64(week), currentSeason.ID)
			if err != nil && err != sql.ErrNoRows {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
				return
			}

			// Play each match
			for _, match := range matches {
				// Calculate scores based on team strengths
				homeStrength := match.HomeTeamStrength.Int64
				guestStrength := match.GuestTeamStrength.Int64

				// Add some randomness to the scores
				homeScore := (homeStrength + int64(rand.Intn(30))) / 20
				guestScore := (guestStrength + int64(rand.Intn(30))) / 20

				// Save match result
				var winnerID sql.NullInt64
				if homeScore > guestScore {
					winnerID = sql.NullInt64{Int64: match.HomeID, Valid: true}
				} else if guestScore > homeScore {
					winnerID = sql.NullInt64{Int64: match.GuestID, Valid: true}
				}

				err = repo.SaveResult(reqCtx, sqlc.SaveResultParams{
					MatchID:    match.ID,
					HomeScore:  homeScore,
					GuestScore: guestScore,
					WinnerID:   winnerID,
				})
				if err != nil {
					log.Printf("Failed to save match result: %v", err)
					continue
				}

				// Mark match as played
				err = repo.MarkMatchAsPlayed(reqCtx, match.ID)
				if err != nil {
					log.Printf("Failed to mark match as played: %v", err)
					continue
				}

				// Update standings
				homeStanding, err := repo.GetStanding(reqCtx, match.HomeID, currentSeason.ID)
				if err != nil && err != sql.ErrNoRows {
					log.Printf("Failed to get home team standing: %v", err)
					continue
				}
				if err == sql.ErrNoRows {
					homeStanding = sqlc.Standing{
						TeamID:   match.HomeID,
						SeasonID: currentSeason.ID,
						Points:   sql.NullInt64{Int64: 0, Valid: true},
						Wins:     sql.NullInt64{Int64: 0, Valid: true},
						Draws:    sql.NullInt64{Int64: 0, Valid: true},
						Losses:   sql.NullInt64{Int64: 0, Valid: true},
						GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
					}
					err = repo.CreateStanding(reqCtx, sqlc.CreateStandingParams{
						TeamID:   match.HomeID,
						SeasonID: currentSeason.ID,
						Points:   sql.NullInt64{Int64: 0, Valid: true},
						Wins:     sql.NullInt64{Int64: 0, Valid: true},
						Draws:    sql.NullInt64{Int64: 0, Valid: true},
						Losses:   sql.NullInt64{Int64: 0, Valid: true},
						GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
					})
					if err != nil {
						log.Printf("Failed to create home team standing: %v", err)
						continue
					}
				}

				guestStanding, err := repo.GetStanding(reqCtx, match.GuestID, currentSeason.ID)
				if err != nil && err != sql.ErrNoRows {
					log.Printf("Failed to get guest team standing: %v", err)
					continue
				}
				if err == sql.ErrNoRows {
					guestStanding = sqlc.Standing{
						TeamID:   match.GuestID,
						SeasonID: currentSeason.ID,
						Points:   sql.NullInt64{Int64: 0, Valid: true},
						Wins:     sql.NullInt64{Int64: 0, Valid: true},
						Draws:    sql.NullInt64{Int64: 0, Valid: true},
						Losses:   sql.NullInt64{Int64: 0, Valid: true},
						GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
					}
					err = repo.CreateStanding(reqCtx, sqlc.CreateStandingParams{
						TeamID:   match.GuestID,
						SeasonID: currentSeason.ID,
						Points:   sql.NullInt64{Int64: 0, Valid: true},
						Wins:     sql.NullInt64{Int64: 0, Valid: true},
						Draws:    sql.NullInt64{Int64: 0, Valid: true},
						Losses:   sql.NullInt64{Int64: 0, Valid: true},
						GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
					})
					if err != nil {
						log.Printf("Failed to create guest team standing: %v", err)
						continue
					}
				}

				// Update standings based on match result
				if homeScore == guestScore {
					// Draw
					homeStanding.Points.Int64++
					homeStanding.Draws.Int64++
					guestStanding.Points.Int64++
					guestStanding.Draws.Int64++
				} else if homeScore > guestScore {
					// Home win
					homeStanding.Points.Int64 += 3
					homeStanding.Wins.Int64++
					guestStanding.Losses.Int64++
				} else {
					// Guest win
					guestStanding.Points.Int64 += 3
					guestStanding.Wins.Int64++
					homeStanding.Losses.Int64++
				}

				// Update goal differences
				homeStanding.GoalDiff.Int64 += homeScore - guestScore
				guestStanding.GoalDiff.Int64 += guestScore - homeScore

				// Save updated standings
				err = repo.UpdateStanding(reqCtx, sqlc.UpdateStandingParams{
					TeamID:   homeStanding.TeamID,
					SeasonID: currentSeason.ID,
					Points:   homeStanding.Points,
					Wins:     homeStanding.Wins,
					Draws:    homeStanding.Draws,
					Losses:   homeStanding.Losses,
					GoalDiff: homeStanding.GoalDiff,
				})
				if err != nil {
					log.Printf("Failed to update home team standing: %v", err)
					continue
				}

				err = repo.UpdateStanding(reqCtx, sqlc.UpdateStandingParams{
					TeamID:   guestStanding.TeamID,
					SeasonID: currentSeason.ID,
					Points:   guestStanding.Points,
					Wins:     guestStanding.Wins,
					Draws:    guestStanding.Draws,
					Losses:   guestStanding.Losses,
					GoalDiff: guestStanding.GoalDiff,
				})
				if err != nil {
					log.Printf("Failed to update guest team standing: %v", err)
					continue
				}
			}

			// Move to next week if not the last week
			if week < totalWeeks {
				err = repo.IncrementWeek(reqCtx, currentSeason.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment week"})
					return
				}
			}
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}

// generateRoundRobinFixtures generates a complete season of fixtures where each team
// plays against every other team twice (home and away)
func generateRoundRobinFixtures(repo repository.Repository, ctx context.Context) error {
	// Get current season
	currentSeason, err := repo.GetCurrentSeason(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch current season: %w", err)
	}

	teams, err := repo.ListTeams(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch teams: %w", err)
	}

	n := len(teams)
	if n < 2 {
		return fmt.Errorf("need at least 2 teams to generate fixtures")
	}

	// If odd number of teams, add a "bye" team
	if n%2 != 0 {
		teams = append(teams, sqlc.Team{ID: -1}) // Dummy team for odd number of teams
		n++
	}

	// Total number of rounds = 2(n-1) for double round-robin
	totalRounds := 2 * (n - 1)

	// First half of the season (each team plays every other team once)
	for round := 1; round <= n-1; round++ {
		// Generate matches for this round
		for i := 0; i < n/2; i++ {
			team1Idx := i
			team2Idx := n - 1 - i

			// Skip matches involving the dummy team
			if teams[team1Idx].ID != -1 && teams[team2Idx].ID != -1 {
				// Create the fixture
				err = repo.CreateFixture(ctx, sqlc.CreateFixtureParams{
					HomeID:   teams[team1Idx].ID,
					GuestID:  teams[team2Idx].ID,
					Played:   sql.NullBool{Bool: false, Valid: true},
					Week:     int64(round),
					SeasonID: currentSeason.ID,
				})
				if err != nil {
					return fmt.Errorf("failed to create fixture: %w", err)
				}
			}
		}

		// Rotate teams for next round (keep first team fixed, rotate others clockwise)
		lastTeam := teams[n-1]
		for i := n - 1; i > 1; i-- {
			teams[i] = teams[i-1]
		}
		teams[1] = lastTeam
	}

	// Second half of the season (reverse home/away for each match)
	for round := n; round <= totalRounds; round++ {
		firstRoundMatches, err := repo.GetMatchesByWeek(ctx, int64(round-n+1), currentSeason.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch first round matches: %w", err)
		}

		// Create reverse fixtures
		for _, match := range firstRoundMatches {
			err = repo.CreateFixture(ctx, sqlc.CreateFixtureParams{
				HomeID:   match.GuestID, // Swap home and away
				GuestID:  match.HomeID,
				Played:   sql.NullBool{Bool: false, Valid: true},
				Week:     int64(round),
				SeasonID: currentSeason.ID,
			})
			if err != nil {
				return fmt.Errorf("failed to create reverse fixture: %w", err)
			}
		}
	}

	return nil
}
