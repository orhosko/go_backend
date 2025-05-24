package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/orhosko/go-backend/repository"
)

// RegisterSeasonRoutes registers all season related routes
func RegisterSeasonRoutes(router *gin.Engine, repo repository.Repository) {
	router.POST("/reset-to-2025", handleResetToYear(repo))
	router.POST("/start-new-season", handleStartNewSeason(repo))
}

func handleResetToYear(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		err := repo.ResetToYear(reqCtx, 2025)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset to 2025"})
			return
		}

		// Generate fixtures for the reset season
		err = generateRoundRobinFixtures(repo, reqCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate fixtures"})
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}

func handleStartNewSeason(repo repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Get current season
		currentSeason, err := repo.GetCurrentSeason(reqCtx)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current season"})
			return
		}

		var newYear int64
		if err == sql.ErrNoRows {
			newYear = 2025 // Start with 2025 if no season exists
		} else {
			newYear = currentSeason.Year + 1
		}

		// Create new season
		newSeason, err := repo.CreateNewSeason(reqCtx, newYear)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new season"})
			return
		}

		// Set it as the current season
		err = repo.SetCurrentSeason(reqCtx, newSeason.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set current season"})
			return
		}

		// Initialize game state for the new season
		err = repo.InitializeGameState(reqCtx, newSeason.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize game state"})
			return
		}

		// Generate fixtures for the new season
		err = generateRoundRobinFixtures(repo, reqCtx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate fixtures"})
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	}
}
