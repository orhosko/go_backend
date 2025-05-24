package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/orhosko/go-backend/repository"
	"github.com/orhosko/go-backend/sqlc"
	"github.com/orhosko/go-backend/templates"
)

// RegisterMatchRoutes registers all match related routes
func RegisterMatchRoutes(router *gin.Engine, repo repository.Repository) {
	router.GET("/matches", handleMatches(repo))
	router.POST("/matches/:id/edit", handleEditMatch(repo))
}

func handleEditMatch(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse match ID from URL
		matchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid match ID")
			return
		}

		// Parse form data
		homeScore, err := strconv.ParseInt(c.PostForm("home_score"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid home score")
			return
		}

		guestScore, err := strconv.ParseInt(c.PostForm("guest_score"), 10, 64)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid guest score")
			return
		}

		reqCtx := c.Request.Context()

		// Get current season
		currentSeason, err := repo.GetCurrentSeason(reqCtx)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to get current season")
			return
		}

		// Get current week
		currentWeek, err := repo.GetCurrentWeek(reqCtx, currentSeason.ID)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to get current week")
			return
		}

		// Search for the match in all weeks up to current week
		var match sqlc.GetMatchesByWeekRow
		matchFound := false

		for week := 1; week <= currentWeek && !matchFound; week++ {
			matches, err := repo.GetMatchesByWeek(reqCtx, int64(week), currentSeason.ID)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to fetch matches")
				return
			}

			for _, m := range matches {
				if m.ID == matchID {
					match = m
					matchFound = true
					break
				}
			}
		}

		if !matchFound {
			c.String(http.StatusNotFound, "Match not found")
			return
		}

		// Determine the winner
		var winnerID sql.NullInt64
		if homeScore > guestScore {
			winnerID = sql.NullInt64{Int64: match.HomeID, Valid: true}
		} else if guestScore > homeScore {
			winnerID = sql.NullInt64{Int64: match.GuestID, Valid: true}
		}

		// Update the match result
		err = repo.SaveResult(reqCtx, sqlc.SaveResultParams{
			MatchID:    matchID,
			HomeScore:  homeScore,
			GuestScore: guestScore,
			WinnerID:   winnerID,
		})
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to update match result")
			return
		}

		// Mark match as played if not already
		if !match.Played.Bool {
			err = repo.MarkMatchAsPlayed(reqCtx, matchID)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to mark match as played")
				return
			}
		}

		// Update standings for both teams
		err = recalculateTeamStanding(reqCtx, repo, currentSeason.ID, match.HomeID)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to update home team standing")
			return
		}

		err = recalculateTeamStanding(reqCtx, repo, currentSeason.ID, match.GuestID)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to update guest team standing")
			return
		}

		// Redirect back to matches page
		c.Redirect(http.StatusSeeOther, "/matches")
	}
}

func handleMatches(repo repository.Repository) gin.HandlerFunc {
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

		// Get current week
		currentWeek, err := repo.GetCurrentWeek(reqCtx, currentSeason.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current week"})
			return
		}

		// Initialize matches map
		matchesByWeek := make(map[int][]templates.MatchData)

		// Fetch matches for each week up to current week
		for week := 1; week <= currentWeek; week++ {
			matches, err := repo.GetMatchesByWeek(reqCtx, int64(week), currentSeason.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
				return
			}

			var weekMatches []templates.MatchData
			for _, match := range matches {
				// Get match result if the match has been played
				var result *sqlc.GetMatchResultRow
				if match.Played.Valid && match.Played.Bool {
					matchResult, err := repo.GetMatchResult(reqCtx, match.ID)
					if err != nil && err != sql.ErrNoRows {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch match result"})
						return
					}
					if err != sql.ErrNoRows {
						result = &matchResult
					}
				}

				matchData := templates.MatchData{
					Match: sqlc.Match{
						ID:       match.ID,
						SeasonID: match.SeasonID,
						HomeID:   match.HomeID,
						GuestID:  match.GuestID,
						Played:   match.Played,
						Week:     match.Week,
					},
					HomeTeamName:  match.HomeTeamName,
					GuestTeamName: match.GuestTeamName,
					Result:        result,
				}
				weekMatches = append(weekMatches, matchData)
			}

			if len(weekMatches) > 0 {
				matchesByWeek[week] = weekMatches
			}
		}

		// Render the matches page
		matchesPage := templates.Matches(templates.MatchesPageData{
			Matches:       matchesByWeek,
			CurrentSeason: currentSeason,
			CurrentWeek:   currentWeek,
		})

		c.Status(http.StatusOK)
		matchesPage.Render(reqCtx, c.Writer)
	}
}
