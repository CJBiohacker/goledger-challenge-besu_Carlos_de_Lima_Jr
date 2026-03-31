package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"goledger-challenge/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// server implementa a interface pb.OracleServiceServer
type server struct {
	pb.UnimplementedOracleServiceServer
}

// Set escreve um valor no contrato inteligente na rede Besu.
func (s *server) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	slog.Info("Recebida requisição Set", slog.String("value", req.GetValue()))
	
	// TODO: Implementar lógica de integração com a blockchain (Besu)
	if req.GetValue() == "" {
		return nil, status.Error(codes.InvalidArgument, "o valor não pode ser vazio")
	}

	return &pb.SetResponse{
		Success: true,
		TxHash:  "mock_tx_hash_aqui",
	}, nil
}

// Get lê o valor atual do contrato inteligente.
func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	slog.Info("Recebida requisição Get")
	
	// TODO: Implementar leitura da blockchain
	return &pb.GetResponse{
		Value: "mock_value_chain",
	}, nil
}

// Sync lê o valor da blockchain e salva/sincroniza no banco de dados (PostgreSQL).
func (s *server) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	slog.Info("Recebida requisição Sync")
	
	// TODO: 1. Ler da blockchain, 2. Atualizar no DB
	return &pb.SyncResponse{
		Success:     true,
		SyncedValue: "mock_value_synced",
	}, nil
}

// Check verifica se o valor armazenado no banco de dados corresponde ao valor da blockchain.
func (s *server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	slog.Info("Recebida requisição Check")
	
	// TODO: Comparar DB e Chain
	return &pb.CheckResponse{
		InSync:     true,
		DbValue:    "mock_value",
		ChainValue: "mock_value",
	}, nil
}

func main() {
	// Logger estruturado usando json conforme boas práticas do projeto
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		slog.Error("Falha ao escutar na porta", slog.String("port", port), slog.Any("error", err))
		os.Exit(1)
	}

	// Cria o servidor gRPC
	grpcServer := grpc.NewServer()

	// Registra o serviço mockado
	pb.RegisterOracleServiceServer(grpcServer, &server{})

	// OBRIGATÓRIO: Habilitar o gRPC Reflection para facilitar os testes via grpcurl, Postman, etc.
	reflection.Register(grpcServer)

	slog.Info("Servidor gRPC iniciado e aguardando conexões", slog.String("address", lis.Addr().String()))
	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("Falha ao servir gRPC", slog.Any("error", err))
		os.Exit(1)
	}
}