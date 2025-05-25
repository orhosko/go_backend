package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/orhosko/go-backend/repository"
	"github.com/orhosko/go-backend/sqlc"
	"github.com/orhosko/go-backend/templates"
)

// RegisterHomeRoutes registers all home related routes
func RegisterHomeRoutes(router *gin.Engine, repo repository.Repository) {
	router.GET("/", handleHome(repo))
}

func handleHome(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Get current season
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

		// Get current week from the database, default to 1 if not set
		currentWeek, err := repo.GetCurrentWeek(reqCtx, currentSeason.ID)
		if err != nil {
			currentWeek = 1 // Default to week 1 if not set
		}

		// Check if fixtures exist for the current week
		matches, err := repo.GetMatchesByWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err == sql.ErrNoRows || len(matches) == 0 {
			// No fixtures exist, generate them
			err = generateRoundRobinFixtures(repo, reqCtx)
			if err != nil {
				log.Printf("Failed to generate fixtures: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate fixtures"})
				return
			}
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check fixtures"})
			return
		}

		// Get team standings
		teams, err := repo.ListTeams(reqCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}

		// Get standings for each team
		var leagueTable []templates.TeamStanding
		for _, team := range teams {
			standing, err := repo.GetStanding(reqCtx, team.ID, currentSeason.ID)
			if err != nil {
				if err == sql.ErrNoRows {
					// If no standing exists, create one with zeros
					err = repo.CreateStanding(reqCtx, sqlc.CreateStandingParams{
						TeamID:   team.ID,
						SeasonID: currentSeason.ID,
						Points:   sql.NullInt64{Int64: 0, Valid: true},
						Wins:     sql.NullInt64{Int64: 0, Valid: true},
						Draws:    sql.NullInt64{Int64: 0, Valid: true},
						Losses:   sql.NullInt64{Int64: 0, Valid: true},
						GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
					})
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create standing"})
						return
					}
					standing = sqlc.Standing{
						TeamID:   team.ID,
						SeasonID: currentSeason.ID,
						Points:   sql.NullInt64{Int64: 0, Valid: true},
						Wins:     sql.NullInt64{Int64: 0, Valid: true},
						Draws:    sql.NullInt64{Int64: 0, Valid: true},
						Losses:   sql.NullInt64{Int64: 0, Valid: true},
						GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
					}
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch standings"})
					return
				}
			}

			leagueTable = append(leagueTable, templates.TeamStanding{
				Team:     team,
				Standing: standing,
			})
		}

		// Sort league table by points (descending), then goal difference
		sort.Slice(leagueTable, func(i, j int) bool {
			if leagueTable[i].Standing.Points.Int64 == leagueTable[j].Standing.Points.Int64 {
				return leagueTable[i].Standing.GoalDiff.Int64 > leagueTable[j].Standing.GoalDiff.Int64
			}
			return leagueTable[i].Standing.Points.Int64 > leagueTable[j].Standing.Points.Int64
		})

		// Get match results for current week (if any exist)
		var matchResults []templates.MatchDisplay
		matches, err = repo.GetMatchesByWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Failed to fetch matches: %v", err)
		} else if err != sql.ErrNoRows {
			for _, match := range matches {
				if match.Played.Bool {
					result, err := repo.GetMatchResult(reqCtx, match.ID)
					if err != nil {
						continue
					}

					matchResults = append(matchResults, templates.MatchDisplay{
						HomeTeamName:  result.HomeTeamName,
						GuestTeamName: result.GuestTeamName,
						HomeScore:     result.HomeScore,
						GuestScore:    result.GuestScore,
					})
				}
			}
		}

		// Get upcoming fixtures (if any exist)
		var fixtures []templates.MatchFixture
		unplayedMatches, err := repo.GetUnplayedMatchesByWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Failed to fetch fixtures: %v", err)
		} else if err != sql.ErrNoRows {
			for _, match := range unplayedMatches {
				fixtures = append(fixtures, templates.MatchFixture{
					HomeTeamName:  match.HomeTeamName,
					GuestTeamName: match.GuestTeamName,
				})
			}
		}

		// Calculate championship predictions
		predictions, err := calculateChampionshipPredictions(reqCtx, repo, currentSeason)
		if err != nil {
			log.Printf("Failed to calculate predictions: %v", err)
			predictions = []TeamPrediction{} // Use empty predictions if calculation fails
		}

		// Convert predictions to template format
		var templatePredictions []templates.TeamPrediction
		for _, pred := range predictions {
			templatePredictions = append(templatePredictions, templates.TeamPrediction{
				TeamName:    pred.TeamName,
				Probability: pred.Probability,
			})
		}

		// Check if season is complete
		totalWeeks := 2 * (len(teams) - 1)

		// Get all unplayed matches for the season
		unplayedMatchesForSeason, err := repo.GetUnplayedMatchesByWeek(reqCtx, int64(totalWeeks), currentSeason.ID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Failed to check for unplayed matches: %v", err)
		}

		isSeasonComplete := currentWeek >= totalWeeks && (err == sql.ErrNoRows || len(unplayedMatchesForSeason) == 0)

		if isSeasonComplete {
			err = repo.CompleteSeason(reqCtx, currentSeason.ID)
			if err != nil {
				log.Printf("Failed to mark season as complete: %v", err)
			}
		}

		standingData := templates.StandingsPageData{
			CurrentWeek:             currentWeek,
			CurrentYear:             int(currentSeason.Year),
			LeagueTable:             leagueTable,
			MatchResults:            matchResults,
			ChampionshipPredictions: templatePredictions,
			Fixtures:                fixtures,
			IsSeasonComplete:        isSeasonComplete,
		}

		component := templates.Index(standingData)
		c.Status(http.StatusOK)
		component.Render(c.Request.Context(), c.Writer)
	}
}
