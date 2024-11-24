package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/pysugar/wheels/grpc/interceptors"
	pb "github.com/pysugar/wheels/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":50051"
)

type server struct {
	pb.UnimplementedEchoServiceServer
}

func (s *server) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	log.Printf("Received message from client: %s", req.Message)
	return &pb.EchoResponse{Message: req.Message}, nil
}

func main() {
	os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "DEBUG")
	os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "DEBUG")
	os.Setenv("GRPC_TRACE", "all")
	os.Setenv("GRPC_VERBOSITY", "DEBUG")

	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		logger.Fatalf("Failed to listen: %v", err)
	}

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      2 * time.Hour,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  1 * time.Hour,
		Timeout:               20 * time.Second,
	}

	s := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.ChainUnaryInterceptor(interceptors.LoggingUnaryServerInterceptor),
	)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("echo-service", grpc_health_v1.HealthCheckResponse_SERVING)

	grpc_health_v1.RegisterHealthServer(s, healthServer)
	reflection.RegisterV1(s)
	service.RegisterChannelzServiceToServer(s)
	pb.RegisterEchoServiceServer(s, &server{})

	logger.Infof("Server is starting on port %s...", port)

	if er := s.Serve(lis); er != nil {
		logger.Fatalf("Failed to serve: %v", er)
	}
}
