package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	pb "github.com/pysugar/wheels/grpc/proto"
	"golang.org/x/net/http2"
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
		grpc.ChainUnaryInterceptor(removePrefixInterceptor, grpcprometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpcprometheus.StreamServerInterceptor),
	)

	healthServer := health.NewServer()
	grphealthv1.RegisterHealthServer(grpcServer, healthServer)
	reflection.RegisterV1(grpcServer)
	service.RegisterChannelzServiceToServer(grpcServer)
	grpcprometheus.Register(grpcServer)
	pb.RegisterEchoServiceServer(grpcServer, &server{})

	handler := http.HandlerFunc(grpcServer.ServeHTTP)
	h2s := &http2.Server{}
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request from client: %s", r.RequestURI)
		conn, err := h2cUpgrade(w, r)
		if err != nil {
			log.Println("HTTP/2 Upgrade Failure", err)
			http.Error(w, "HTTP/2 Upgrade Failure", http.StatusInternalServerError)
			return
		}

		go h2s.ServeConn(conn, &http2.ServeConnOpts{
			BaseConfig: &http.Server{
				Handler: handler,
			},
		})
	}
}

func removePrefixInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	originalMethod := info.FullMethod
	log.Printf("Original Method: %v", originalMethod)

	if strings.HasPrefix(originalMethod, "/grpc") {
		info.FullMethod = strings.TrimPrefix(originalMethod, "/grpc")
		log.Printf("Modified Method: %v", info.FullMethod)
	}

	return handler(ctx, req)
}

func h2cUpgrade(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("h2c upgrade server unsupport hijacker")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("h2c upgrade conn hijack failure: %v", err)
	}
	defer rw.Flush()

	response := "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: h2c\r\n\r\n"
	if _, er := rw.WriteString(response); er != nil {
		return nil, fmt.Errorf("send h2c upgrade response failure: %v", er)
	}
	if er := rw.Flush(); er != nil {
		return nil, fmt.Errorf("h2c upgrade flush buffer failure：%v", er)
	}

	return conn, nil
}
