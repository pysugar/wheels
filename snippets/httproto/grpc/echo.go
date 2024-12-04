package grpc

import (
	"context"
	"github.com/pysugar/wheels/grpc/interceptors"
	"log"
	"net/http"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	pb "github.com/pysugar/wheels/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/health"
	grphealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedEchoServiceServer
}

func (s *server) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	log.Printf("Received message from client: %s", req.Message)
	return &pb.EchoResponse{Message: req.Message}, nil
}
func NewEchoHandler() http.HandlerFunc {
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      2 * time.Hour,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  1 * time.Hour,
		Timeout:               20 * time.Second,
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.ChainUnaryInterceptor(interceptors.LoggingUnaryServerInterceptor, grpcprometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor),
	)

	healthServer := health.NewServer()
	grphealthv1.RegisterHealthServer(grpcServer, healthServer)
	reflection.RegisterV1(grpcServer)
	service.RegisterChannelzServiceToServer(grpcServer)
	grpcprometheus.Register(grpcServer)
	pb.RegisterEchoServiceServer(grpcServer, &server{})

	return grpcServer.ServeHTTP
}
