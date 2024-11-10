package main

import (
	"fmt"
	"github.com/pysugar/wheels/protocol/http/extensions"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pysugar/wheels/grpc/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health"
	grphealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func main() {
	fmt.Printf("GODEBUG = %s\n", os.Getenv("GODEBUG"))
	os.Setenv("GODEBUG", "http2debug=1")
	os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "DEBUG")
	os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "DEBUG")
	os.Setenv("GRPC_TRACE", "all")
	os.Setenv("GRPC_VERBOSITY", "DEBUG")

	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      2 * time.Hour,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  1 * time.Hour,
		Timeout:               20 * time.Second,
	}

	var grpcServer = grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor),
		grpc.ChainUnaryInterceptor(grpcprometheus.UnaryServerInterceptor, interceptors.LoggingUnaryServerInterceptor),
	)
	healthServer := health.NewServer()

	grphealthv1.RegisterHealthServer(grpcServer, healthServer)
	reflection.RegisterV1(grpcServer)
	service.RegisterChannelzServiceToServer(grpcServer)
	grpcprometheus.Register(grpcServer)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("**************** %s %s %s ******************\n", r.Method, r.URL.RequestURI(), r.Proto)
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			switch r.URL.Path {
			case "/health":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			case "/metrics":
				promhttp.Handler().ServeHTTP(w, r)
			default:
				http.NotFound(w, r)
			}
		}
	})

	h2cHandler := h2c.NewHandler(extensions.LoggingMiddleware(handler), &http2.Server{})
	server := &http.Server{
		Addr:    ":8080",
		Handler: h2cHandler,
		// Handler: h2c.NewHandler(handler, &http2.Server{}),
	}

	log.Println("server listen on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("start server failure: %v", err)
	}
}
