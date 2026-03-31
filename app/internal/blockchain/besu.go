package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ChainClient é a "Porta" da Arquitetura Hexagonal para a Blockchain.
type ChainClient interface {
	SetValue(ctx context.Context, value string) (string, error)
	GetValue(ctx context.Context) (string, error)
}

// besuClient é o "Adaptador" de fato usando go-ethereum
type besuClient struct {
	client          *ethclient.Client
	contractAddress common.Address
	parsedABI       abi.ABI
	privateKey      string
	chainID         *big.Int
}

// Config centraliza dependências do construtor
type Config struct {
	RPCUrl          string
	ContractAddress string
	PrivateKey      string
	ABIPath         string // Caminho para SimpleStorage.json
}

// foundryOutput auxilia a destrinchar o JSON massivo num unmarshal
type foundryOutput struct {
	ABI json.RawMessage `json:"abi"`
}

// NewBesuClient é a factory pattern
func NewBesuClient(ctx context.Context, cfg Config) (ChainClient, error) {
	// 1. Conecta ao RPC do Besu
	client, err := ethclient.DialContext(ctx, cfg.RPCUrl)
	if err != nil {
		return nil, fmt.Errorf("erro conectando no Besu RPC: %w", err)
	}

	// 2. Coleta ChainID da rede na qual conectamos (geralmente local=1337)
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro lendo chain id: %w", err)
	}

	// 3. Lê e faz o Parse do ABI da compilação do Foundry em disco
	rawFile, err := os.ReadFile(cfg.ABIPath)
	if err != nil {
		return nil, fmt.Errorf("erro lendo arquivo json da ABI no path %s: %w", cfg.ABIPath, err)
	}

	var output foundryOutput
	if err := json.Unmarshal(rawFile, &output); err != nil {
		return nil, fmt.Errorf("falha extraindo 'abi' do json: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(string(output.ABI)))
	if err != nil {
		return nil, fmt.Errorf("erro transformando interface ABI para pacote eth: %w", err)
	}

	slog.Info("Besu Blockchain Client adaptado em memória", slog.String("chain_id", chainID.String()))

	return &besuClient{
		client:          client,
		contractAddress: common.HexToAddress(cfg.ContractAddress),
		parsedABI:       parsedABI,
		privateKey:      cfg.PrivateKey,
		chainID:         chainID,
	}, nil
}

// SetValue escreve "set(uint256)" e aguarda mineração, retornando a hash.
func (b *besuClient) SetValue(ctx context.Context, value string) (string, error) {
	// A função na chain espera um número grande (uint256)
	valInt, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return "", fmt.Errorf("valor de input nao e um numero inteiro valido string=%s", value)
	}

	// Assinatura usando chave privada
	priv, err := crypto.HexToECDSA(b.privateKey)
	if err != nil {
		return "", fmt.Errorf("erro lendo chave privada ECSDA: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(priv, b.chainID)
	if err != nil {
		return "", fmt.Errorf("erro criando transactor para o context: %w", err)
	}

	boundContract := bind.NewBoundContract(
		b.contractAddress,
		b.parsedABI,
		b.client,
		b.client,
		b.client,
	)

	// O nome da função real na string do contrato
	tx, err := boundContract.Transact(auth, "set", valInt)
	if err != nil {
		return "", fmt.Errorf("erro disparando metodo 'set': %w", err)
	}

	slog.Info("Transação Besu submetida. Aguardando mineiro...", slog.String("tx", tx.Hash().Hex()))

	// Aguardando ser escrito num bloco oficialmente
	receipt, err := bind.WaitMined(ctx, b.client, tx)
	if err != nil {
		return "", fmt.Errorf("falha aguardando confirmacao na fila: %w", err)
	}
	
	if receipt.Status != 1 {
		return tx.Hash().Hex(), fmt.Errorf("transacao minerada porem revertida. status=%v", receipt.Status)
	}

	return tx.Hash().Hex(), nil
}

// GetValue faz a Call seca (read-only view) na função "get()"
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
		return "", fmt.Errorf("falha chamando metodo view 'get()': %w", err)
	}

	if len(output) == 0 {
		return "", fmt.Errorf("o contrato respondeu vazio")
	}

	// Em Solidity o output é um uint256 apontado pro ponteiro de interface *big.Int
	bigVal, ok := output[0].(*big.Int)
	if !ok {
		return "", fmt.Errorf("contrato respondeu, mas casting para array int falhou incrivelmente")
	}

	return bigVal.String(), nil
}
