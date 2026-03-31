package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"goledger-challenge/internal/blockchain"
	"goledger-challenge/internal/repository"
)

// OracleUseCase define a regra de negócio orquestrando o Banco e a Blockchain
type OracleUseCase struct {
	repo  repository.OracleRepository
	chain blockchain.ChainClient
}

// NewOracleUseCase injeta as portas de Banco e Chain na regra de negócio
func NewOracleUseCase(repo repository.OracleRepository, chain blockchain.ChainClient) *OracleUseCase {
	return &OracleUseCase{
		repo:  repo,
		chain: chain,
	}
}

// Set invoca o adaptador de blockchain para escrever um valor
// Retorna a transaction hash
func (uc *OracleUseCase) Set(ctx context.Context, value string) (string, error) {
	slog.Info("[UseCase] Iniciando Set()", slog.String("value", value))
	
	txHash, err := uc.chain.SetValue(ctx, value)
	if err != nil {
		return "", fmt.Errorf("UseCase Set falhou: %w", err)
	}

	return txHash, nil
}

// Get lê o estado atual on-chain (via Besu/Network)
func (uc *OracleUseCase) Get(ctx context.Context) (string, error) {
	slog.Info("[UseCase] Iniciando Get() na Blockchain")
	
	value, err := uc.chain.GetValue(ctx)
	if err != nil {
		return "", fmt.Errorf("UseCase Get falhou: %w", err)
	}

	return value, nil
}

// Sync puxa a verdade da blockchain e faz UPSERT no postgresql
func (uc *OracleUseCase) Sync(ctx context.Context) (string, error) {
	slog.Info("[UseCase] Iniciando Sync(). Lendo state on-chain...")
	
	// 1. Usa o próprio UseCase pra ler de forma segura
	chainValue, err := uc.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("falha ao ler valor para Sync: %w", err)
	}

	slog.Info("[UseCase] Valor lido da Blockchain. Sincronizando com Postgres...", slog.String("chainValue", chainValue))

	// 2. Grava no banco
	if err := uc.repo.UpdateSyncedValue(ctx, chainValue); err != nil {
		return "", fmt.Errorf("falha ao sincronizar o cache do postgres: %w", err)
	}

	return chainValue, nil
}

// Check cruza os dados das duas fontes (Banco Local vs Blockchain)
func (uc *OracleUseCase) Check(ctx context.Context) (bool, string, string, error) {
	slog.Info("[UseCase] Iniciando Check() de divergências")
	
	// 1. Lê a Blockchain
	chainVal, err := uc.chain.GetValue(ctx)
	if err != nil {
		return false, "", "", fmt.Errorf("falha ao ler da chain no check: %w", err)
	}

	// 2. Lê do Banco de Dados
	dbVal, err := uc.repo.GetSyncedValue(ctx)
	if err != nil {
		return false, "", "", fmt.Errorf("falha ao ler do db no check: %w", err)
	}

	// Se ambos baterem, o valor central (inSync) será true
	inSync := chainVal == dbVal
	
	if inSync {
		slog.Info("[UseCase] Estado íntegro!", slog.String("db", dbVal), slog.String("chain", chainVal))
	} else {
		slog.Warn("[UseCase] Divergência de Estado Local vs Global detectada!", slog.String("db", dbVal), slog.String("chain", chainVal))
	}

	return inSync, dbVal, chainVal, nil
}
