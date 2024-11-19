package client

import (
	"bytes"
	"context"
	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestFetcher_Do(t *testing.T) {
	// serverURL, _ := url.Parse("http://ipinfo.io/")
	// serverURL, _ := url.Parse("https://ipinfo.io/")
	serverURL, _ := url.Parse("http://ifconfig.me/")
	// serverURL, _ := url.Parse("https://ifconfig.me/")
	// serverURL, _ := url.Parse("http://localhost:8080/grpc/grpc.health.v1.Health/Check")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL.String(), nil)
	if err != nil {
		t.Fatal(err)
	}

	cp := newConnPool()
	cp.verbose = true
	f := &fetcher{
		verbose:  true,
		connPool: cp,
	}
	res, err := f.Do(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", res)
}

func TestFetcher_H2C_GRPC(t *testing.T) {
	// serverURL, _ := url.Parse("http://localhost:8080/grpc/grpc.health.v1.Health/Check")
	serverURL, _ := url.Parse("http://127.0.0.1:50051/grpc.health.v1.Health/Check")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &grpchealthv1.HealthCheckRequest{}
	res := &grpchealthv1.HealthCheckResponse{}
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", serverURL.String(), bytes.NewReader(EncodeGrpcPayload(reqBytes)))
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set("content-type", "application/grpc")
	httpReq.Header.Set("te", "trailers")
	httpReq.Header.Set("grpc-encoding", "identity")
	httpReq.Header.Set("grpc-accept-encoding", "identity")

	cp := newConnPool()
	cp.verbose = true
	f := &fetcher{
		verbose:  true,
		connPool: cp,
	}

	httpRes, err := f.Do(ctx, httpReq)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", httpRes)

	resOriginBytes, err := io.ReadAll(httpRes.Body)
	if err != nil {
		t.Fatal(err)
	}

	resBytes, err := DecodeGrpcPayload(resOriginBytes)
	if err != nil {
		t.Fatalf("resBytes: %s, error: %v", resBytes, err)
	}
	err = proto.Unmarshal(resBytes, res)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("res: %+v, err: %v", res, err)
}
