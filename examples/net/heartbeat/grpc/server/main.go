package main

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"net"

	pb "github.com/pysugar/wheels/examples/net/heartbeat/grpc/heartbeat"
)

type server struct {
	pb.UnimplementedHeartbeatServiceServer
}

func (s *server) Heartbeat(ctx context.Context, in *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	log.Printf("Received heartbeat: %s", in.Message)
	return &pb.HeartbeatResponse{Message: "PONG"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterHeartbeatServiceServer(s, &server{})

	log.Printf("Server is listening on port %d", 50051)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
