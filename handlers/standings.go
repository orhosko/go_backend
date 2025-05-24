package handlers

import (
	"database/sql"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/orhosko/go-backend/repository"
	"github.com/orhosko/go-backend/sqlc"
	"github.com/orhosko/go-backend/templates"
)

func RegisterStandingsRoutes(router *gin.Engine, repo repository.Repository) {
	router.GET("/", handleGetStandings(repo))
}

func handleGetStandings(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Get current season
		currentSeason, err := repo.GetCurrentSeason(reqCtx)
		if err != nil {
			if err == sql.ErrNoRows {
				currentSeason, err = repo.CreateNewSeason(reqCtx, 2025)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create initial season"})
					return
				}
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

		// Get all teams
		teams, err := repo.ListTeams(reqCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}

		// Get standings for each team
		var standings []templates.TeamStanding
		for _, team := range teams {
			standing, err := repo.GetStanding(reqCtx, team.ID, currentSeason.ID)
			if err != nil && err != sql.ErrNoRows {
				continue
			}
			if err == sql.ErrNoRows {
				standing = sqlc.Standing{
					TeamID:   team.ID,
					SeasonID: currentSeason.ID,
					Points:   sql.NullInt64{Int64: 0, Valid: true},
					Wins:     sql.NullInt64{Int64: 0, Valid: true},
					Draws:    sql.NullInt64{Int64: 0, Valid: true},
					Losses:   sql.NullInt64{Int64: 0, Valid: true},
					GoalDiff: sql.NullInt64{Int64: 0, Valid: true},
				}
			}
			standings = append(standings, templates.TeamStanding{
				Team:     team,
				Standing: standing,
			})
		}

		// Sort standings by points and goal difference
		sortStandings(standings)

		// Get match results for current week
		var matchResults []templates.MatchDisplay
		matches, err := repo.GetMatchesByWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
			return
		}
		if err != sql.ErrNoRows {
			for _, match := range matches {
				if !match.Played.Bool {
					continue
				}
				result, err := repo.GetMatchResult(reqCtx, match.ID)
				if err != nil {
					continue
				}
				matchResults = append(matchResults, templates.MatchDisplay{
					HomeTeamName:  match.HomeTeamName,
					GuestTeamName: match.GuestTeamName,
					HomeScore:     result.HomeScore,
					GuestScore:    result.GuestScore,
				})
			}
		}

		// Get upcoming fixtures
		var fixtures []templates.MatchFixture
		upcomingMatches, err := repo.GetUnplayedMatchesByWeek(reqCtx, int64(currentWeek), currentSeason.ID)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch fixtures"})
			return
		}
		if err != sql.ErrNoRows {
			for _, match := range upcomingMatches {
				fixtures = append(fixtures, templates.MatchFixture{
					HomeTeamName:  match.HomeTeamName,
					GuestTeamName: match.GuestTeamName,
				})
			}
		}

		// Calculate championship predictions
		predictions, err := calculateChampionshipPredictions(reqCtx, repo, currentSeason)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate predictions"})
			return
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
		isSeasonComplete := currentWeek >= totalWeeks && len(fixtures) == 0

		component := templates.Standings(templates.StandingsPageData{
			CurrentWeek:             currentWeek,
			CurrentYear:             int(currentSeason.Year),
			LeagueTable:             standings,
			MatchResults:            matchResults,
			ChampionshipPredictions: templatePredictions,
			Fixtures:                fixtures,
			IsSeasonComplete:        isSeasonComplete,
		})

		component.Render(reqCtx, c.Writer)
	}
}

// sortStandings sorts the standings by points (descending) and goal difference (descending)
func sortStandings(standings []templates.TeamStanding) {
	sort.Slice(standings, func(i, j int) bool {
		if standings[i].Standing.Points.Int64 != standings[j].Standing.Points.Int64 {
			return standings[i].Standing.Points.Int64 > standings[j].Standing.Points.Int64
		}
		return standings[i].Standing.GoalDiff.Int64 > standings[j].Standing.GoalDiff.Int64
	})
}
