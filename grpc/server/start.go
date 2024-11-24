package server

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pysugar/wheels/grpc/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func StartGrpcServer(port int, serviceName string, serviceRegistry func(*grpc.Server)) error {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Errorf("Failed to listen: %v", err)
		return err
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
	healthServer.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_SERVING)

	grpc_health_v1.RegisterHealthServer(s, healthServer)
	reflection.RegisterV1(s)
	service.RegisterChannelzServiceToServer(s)
	serviceRegistry(s)

	logger.Infof("Server is starting on port :%d...", port)

	if er := s.Serve(lis); er != nil {
		logger.Errorf("Failed to serve: %v", er)
		return err
	}
	return nil
}
