package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool initializes the PostgreSQL connection pool
func NewPostgresPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	slog.Info("Initializing database connection pool...")

	// Parses the connection string with pgx pattern configurations
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("error parsing db URL: %w", err)
	}

	// Creates the actual managed pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	// Performs a safety ping to ensure real connectivity
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	slog.Info("Connection with PostgreSQL established successfully")
	return pool, nil
}
