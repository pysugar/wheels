package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pysugar/wheels/grpc/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func main() {
	var (
		port     = flag.Int("port", 8443, "The server port")
		certFile = flag.String("cert", "../cert/server.crt", "TLS cert file")
		keyFile  = flag.String("key", "../cert/server.key", "TLS key file")
	)
	flag.Parse()

	os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "DEBUG")
	os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "DEBUG")
	os.Setenv("GRPC_TRACE", "all")
	os.Setenv("GRPC_VERBOSITY", "DEBUG")

	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
	if err != nil {
		log.Fatalf("Failed to load TLS credentials: %v", err)
	}

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,  // 最大空闲时间
		MaxConnectionAge:      2 * time.Hour,    // 连接最大存活时间
		MaxConnectionAgeGrace: 5 * time.Minute,  // 优雅关闭时间
		Time:                  1 * time.Hour,    // 发送 ping 的间隔
		Timeout:               20 * time.Second, // ping 超时时间
	}

	s := grpc.NewServer(
		grpc.Creds(creds),
		grpc.KeepaliveParams(kaParams),
		grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor),
		grpc.ChainUnaryInterceptor(interceptors.LoggingUnaryServerInterceptor, grpcprometheus.UnaryServerInterceptor),
	)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("my_service", grpchealthv1.HealthCheckResponse_SERVING)

	grpchealthv1.RegisterHealthServer(s, healthServer)
	reflection.RegisterV1(s)
	service.RegisterChannelzServiceToServer(s)
	grpcprometheus.Register(s)

	go startPrometheus(logger)

	log.Printf("Server is listening on port %d with TLS", *port)
	if er := s.Serve(lis); er != nil {
		log.Fatalf("Failed to serve: %v", er)
	}
}

func startPrometheus(logger grpclog.LoggerV2) {
	http.Handle("/metrics", promhttp.Handler())
	logger.Info("Serving metrics on :9092/metrics")
	if err := http.ListenAndServe(":9092", nil); err != nil {
		logger.Fatalf("Failed to serve metrics: %v", err)
	}
}
