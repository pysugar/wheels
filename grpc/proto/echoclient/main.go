package main

import (
	"context"
	"log"
	"os"
	"time"

	pb "github.com/pysugar/wheels/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/keepalive"
)

const (
	address = "127.0.0.1:50051"
)

func main() {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	kaParams := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kaParams),
	)
	if err != nil {
		logger.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewEchoServiceClient(conn)

	message := "Hello, gRPC!"
	if len(os.Args) > 1 {
		message = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	res, err := c.Echo(ctx, &pb.EchoRequest{Message: message})
	if err != nil {
		log.Fatalf("Could not echo: %v", err)
	}

	log.Printf("Echo from server: %s", res.Message)
}
