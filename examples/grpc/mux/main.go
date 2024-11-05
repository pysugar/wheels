package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
		grpc.ChainUnaryInterceptor(grpcprometheus.UnaryServerInterceptor),
	)

	healthServer := health.NewServer()
	grphealthv1.RegisterHealthServer(grpcServer, healthServer)
	reflection.RegisterV1(grpcServer)
	service.RegisterChannelzServiceToServer(grpcServer)
	grpcprometheus.Register(grpcServer)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})
	mux.HandleFunc("/grpc/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s\n", r.Method, r.URL.Path, r.Proto)
		for k, v := range r.Header {
			log.Printf("Header[%q] = %q\n", k, v)
		}
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/grpc")
		grpcServer.ServeHTTP(w, r)
	})
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":8080",
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	log.Println("server listen on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("start server failure: %v", err)
	}
}
