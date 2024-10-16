package main

import (
	"context"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"net"
	"net/http"
	"os"
	"time"
)

const (
	port = ":50051"
)

func main() {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		logger.Fatalf("Failed to listen: %v", err)
	}

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,  // 最大空闲时间
		MaxConnectionAge:      2 * time.Hour,    // 连接最大存活时间
		MaxConnectionAgeGrace: 5 * time.Minute,  // 优雅关闭时间
		Time:                  1 * time.Hour,    // 发送 ping 的间隔
		Timeout:               20 * time.Second, // ping 超时时间
	}

	s := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpcprometheus.UnaryServerInterceptor),
	)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("my_service", grpc_health_v1.HealthCheckResponse_SERVING)

	grpc_health_v1.RegisterHealthServer(s, healthServer)
	reflection.Register(s)
	service.RegisterChannelzServiceToServer(s)
	grpcprometheus.Register(s)

	go startPrometheus(logger)

	logger.Infof("Server is starting on port %s...", port)

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go startChaos(appCtx, healthServer, logger)

	if er := s.Serve(lis); er != nil {
		logger.Fatalf("Failed to serve: %v", er)
	}
}

// 启动一个 goroutine 来模拟服务状态的变化（可选）
func startChaos(ctx context.Context, server *health.Server, logger grpclog.LoggerV2) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(30 * time.Second)
			server.SetServingStatus("my_service", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
			logger.Info("Service status set to NOT_SERVING")
			time.Sleep(30 * time.Second)
			server.SetServingStatus("my_service", grpc_health_v1.HealthCheckResponse_SERVING)
			logger.Info("Service status set to SERVING")
		}
	}
}

func startPrometheus(logger grpclog.LoggerV2) {
	http.Handle("/metrics", promhttp.Handler())
	logger.Info("Serving metrics on :9092/metrics")
	if err := http.ListenAndServe(":9092", nil); err != nil {
		logger.Fatalf("Failed to serve metrics: %v", err)
	}
}
