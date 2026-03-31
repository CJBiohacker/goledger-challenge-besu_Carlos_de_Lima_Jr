package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OracleRepository é a nossa "Porta" (Interface).
// A camada de UseCase (Regra de Negócio) só conversará com esta interface, e não com o pgx/v5 diretamente.
type OracleRepository interface {
	// UpdateSyncedValue adiciona ou atualiza o último valor lido da blockchain.
	UpdateSyncedValue(ctx context.Context, value string) error
	
	// GetSyncedValue lê o valor guardado no nosso banco de dados.
	GetSyncedValue(ctx context.Context) (string, error)
}

// postgresOracleRepo é o nosso "Adaptador" de Banco de Dados.
type postgresOracleRepo struct {
	db *pgxpool.Pool
}

// NewPostgresOracleRepo atua como o Factory Pattern que injeta nossa pool e cria a tabela provisória
func NewPostgresOracleRepo(ctx context.Context, db *pgxpool.Pool) (OracleRepository, error) {
	repo := &postgresOracleRepo{db: db}
	
	// Ao instanciar o repositório, garantimos que o Schema e Tabela existem
	if err := repo.initSchema(ctx); err != nil {
		return nil, err
	}
	
	return repo, nil
}

// initSchema Cria a tabela `state_cache` automaticamente caso a infra a suba vazia
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
		slog.Error("Erro ao preparar schema do banco de dados", slog.Any("error", err))
		return fmt.Errorf("falha ao criar tabela: %w", err)
	}
	return nil
}

func (r *postgresOracleRepo) UpdateSyncedValue(ctx context.Context, value string) error {
	// Como somos apenas um "cache" espelho da Blockchain, manteremos sempre no ID = 1.
	// O UPSERT no Postgres ocorre via "ON CONFLICT (id) DO UPDATE".
	query := `
		INSERT INTO state_cache (id, value, updated_at) 
		VALUES (1, $1, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE 
		SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP;
	`
	
	_, err := r.db.Exec(ctx, query, value)
	if err != nil {
		slog.Error("Erro ao gravar (UPSERT) na tabela state_cache", slog.Any("error", err))
		return fmt.Errorf("falha gravar no banco: %w", err)
	}
	
	slog.Info("Valor atualizado no PostgreSQL com sucesso", slog.String("novo_valor", value))
	return nil
}

func (r *postgresOracleRepo) GetSyncedValue(ctx context.Context) (string, error) {
	query := `SELECT value FROM state_cache WHERE id = 1;`
	
	var value string
	err := r.db.QueryRow(ctx, query).Scan(&value)
	
	if err != nil {
		// Se não houver nada salvo ainda, o aplicativo não deve capotar. O cache está apenas vazio.
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("Nenhum dado encontrado no cache estadual (state_cache)")
			return "", nil
		}
		
		slog.Error("Falha na varredura do state_cache", slog.Any("error", err))
		return "", fmt.Errorf("erro lendo do banco: %w", err)
	}
	
	return value, nil
}
