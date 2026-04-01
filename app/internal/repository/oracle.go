package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OracleRepository is our "Port" (Interface).
// The UseCase layer (Business Logic) will only talk to this interface, not directly to pgx/v5.
type OracleRepository interface {
	// UpdateSyncedValue adds or updates the last value read from the blockchain.
	UpdateSyncedValue(ctx context.Context, value string) error
	
	// GetSyncedValue reads the value stored in our database.
	GetSyncedValue(ctx context.Context) (string, error)
}

// postgresOracleRepo is our Database "Adapter".
type postgresOracleRepo struct {
	db *pgxpool.Pool
}

// NewPostgresOracleRepo acts as the Factory Pattern that injects our pool and creates the provisional table
func NewPostgresOracleRepo(ctx context.Context, db *pgxpool.Pool) (OracleRepository, error) {
	repo := &postgresOracleRepo{db: db}
	
	// When instantiating the repository, we ensure the Schema and Table exist
	if err := repo.initSchema(ctx); err != nil {
		return nil, err
	}
	
	return repo, nil
}

// initSchema Creates the `state_cache` table automatically in case the infra spins it up empty
func (r *postgresOracleRepo) initSchema(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS state_cache (
			id INT PRIMARY KEY,
			value TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		slog.Error("Error preparing database schema", slog.Any("error", err))
		return fmt.Errorf("failed creating table: %w", err)
	}
	return nil
}

func (r *postgresOracleRepo) UpdateSyncedValue(ctx context.Context, value string) error {
	// Since we are just a mirror "cache" of the Blockchain, we will always keep ID = 1.
	// The UPSERT in Postgres occurs via "ON CONFLICT (id) DO UPDATE".
	query := `
		INSERT INTO state_cache (id, value, updated_at) 
		VALUES (1, $1, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE 
		SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP;
	`
	
	_, err := r.db.Exec(ctx, query, value)
	if err != nil {
		slog.Error("Error writing (UPSERT) to state_cache table", slog.Any("error", err))
		return fmt.Errorf("failed saving to database: %w", err)
	}
	
	slog.Info("Value updated successfully in PostgreSQL", slog.String("new_value", value))
	return nil
}

func (r *postgresOracleRepo) GetSyncedValue(ctx context.Context) (string, error) {
	query := `SELECT value FROM state_cache WHERE id = 1;`
	
	var value string
	err := r.db.QueryRow(ctx, query).Scan(&value)
	
	if err != nil {
		// If nothing is saved yet, the application shouldn't crash. Cache is simply empty.
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("No data found in state cache (state_cache)")
			return "", nil
		}
		
		slog.Error("Failed scanning state_cache", slog.Any("error", err))
		return "", fmt.Errorf("error reading from database: %w", err)
	}
	
	return value, nil
}
