package main

import (
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      2 * time.Hour,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  1 * time.Hour,
		Timeout:               20 * time.Second,
	}

	s := grpc.NewServer(grpc.KeepaliveParams(kaParams))

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	reflection.Register(s)
	service.RegisterChannelzServiceToServer(s)

	healthServer.SetServingStatus("heartbeat", grpc_health_v1.HealthCheckResponse_SERVING)

	defer healthServer.Shutdown()

	log.Printf("Server is listening on port %d", 50051)
	if er := s.Serve(lis); er != nil {
		log.Fatalf("Failed to serve: %v", er)
	}
}
