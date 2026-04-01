package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"goledger-challenge/internal/blockchain"
	"goledger-challenge/internal/repository"
)

// OracleUseCase defines the business logic orchestrating the Database and Blockchain
type OracleUseCase struct {
	repo  repository.OracleRepository
	chain blockchain.ChainClient
}

// NewOracleUseCase injects the Database and Chain ports into the business logic
func NewOracleUseCase(repo repository.OracleRepository, chain blockchain.ChainClient) *OracleUseCase {
	return &OracleUseCase{
		repo:  repo,
		chain: chain,
	}
}

// Set invokes the blockchain adapter to write a value
// Returns the transaction hash
func (uc *OracleUseCase) Set(ctx context.Context, value string) (string, error) {
	slog.Info("[UseCase] Starting Set()", slog.String("value", value))
	
	txHash, err := uc.chain.SetValue(ctx, value)
	if err != nil {
		return "", fmt.Errorf("UseCase Set failed: %w", err)
	}

	return txHash, nil
}

// Get reads the current on-chain state (via Besu/Network)
func (uc *OracleUseCase) Get(ctx context.Context) (string, error) {
	slog.Info("[UseCase] Starting Get() on Blockchain")
	
	value, err := uc.chain.GetValue(ctx)
	if err != nil {
		return "", fmt.Errorf("UseCase Get failed: %w", err)
	}

	return value, nil
}

// Sync pulls the truth from the blockchain and UPSERTs into postgresql
func (uc *OracleUseCase) Sync(ctx context.Context) (string, error) {
	slog.Info("[UseCase] Starting Sync(). Reading on-chain state...")
	
	// 1. Uses the UseCase itself to read safely
	chainValue, err := uc.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read value for Sync: %w", err)
	}

	slog.Info("[UseCase] Value read from Blockchain. Synchronizing with Postgres...", slog.String("chainValue", chainValue))

	// 2. Saves to the database
	if err := uc.repo.UpdateSyncedValue(ctx, chainValue); err != nil {
		return "", fmt.Errorf("failed to synchronize postgres cache: %w", err)
	}

	return chainValue, nil
}

// Check compares data from both sources (Local DB vs Blockchain)
func (uc *OracleUseCase) Check(ctx context.Context) (bool, string, string, error) {
	slog.Info("[UseCase] Starting divergence Check()")
	
	// 1. Reads from Blockchain
	chainVal, err := uc.chain.GetValue(ctx)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to read from chain during check: %w", err)
	}

	// 2. Reads from Database
	dbVal, err := uc.repo.GetSyncedValue(ctx)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to read from db during check: %w", err)
	}

	// If both match, the central value (inSync) will be true
	inSync := chainVal == dbVal
	
	if inSync {
		slog.Info("[UseCase] State is consistent!", slog.String("db", dbVal), slog.String("chain", chainVal))
	} else {
		slog.Warn("[UseCase] Divergence between Local and Global state detected!", slog.String("db", dbVal), slog.String("chain", chainVal))
	}

	return inSync, dbVal, chainVal, nil
}
