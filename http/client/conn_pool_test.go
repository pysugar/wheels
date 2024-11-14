package client

import (
	"net/url"
	"sync"
	"testing"
)

func TestCallGrpcConcurrency(t *testing.T) {
	serverURL, _ := url.Parse("https://localhost:8443/grpc.health.v1.Health/Check")

	cp := newConnPool()
	var wg sync.WaitGroup
	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cc, err := cp.getConn(serverURL)
			if err != nil {
				t.Error(err)
			}
			callHealthCheck(t, cc, serverURL)
		}()
	}
	wg.Wait()
}
