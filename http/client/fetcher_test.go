package client

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestFetcher_Do(t *testing.T) {
	serverURLs := []string{
		"http://ipinfo.io/",
		"https://ipinfo.io/",
		"http://ifconfig.me",
		"https://ifconfig.me",
		"http://localhost:8080/grpc/grpc.health.v1.Health/Check",
	}

	for _, serverURL := range serverURLs {
		doGetRequest(t, serverURL)
	}
}

func TestFetcher_H2C_GRPC(t *testing.T) {
	// serverURL, _ := url.Parse("http://localhost:8080/grpc/grpc.health.v1.Health/Check")
	serverURL, _ := url.Parse("http://127.0.0.1:50051/grpc.health.v1.Health/Check")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := &grpchealthv1.HealthCheckRequest{}
	res := &grpchealthv1.HealthCheckResponse{}

	cp := newConnPool()
	f := &fetcher{
		connPool: cp,
	}

	err := f.CallGRPC(ctx, serverURL, req, res)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("res: %+v, err: %v", res, err)
}

func doGetRequest(t *testing.T, rawURL string) {
	serverURL, _ := url.Parse(rawURL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL.String(), nil)
	if err != nil {
		t.Fatal(err)
	}

	cp := newConnPool()
	f := &fetcher{
		connPool: cp,
	}
	res, err := f.Do(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", res.Status)
	body, _ := io.ReadAll(res.Body)
	t.Logf("Body: %s", body)
}
