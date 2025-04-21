package database

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPgxPool creates a new PostgreSQL connection pool.
func NewPgxPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL cannot be empty")
	}

	// pgxpool.ParseConfig automatically reads environment variables like PGHOST, PGUSER, etc.
	// but we can also force the use of the URL.
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config from URL: %w", err)
	}

	// Example: Set connection timeout if needed
	// config.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	err = pool.Ping(ctx)
	if err != nil {
		pool.Close() // Close the pool if ping fails
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database.")
	return pool, nil
}

// ClosePgxPool closes the PostgreSQL connection pool.
func ClosePgxPool(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
		log.Println("PostgreSQL connection pool closed.")
	}
}
