package client

import (
	"context"
	"net/url"
	"sync"
	"testing"
)

func TestCallGrpcConcurrency(t *testing.T) {
	serverURL, _ := url.Parse("https://localhost:8443/grpc.health.v1.Health/Check")

	cp := newConnPool()
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			cc, err := cp.getConn(ctx, serverURL.Host, WithTLS())
			if err != nil {
				t.Errorf("getConn err: %v", err)
				return
			}
			callHealthCheck(t, cc, serverURL)
		}()
	}
	wg.Wait()
}
