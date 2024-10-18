package main

import (
	"context"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

const (
	address           = "localhost:50051"
	heartbeatInterval = 10 * time.Second
)

func main() {
	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	ka := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             3 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithKeepaliveParams(ka))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startHeartbeat(appCtx, client)

	select {}
}

func startHeartbeat(ctx context.Context, client grpc_health_v1.HealthClient) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case v, ok := <-ctx.Done():
			log.Printf("exit heartbeat since context done, v: %v(%v)\n", v, ok)
			return
		case v, ok := <-ticker.C:
			log.Printf("[%v]Tick start: %v\n", v, ok)
			doHeartbeat(ctx, client)
		}
	}
}

func doHeartbeat(appCtx context.Context, client grpc_health_v1.HealthClient) {
	ctx, cancel := context.WithTimeout(appCtx, 5*time.Second)
	defer cancel()

	res, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "heartbeat"})
	if err != nil {
		log.Printf("Heartbeat error: %v", err)
	} else {
		log.Printf("Received response: %v", res.Status)
	}
}
