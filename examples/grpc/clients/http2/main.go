package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/pysugar/wheels/grpc/http2client"
	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	serverURL, _ := url.Parse("https://127.0.0.1:8443")
	// serverURL, _ := url.Parse("http://127.0.0.1:50051")

	client, err := http2client.NewGRPCClient(serverURL)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//for i := 0; i < 100; i++ {
	//	callHealthCheck(ctx, client)
	//}

	var wg sync.WaitGroup
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			callHealthCheck(ctx, client)
		}()
	}
	wg.Wait()
}

func callHealthCheck(ctx context.Context, client http2client.GRPCClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &grpchealthv1.HealthCheckRequest{Service: ""}
	res := &grpchealthv1.HealthCheckResponse{}
	serviceMethod := "grpc.health.v1.Health/Check"

	if err := client.Call(ctx, serviceMethod, req, res); err != nil {
		log.Printf("call service failed: %v", err)
		return
	}
	fmt.Printf("res: %+v\n", res)
}
