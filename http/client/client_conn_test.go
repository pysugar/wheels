package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

// export GODEBUG=http2debug=1
func TestCallGPRC(t *testing.T) {
	//serverURL, _ := url.Parse("https://localhost:8443/grpc.health.v1.Health/Check")
	//callServiceConcurrency(t, serverURL, 300)

	serverURL2, _ := url.Parse("http://localhost:8080/grpc/grpc.health.v1.Health/Check")
	callServiceConcurrency(t, serverURL2, 300)

	//serverURL3, _ := url.Parse("http://localhost:8080/grpc.health.v1.Health/Check")
	//callServiceConcurrency(t, serverURL3, 1)
}

func TestCallHTTP2(t *testing.T) {
	ipinfoURL, _ := url.Parse("https://ipinfo.io/")
	cc, err := newClientConn(ipinfoURL)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ipinfoURL.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := cc.do(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s %s", res.Status, res.Proto)
	for n, v := range res.Header {
		t.Logf("Header %s: %v", n, v)
	}
	for n, v := range res.Trailer {
		t.Logf("Trailer %s: %v", n, v)
	}
	body, _ := io.ReadAll(res.Body)
	t.Logf("Body: %s", body)
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url.String(), bytes.NewReader(EncodeGrpcPayload(reqBytes)))
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set("content-type", "application/grpc")
	httpReq.Header.Set("te", "trailers")
	httpReq.Header.Set("grpc-encoding", "identity")
	httpReq.Header.Set("grpc-accept-encoding", "identity")

	httpRes, err := cc.do(ctx, httpReq)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s %s", httpRes.Status, httpRes.Proto)
	for n, v := range httpRes.Header {
		t.Logf("Header %s: %v", n, v)
	}
	for n, v := range httpRes.Trailer {
		t.Logf("Trailer %s: %v", n, v)
	}

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
