package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/keepalive"
	"log"
	"os"
	"time"

	pb "github.com/pysugar/wheels/examples/net/heartbeat/grpc/heartbeat"
)

func main() {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	ka := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure(), grpc.WithKeepaliveParams(ka))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewHeartbeatServiceClient(conn)

	for {
		heartbeat(client)
		time.Sleep(10 * time.Second)
	}
}

func heartbeat(client pb.HeartbeatServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Heartbeat(ctx, &pb.HeartbeatRequest{Message: "PING"})
	if err != nil {
		log.Printf("Heartbeat error: %v", err)
	} else {
		log.Printf("Received response: %s", res.Message)
	}
}
