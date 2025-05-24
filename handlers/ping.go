package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterPingRoutes registers all ping related routes
func RegisterPingRoutes(router *gin.Engine) {
	router.GET("/ping", HandlePing)
}

// HandlePing handles the ping endpoint
func HandlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong pong",
	})
}
