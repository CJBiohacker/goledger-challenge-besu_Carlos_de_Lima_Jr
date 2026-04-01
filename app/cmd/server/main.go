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

// server implements the pb.OracleServiceServer interface
type server struct {
	pb.UnimplementedOracleServiceServer
	uc *usecase.OracleUseCase
}

// Set writes a value to the smart contract on the Besu network.
func (s *server) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	if req.GetValue() == "" {
		return nil, status.Error(codes.InvalidArgument, "value cannot be empty")
	}

	txHash, err := s.uc.Set(ctx, req.GetValue())
	if err != nil {
		slog.Error("Set gRPC Failed", slog.Any("err", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.SetResponse{
		Success: true,
		TxHash:  txHash,
	}, nil
}

// Get reads the current value from the smart contract.
func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	val, err := s.uc.Get(ctx)
	if err != nil {
		slog.Error("Get gRPC Failed", slog.Any("err", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetResponse{
		Value: val,
	}, nil
}

// Sync reads the value from the blockchain and saves/synchronizes it to the database (PostgreSQL).
func (s *server) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	syncedVal, err := s.uc.Sync(ctx)
	if err != nil {
		slog.Error("Sync gRPC Failed", slog.Any("err", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.SyncResponse{
		Success:     true,
		SyncedValue: syncedVal,
	}, nil
}

// Check verifies if the value stored in the database matches the blockchain value.
func (s *server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	inSync, dbVal, chainVal, err := s.uc.Check(ctx)
	if err != nil {
		slog.Error("Check gRPC Failed", slog.Any("err", err))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CheckResponse{
		InSync:     inSync,
		DbValue:    dbVal,
		ChainValue: chainVal,
	}, nil
}

func main() {
	// Structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Load variables from .env (ignore if it doesn't exist in a prod server)
	_ = godotenv.Load(".env")
	
	ctx := context.Background()

	// 2. Initialize the Database (SQL Adapter)
	dbURL := os.Getenv("DB_URL")
	pool, err := database.NewPostgresPool(ctx, dbURL)
	if err != nil {
		slog.Error("Failed initializing Postgres Pool", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	repo, err := repository.NewPostgresOracleRepo(ctx, pool)
	if err != nil {
		slog.Error("Failed creating Oracle repository", slog.Any("err", err))
		os.Exit(1)
	}

	// 3. Initialize the Blockchain (Besu Adapter)
	chainCfg := blockchain.Config{
		RPCUrl:          os.Getenv("BESU_RPC_URL"),
		ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		PrivateKey:      os.Getenv("PRIVATE_KEY"),
		ABIPath:         "../SimpleStorage/out/SimpleStorage.sol/SimpleStorage.json",
	}
	
	chainClient, err := blockchain.NewBesuClient(ctx, chainCfg)
	if err != nil {
		slog.Error("Failed creating Besu Blockchain client", slog.Any("err", err))
		os.Exit(1)
	}

	// 4. Inject the two Ports into our Business Logic (Hexagonal)
	uc := usecase.NewOracleUseCase(repo, chainClient)

	// 5. Start gRPC server injecting the UseCase
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		slog.Error("Failed listening on port", slog.String("port", port), slog.Any("error", err))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()

	// Register the service with the real logic coupled through pointers
	pb.RegisterOracleServiceServer(grpcServer, &server{uc: uc})

	// MANDATORY: Enable gRPC Reflection
	reflection.Register(grpcServer)

	slog.Info("gRPC server started and waiting for connections", slog.String("address", lis.Addr().String()))
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("Failed to serve gRPC", slog.Any("error", err))
		os.Exit(1)
	}
}