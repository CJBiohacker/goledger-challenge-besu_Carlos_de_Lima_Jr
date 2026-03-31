package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"goledger-challenge/internal/blockchain"
	"goledger-challenge/internal/database"
	"goledger-challenge/internal/repository"
	"goledger-challenge/internal/usecase"
	"goledger-challenge/pb"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// server implementa a interface pb.OracleServiceServer
type server struct {
	pb.UnimplementedOracleServiceServer
	uc *usecase.OracleUseCase
}

// Set escreve um valor no contrato inteligente na rede Besu.
func (s *server) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	if req.GetValue() == "" {
		return nil, status.Error(codes.InvalidArgument, "o valor não pode ser vazio")
	}

	txHash, err := s.uc.Set(ctx, req.GetValue())
	if err != nil {
		slog.Error("Set gRPC Falhou", slog.Any("err", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.SetResponse{
		Success: true,
		TxHash:  txHash,
	}, nil
}

// Get lê o valor atual do contrato inteligente.
func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	val, err := s.uc.Get(ctx)
	if err != nil {
		slog.Error("Get gRPC Falhou", slog.Any("err", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetResponse{
		Value: val,
	}, nil
}

// Sync lê o valor da blockchain e salva/sincroniza no banco de dados (PostgreSQL).
func (s *server) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	syncedVal, err := s.uc.Sync(ctx)
	if err != nil {
		slog.Error("Sync gRPC Falhou", slog.Any("err", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.SyncResponse{
		Success:     true,
		SyncedValue: syncedVal,
	}, nil
}

// Check verifica se o valor armazenado no banco de dados corresponde ao valor da blockchain.
func (s *server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	inSync, dbVal, chainVal, err := s.uc.Check(ctx)
	if err != nil {
		slog.Error("Check gRPC Falhou", slog.Any("err", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CheckResponse{
		InSync:     inSync,
		DbValue:    dbVal,
		ChainValue: chainVal,
	}, nil
}

func main() {
	// Logger estruturado
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Carrega variaveis do .env (ignora se não existir num server de prod)
	_ = godotenv.Load(".env")
	
	ctx := context.Background()

	// 2. Inicializa o Banco (Adaptador SQL)
	dbURL := os.Getenv("DB_URL")
	pool, err := database.NewPostgresPool(ctx, dbURL)
	if err != nil {
		slog.Error("Falha inicalizando Postgres Pool", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	repo, err := repository.NewPostgresOracleRepo(ctx, pool)
	if err != nil {
		slog.Error("Falha criando o repositorio Oracle", slog.Any("err", err))
		os.Exit(1)
	}

	// 3. Inicializa a Blockchain (Adaptador Besu)
	chainCfg := blockchain.Config{
		RPCUrl:          os.Getenv("BESU_RPC_URL"),
		ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		PrivateKey:      os.Getenv("PRIVATE_KEY"),
		ABIPath:         "../SimpleStorage/out/SimpleStorage.sol/SimpleStorage.json",
	}
	
	chainClient, err := blockchain.NewBesuClient(ctx, chainCfg)
	if err != nil {
		slog.Error("Falha criando cliente da Blockchain Besu", slog.Any("err", err))
		os.Exit(1)
	}

	// 4. Injeta as duas Portas na nossa Regra de Negócio (Hexagonal)
	uc := usecase.NewOracleUseCase(repo, chainClient)

	// 5. Inicia servidor gRPC injetando o UseCase
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		slog.Error("Falha ao escutar na porta", slog.String("port", port), slog.Any("error", err))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()

	// Registra o serviço com a verdadeira lógica acoplada através de ponteiros
	pb.RegisterOracleServiceServer(grpcServer, &server{uc: uc})

	// OBRIGATÓRIO: Habilitar o gRPC Reflection
	reflection.Register(grpcServer)

	slog.Info("Servidor gRPC iniciado e aguardando conexões", slog.String("address", lis.Addr().String()))
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("Falha ao servir gRPC", slog.Any("error", err))
		os.Exit(1)
	}
}