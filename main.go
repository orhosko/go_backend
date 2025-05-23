package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	// "os"
	// "os/signal"
	// "syscall"
	// "time"
	// "reflect"

	_ "modernc.org/sqlite"

	"github.com/orhosko/go-backend/templates"
	"github.com/orhosko/go-backend/tutorial"

	"github.com/orhosko/go-backend/config"
	"github.com/orhosko/go-backend/db"
	"github.com/orhosko/go-backend/repository"
	"github.com/orhosko/go-backend/sqlc"

	// "your_project_name/internal/handlers"   // Adjust import path
	// "your_project_name/internal/service"    // Adjust import path

	"github.com/gin-gonic/gin"
	"net/http"
)

//go:embed sqlc/schema.sql
var ddl string

var ctx context.Context
var queries *tutorial.Queries

func getTeams() ([]tutorial.Team, error) {
	teams, err := queries.ListTeams(ctx)
	if err != nil {
		return nil, err
	}

	return teams, nil
}

func run() error {
	ctx = context.Background()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}

	// create tables
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return err
	}

	queries = tutorial.New(db)

	// list all authors
	teams, err := queries.ListTeams(ctx)
	if err != nil {
		return err
	}
	log.Println(teams)

	// create an author
	// insertedAuthor, err := queries.CreateAuthor(ctx, tutorial.CreateAuthorParams{
	// 	Name: "Brian Kernighan",
	// 	Bio:  sql.NullString{String: "Co-author of The C Programming Language and The Go Programming Language", Valid: true},
	// })
	// if err != nil {
	// 	return err
	// }
	// log.Println(insertedAuthor)

	// get the author we just inserted
	// fetchedAuthor, err := queries.GetAuthor(ctx, insertedAuthor.ID)
	// if err != nil {
	// 	return err
	// }
	//
	// // prints true
	// log.Println(reflect.DeepEqual(insertedAuthor, fetchedAuthor))
	return nil
}

// interface Actions {
//   run(),
// }

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	dbConn, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}()

	println("Database initialized")

	// Ensure the schema is created for SQLite in development
	// In production, use migrations
	if err := dbConn.EnsureSchema("sqlc/schema.sql"); err != nil {
		log.Fatalf("Error ensuring database schema: %v", err)
	}

	// Initialize repositories
	repo := repository.NewSQLCRepository(dbConn.Queries)

	// Initialize services (business logic)
	// leagueService := service.NewLeagueService(repo) // Assuming you'll create a LeagueService

	// Initialize Gin router
	router := gin.Default()

	// Register handlers
	// handlers.RegisterLeagueHandlers(router, leagueService) // You'll create this function

	/*
		// Start HTTP server
		srv := &http.Server{
			Addr:    ":" + cfg.ServerPort,
			Handler: router,
		}

		// Graceful shutdown
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Listen and serve error: %s\n", err)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}

		log.Println("Server exiting")
	*/

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong pong",
		})
	})

	cont := context.Background()

	router.GET("/teams", func(c *gin.Context) {
		teams, err := repository.TeamRepository.ListTeams(repo, cont)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		helloComponent := templates.Hello("World", teams)
		c.Status(http.StatusOK) // Set the HTTP status code
		helloComponent.Render(c.Request.Context(), c.Writer)
	})

	router.GET("/", func(c *gin.Context) {
		helloComponent := templates.Hello("World", nil)

		c.Status(http.StatusOK) // Set the HTTP status code
		helloComponent.Render(c.Request.Context(), c.Writer)
	})

	router.GET("/standings", func(c *gin.Context) {

		standingData := templates.StandingPageData{
			CurrentWeek: 1,
			LeagueTable: []sqlc.TeamStanding{
				{
					TeamName: "Team A",
					Played:   10,
					Won:      8,
					Drawn:    1,
					Lost:     1,
					Points:   25,
				},
				{
					TeamName: "Team B",
					Played:   10,
					Won:      7,
					Drawn:    2,
					Lost:     1,
					Points:   23,
				},
			},
			MatchResults: []templates.MatchDisplay{
				{
					HomeTeamName:  "Team A",
					GuestTeamName: "Team B",
					HomeScore:     2,
					GuestScore:    1,
				},
			},
			ChanpionshipPredictions: []templates.TeamPrediction{
				{
					TeamName:   "Team A",
					Probabilty: 1,
				},
			},
		}

		helloComponent := templates.Standings(standingData)

		c.Status(http.StatusOK) // Set the HTTP status code
		helloComponent.Render(c.Request.Context(), c.Writer)

	})

	router.Run() // listen and serve on 0.0.0.0:8080
}
