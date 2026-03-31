package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool inicializa a piscina de conexões com o PostgreSQL
func NewPostgresPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	slog.Info("Inicializando pool de conexão com o banco de dados...")

	// Faz o parse da string de conexão com configurações pattern do pgx
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear URL do banco: %w", err)
	}

	// Cria de fato a pool gerenciada
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar o pool de conexao: %w", err)
	}

	// Executa um Ping de segurança para garantir a conectividade real
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping no banco de dados falhou: %w", err)
	}

	slog.Info("Conexão com PostgreSQL estabelecida com sucesso")
	return pool, nil
}
