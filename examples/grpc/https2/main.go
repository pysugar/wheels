package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	grpcextensions "github.com/pysugar/wheels/grpc/extensions"
	httpextensions "github.com/pysugar/wheels/protocol/http/extensions"
	"google.golang.org/grpc"
	"google.golang.org/grpc/channelz/service"
	"google.golang.org/grpc/health"
	grphealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func main() {
	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      2 * time.Hour,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  1 * time.Hour,
		Timeout:               20 * time.Second,
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kaParams),
		grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor),
		grpc.ChainUnaryInterceptor(grpcprometheus.UnaryServerInterceptor, grpcextensions.LoggingUnaryServerInterceptor),
	)

	healthServer := health.NewServer()

	grphealthv1.RegisterHealthServer(grpcServer, healthServer)
	reflection.RegisterV1(grpcServer)
	service.RegisterChannelzServiceToServer(grpcServer)
	grpcprometheus.Register(grpcServer)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprintln(w, "OK")
		} else if r.URL.Path == "/metrics" {
			promhttp.Handler().ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	server := &http.Server{
		Addr:    ":8443",
		Handler: httpextensions.LoggingMiddleware(grpcMiddleware(grpcServer, handler)),
	}

	log.Println("server listen on :8443")
	if err := server.ListenAndServeTLS("server.crt", "server.key"); err != nil {
		log.Fatalf("start server failure: %v", err)
	}
}

func grpcMiddleware(grpcServer *grpc.Server, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
