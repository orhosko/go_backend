package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"

	"github.com/orhosko/go-backend/sqlc"
)

// DB represents the database connection and sqlc queries.
type DB struct {
	Queries *sqlc.Queries
	Conn    *sql.DB
}

// NewDB initializes a new database connection and sqlc queries.
func NewDB(databaseURL string) (*DB, error) {
	// Change "sqlite3" to "pgx" if using PostgreSQL
	db, err := sql.Open("sqlite", databaseURL)

	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Ping the database to ensure connection is established
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established successfully.")

	// Create a new Queries instance from sqlc generated code
	queries := sqlc.New(db)

	return &DB{
		Queries: queries,
		Conn:    db,
	}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d.Conn != nil {
		log.Println("Closing database connection.")
		return d.Conn.Close()
	}
	return nil
}

// EnsureSchema initializes the database schema (useful for SQLite in development).
// In a production environment, you would use a dedicated migration tool.
func (d *DB) EnsureSchema(schemaPath string) error {
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = d.Conn.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}
	log.Println("Database schema ensured.")
	return nil
}
