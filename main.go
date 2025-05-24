package main

import (
	_ "embed"
	"log"

	_ "modernc.org/sqlite"

	"github.com/orhosko/go-backend/config"
	database "github.com/orhosko/go-backend/db"
	"github.com/orhosko/go-backend/handlers"
	"github.com/orhosko/go-backend/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	dbConn, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	println("Database initialized")

	// Ensure the schema is created for SQLite in development
	// In production, use migrations
	if err := dbConn.EnsureSchema("sqlc/schema.sql"); err != nil {
		log.Fatalf("Error ensuring database schema: %v", err)
	}

	// Initialize repositories
	repo := repository.NewSQLCRepository(dbConn.Queries, dbConn.Conn)

	// Initialize Gin router
	router := gin.Default()

	// Register all routes
	handlers.RegisterHomeRoutes(router, repo)
	handlers.RegisterPingRoutes(router)
	handlers.RegisterTeamRoutes(router, repo)
	handlers.RegisterFixtureRoutes(router, repo)
	handlers.RegisterSeasonRoutes(router, repo)
	handlers.RegisterMatchRoutes(router, repo)
	handlers.RegisterStandingsRoutes(router, repo)

	// Start the server without closing the database connection
	if err := router.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
