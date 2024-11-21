package client

import (
	"context"
	"math/rand"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestCallGrpcConcurrency(t *testing.T) {
	serverURL, _ := url.Parse("https://localhost:8443/grpc.health.v1.Health/Check")

	cp := newConnPool()
	cp.verbose = true
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			time.Sleep(time.Millisecond * time.Duration(rand.Int()%100))
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

func TestCallHTTP(t *testing.T) {
	serverURL, _ := url.Parse("https://ipinfo.io")

	cp := newConnPool()
	cp.verbose = true
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			time.Sleep(time.Millisecond * time.Duration(rand.Int()%100))
			cc, err := cp.getConn(ctx, serverURL.Host, WithTLS())
			if err != nil {
				t.Errorf("getConn err: %v", err)
				return
			}
			callHTTP2(t, cc, serverURL)
		}()
	}
	wg.Wait()
}
