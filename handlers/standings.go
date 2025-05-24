package handlers

import (
	"database/sql"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/orhosko/go-backend/repository"
	"github.com/orhosko/go-backend/sqlc"
	"github.com/orhosko/go-backend/templates"
)

// RegisterStandingsRoutes registers all standings related routes
func RegisterStandingsRoutes(router *gin.Engine, repo repository.Repository) {
	router.GET("/standings", handleGetStandings(repo))
	router.POST("/standings/recalculate", handleRecalculateStandings(repo))
	router.POST("/standings/team/:teamId", handleUpdateTeamStanding(repo))
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

		component := templates.Index(templates.StandingsPageData{
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

func handleRecalculateStandings(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Get current season if not specified
		seasonID := c.Query("seasonId")
		var currentSeason sqlc.Season
		var err error

		if seasonID == "" {
			currentSeason, err = repo.GetCurrentSeason(reqCtx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current season"})
				return
			}
		} else {
			seasonIDInt, err := strconv.ParseInt(seasonID, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid season ID"})
				return
			}
			currentSeason.ID = seasonIDInt
		}

		// Get all teams in the season
		teams, err := repo.ListTeams(reqCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}

		// Recalculate standings for each team
		for _, team := range teams {
			err = recalculateTeamStanding(reqCtx, repo, currentSeason.ID, team.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update standings for team " + team.Name})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Standings recalculated successfully"})
	}
}

func handleUpdateTeamStanding(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Parse team ID from URL
		teamID, err := strconv.ParseInt(c.Param("teamId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
			return
		}

		// Get current season if not specified
		seasonID := c.Query("seasonId")
		var currentSeason sqlc.Season

		if seasonID == "" {
			currentSeason, err = repo.GetCurrentSeason(reqCtx)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current season"})
				return
			}
		} else {
			seasonIDInt, err := strconv.ParseInt(seasonID, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid season ID"})
				return
			}
			currentSeason.ID = seasonIDInt
		}

		// Recalculate standings for the specified team
		err = recalculateTeamStanding(reqCtx, repo, currentSeason.ID, teamID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update team standing"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Team standing updated successfully"})
	}
}
