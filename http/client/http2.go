package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

func (f *fetcher) doHTTP2(ctx context.Context, req *http.Request) (*http.Response, error) {
	logger := newVerboseLogger(ctx)
	var netOpErr *net.OpError

	cc, err := f.tryHTTP2Direct(ctx, req)
	if errors.As(err, &netOpErr) {
		logger.Printf("[%s] try http2 direct failure: %v", req.URL.RequestURI(), netOpErr)
		return nil, netOpErr
	}

	if err == nil && cc != nil {
		logger.Printf("try http2 direct success: %v", req.URL.RequestURI())
		return cc.do(ctx, req)
	}

	cc, err = f.tryHTTP2Upgrade(ctx, req)
	if errors.As(err, &netOpErr) {
		logger.Printf("[%s] try http2 upgrade failure: %v", req.URL.RequestURI(), netOpErr)
		return nil, netOpErr
	}

	if err == nil && cc != nil {
		logger.Printf("try http2 upgrade success: %v", req.URL.RequestURI())
		return cc.do(ctx, req)
	}

	return nil, ErrHTTP2Unsupported
}

func (f *fetcher) tryHTTP2Direct(ctx context.Context, req *http.Request) (*clientConn, error) {
	conn, err := dialConn(req.Host, false)
	if err != nil {
		return nil, err
	}

	if _, er := conn.Write(clientPreface); er != nil {
		conn.Close()
		return nil, fmt.Errorf("[%s] Failed to connect using HTTP/2 Prior Knowledge: %v", req.URL.RequestURI(), er)
	}

	newVerboseLogger(ctx).Printf("[%s] Connect using HTTP/2 Prior Knowledge", req.URL.RequestURI())
	return f.connPool.getConn(ctx, req.URL.Host, WithConn(conn))
}

func (f *fetcher) tryHTTP2Upgrade(ctx context.Context, req *http.Request) (*clientConn, error) {
	conn, err := dialConn(req.Host, false)
	if err != nil {
		return nil, err
	}

	logger := newVerboseLogger(ctx)
	logger.Printf("[%s] Attempting HTTP/2 Upgrade", req.URL.RequestURI())
	err = sendUpgradeRequestHTTP1(conn, req.Method, req.URL)
	if err != nil {
		logger.Printf("Failed to send HTTP/1.1 Upgrade request: %v", err)
		conn.Close()
		return nil, err
	}

	upgraded, err := readUpgradeResponse(conn)
	if err != nil {
		logger.Printf("Failed to read response from Upgrade: %v", err)
		conn.Close()
		return nil, err
	}

	if upgraded {
		logger.Printf("[%s] Successfully upgraded to HTTP/2 (h2c)", req.URL.RequestURI())
		return f.connPool.getConn(ctx, req.URL.Host, WithConn(conn))
	}

	conn.Close()
	logger.Printf("[%s] Server does not support HTTP/2 Upgrade", req.URL.RequestURI())
	return nil, errors.New("server does not support HTTP/2 Upgrade")
}
