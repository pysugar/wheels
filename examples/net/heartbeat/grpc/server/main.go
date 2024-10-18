package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/pysugar/wheels/examples/net/heartbeat/grpc/heartbeat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedHeartbeatServiceServer
}

func (s *server) Heartbeat(ctx context.Context, in *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	log.Printf("Received heartbeat: %s", in.Message)
	return &pb.HeartbeatResponse{Message: "PONG"}, nil
}

func main() {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      2 * time.Hour,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  1 * time.Hour,
		Timeout:               20 * time.Second,
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.KeepaliveParams(kaParams))

	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	reflection.Register(s)
	service.RegisterChannelzServiceToServer(s)
	pb.RegisterHeartbeatServiceServer(s, &server{})

	log.Printf("Server is listening on port %d", 50051)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
