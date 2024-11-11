package client

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

func TestCallGPRC(t *testing.T) {
	serverURL, _ := url.Parse("https://localhost:8443/grpc.health.v1.Health/Check")
	callServiceConcurrency(t, serverURL, 100)

	serverURL2, _ := url.Parse("http://localhost:8080/grpc/grpc.health.v1.Health/Check")
	callServiceConcurrency(t, serverURL2, 100)
}

func callServiceConcurrency(t *testing.T, serverURL *url.URL, concurrent int) {
	cc, err := newClientConn(serverURL)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			callHealthCheck(t, cc, serverURL)
		}()
	}
	wg.Wait()
}

func callHealthCheck(t *testing.T, cc *clientConn, url *url.URL) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req := &grpchealthv1.HealthCheckRequest{}
	res := &grpchealthv1.HealthCheckResponse{}
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resOriginBytes, err := cc.call(ctx, url, EncodeGrpcPayload(reqBytes))
	if err != nil {
		t.Fatal(err)
	}
	resBytes, err := DecodeGrpcPayload(resOriginBytes)
	if err != nil {
		t.Fatal(err)
	}
	err = proto.Unmarshal(resBytes, res)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("res: %+v", res)
}
