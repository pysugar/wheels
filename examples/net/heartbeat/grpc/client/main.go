package main

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"time"

	pb "github.com/pysugar/wheels/examples/net/heartbeat/grpc/heartbeat"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
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
