package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ChainClient is the Hexagonal Architecture "Port" to the Blockchain.
type ChainClient interface {
	SetValue(ctx context.Context, value string) (string, error)
	GetValue(ctx context.Context) (string, error)
}

// besuClient is the actual "Adapter" using go-ethereum
type besuClient struct {
	client          *ethclient.Client
	contractAddress common.Address
	parsedABI       abi.ABI
	privateKey      string
	chainID         *big.Int
}

// Config centralizes constructor dependencies
type Config struct {
	RPCUrl          string
	ContractAddress string
	PrivateKey      string
	ABIPath         string // Path to SimpleStorage.json
}

// foundryOutput helps unpack massive JSON in an unmarshal operation
type foundryOutput struct {
	ABI json.RawMessage `json:"abi"`
}

// NewBesuClient is the factory pattern
func NewBesuClient(ctx context.Context, cfg Config) (ChainClient, error) {
	// 1. Connect to Besu RPC
	client, err := ethclient.DialContext(ctx, cfg.RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("error connecting to Besu RPC: %w", err)
	}

	// 2. Fetch ChainID of the connected network (usually local=1337)
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading chain id: %w", err)
	}

	// 3. Read and Parse the compiled Foundry ABI from disk
	rawFile, err := os.ReadFile(cfg.ABIPath)
	if err != nil {
		return nil, fmt.Errorf("error reading json ABI file at path %s: %w", cfg.ABIPath, err)
	}

	var output foundryOutput
	if err := json.Unmarshal(rawFile, &output); err != nil {
		return nil, fmt.Errorf("failed extracting 'abi' from json: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(output.ABI)))
	if err != nil {
		return nil, fmt.Errorf("error transforming ABI interface to eth package: %w", err)
	}

	slog.Info("Besu Blockchain Client adapted in memory", slog.String("chain_id", chainID.String()))

	return &besuClient{
		client:          client,
		contractAddress: common.HexToAddress(cfg.ContractAddress),
		parsedABI:       parsedABI,
		privateKey:      cfg.PrivateKey,
		chainID:         chainID,
	}, nil
}

// SetValue writes "set(uint256)" and awaits mining, returning the hash.
func (b *besuClient) SetValue(ctx context.Context, value string) (string, error) {
	// The function on-chain expects a large number (uint256)
	valInt, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return "", fmt.Errorf("input value is not a valid integer string=%s", value)
	}

	// Signature using private key
	priv, err := crypto.HexToECDSA(b.privateKey)
	if err != nil {
		return "", fmt.Errorf("error reading ECDSA private key: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(priv, b.chainID)
	if err != nil {
		return "", fmt.Errorf("error creating transactor for the context: %w", err)
	}

	boundContract := bind.NewBoundContract(
		b.contractAddress,
		b.parsedABI,
		b.client,
		b.client,
		b.client,
	)

	// The actual function name in the contract string
	tx, err := boundContract.Transact(auth, "set", valInt)
	if err != nil {
		return "", fmt.Errorf("error triggering 'set' method: %w", err)
	}

	slog.Info("Besu transaction submitted. Awaiting miner...", slog.String("tx", tx.Hash().Hex()))

	// Awaiting to be officially written into a block
	receipt, err := bind.WaitMined(ctx, b.client, tx)
	if err != nil {
		return "", fmt.Errorf("failed waiting for confirmation in queue: %w", err)
	}
	
	if receipt.Status != 1 {
		return tx.Hash().Hex(), fmt.Errorf("transaction mined but reverted. status=%v", receipt.Status)
	}

	return tx.Hash().Hex(), nil
}

// GetValue executes a dry Call (read-only view) on the "get()" function
func (b *besuClient) GetValue(ctx context.Context) (string, error) {
	caller := &bind.CallOpts{
		Pending: false,
		Context: ctx,
	}

	boundContract := bind.NewBoundContract(
		b.contractAddress,
		b.parsedABI,
		b.client,
		b.client,
		b.client,
	)

	var output []interface{}
	err := boundContract.Call(caller, &output, "get")
	if err != nil {
		return "", fmt.Errorf("failed calling view method 'get()': %w", err)
	}

	if len(output) == 0 {
		return "", fmt.Errorf("contract responded empty")
	}

	// In Solidity the output is a uint256 pointed to the interface pointer *big.Int
	bigVal, ok := output[0].(*big.Int)
	if !ok {
		return "", fmt.Errorf("contract responded, but casting to int array failed incredibly")
	}

	return bigVal.String(), nil
}
