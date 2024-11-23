package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

func (f *fetcher) doHTTP2(ctx context.Context, req *http.Request) (*http.Response, error) {
	cc, err := f.tryHTTP2Direct(ctx, req)
	if err == nil && cc != nil {
		f.printf("try http2 direct failure: %v", req.URL.RequestURI())
		return cc.do(ctx, req)
	}

	cc, err = f.tryHTTP2Upgrade(ctx, req)
	if err == nil && cc != nil {
		f.printf("try http2 upgrade failure: %v", req.URL.RequestURI())
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

	f.printf("[%s] Connect using HTTP/2 Prior Knowledge", req.URL.RequestURI())
	return f.connPool.getConn(ctx, req.URL.Host, WithConn(conn))
}

func (f *fetcher) tryHTTP2Upgrade(ctx context.Context, req *http.Request) (*clientConn, error) {
	conn, err := dialConn(req.Host, false)
	if err != nil {
		return nil, err
	}

	f.printf("[%s] Attempting HTTP/2 Upgrade", req.URL.RequestURI())
	err = sendUpgradeRequestHTTP1(conn, req.Method, req.URL)
	if err != nil {
		f.printf("Failed to send HTTP/1.1 Upgrade request: %v", err)
		conn.Close()
		return nil, err
	}

	upgraded, err := readUpgradeResponse(conn)
	if err != nil {
		f.printf("Failed to read response from Upgrade: %v", err)
		conn.Close()
		return nil, err
	}

	if upgraded {
		f.printf("[%s] Successfully upgraded to HTTP/2 (h2c)", req.URL.RequestURI())
		return f.connPool.getConn(ctx, req.URL.Host, WithConn(conn))
	}

	conn.Close()
	f.printf("[%s] Server does not support HTTP/2 Upgrade", req.URL.RequestURI())
	return nil, errors.New("server does not support HTTP/2 Upgrade")
}
