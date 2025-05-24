package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/orhosko/go-backend/repository"
	"github.com/orhosko/go-backend/sqlc"
	"github.com/orhosko/go-backend/templates"
)

// RegisterTeamRoutes registers all team related routes
func RegisterTeamRoutes(router *gin.Engine, repo repository.Repository) {
	router.GET("/teams", handleTeams(repo))
}

func handleTeams(repo repository.Repository) gin.HandlerFunc {
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

		// Get all teams
		teams, err := repo.ListTeams(reqCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}

		// Prepare team details
		var teamsData []templates.TeamDetailData
		for _, team := range teams {
			// Get team's standing
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

			// Calculate total matches
			totalMatches := standing.Wins.Int64 + standing.Draws.Int64 + standing.Losses.Int64

			// Calculate percentages
			var winPercentage, drawPercentage, lossPercentage float64
			if totalMatches > 0 {
				winPercentage = float64(standing.Wins.Int64) / float64(totalMatches) * 100
				drawPercentage = float64(standing.Draws.Int64) / float64(totalMatches) * 100
				lossPercentage = float64(standing.Losses.Int64) / float64(totalMatches) * 100
			}

			// Calculate goals
			var goalsScored, goalsConceded int64
			if standing.GoalDiff.Valid {
				goalsConceded = 0                     // Initialize with a default value
				goalsScored = standing.GoalDiff.Int64 // This is just an approximation
				if goalsScored < 0 {
					goalsConceded = -goalsScored
					goalsScored = 0
				}
			}

			teamData := templates.TeamDetailData{
				Team:     team,
				Standing: standing,
				Stats: templates.TeamStats{
					TotalMatches:   int(totalMatches),
					GoalsScored:    goalsScored,
					GoalsConceded:  goalsConceded,
					WinPercentage:  winPercentage,
					DrawPercentage: drawPercentage,
					LossPercentage: lossPercentage,
				},
			}
			teamsData = append(teamsData, teamData)
		}

		// Render the teams page
		teamsPage := templates.Teams(templates.TeamsPageData{
			Teams:         teamsData,
			CurrentSeason: currentSeason,
		})

		c.Status(http.StatusOK)
		teamsPage.Render(reqCtx, c.Writer)
	}
}
