package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type (
	Fetcher interface {
		Do(context.Context, *http.Request) (*http.Response, error)
		WS(ctx context.Context, req *http.Request) error
		CallGRPC(ctx context.Context, serviceURL *url.URL, req, res proto.Message) error
		Close() error
	}

	fetcher struct {
		userAgent string
		connPool  *connPool
	}
)

var (
	ErrHTTP2Unsupported = errors.New("unsupported protocol http2")
)

func NewFetcher() Fetcher {
	return &fetcher{
		connPool: newConnPool(),
	}
}

func (f *fetcher) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	logger := newVerboseLogger(ctx)
	logger.Printf("[http] upgrade: %v", UpgradeFromContext(ctx))
	logger.Printf("[http] protocol: %v", ProtocolFromContext(ctx))
	logger.Printf("[http] gorilla: %v", GorillaFromContext(ctx))
	useTLS := req.URL.Scheme == "https"
	if useTLS {
		return f.doTLS(ctx, req)
	}
	return f.doHTTP(ctx, req)
}

func (f *fetcher) WS(ctx context.Context, req *http.Request) error {
	logger := newVerboseLogger(ctx)
	logger.Printf("[ws] upgrade: %v", UpgradeFromContext(ctx))
	logger.Printf("[ws] protocol: %v", ProtocolFromContext(ctx))
	logger.Printf("[ws] gorilla: %v", GorillaFromContext(ctx))
	if GorillaFromContext(ctx) {
		return f.doGorilla(ctx, req)
	}
	return f.doWebsocket(ctx, req)
}

func (f *fetcher) CallGRPC(ctx context.Context, serviceURL *url.URL, req, res proto.Message) error {
	logger := newVerboseLogger(ctx)
	logger.Printf("[grpc] upgrade: %v", UpgradeFromContext(ctx))
	logger.Printf("[grpc] protocol: %v", ProtocolFromContext(ctx))
	logger.Printf("[grpc] gorilla: %v", GorillaFromContext(ctx))

	ctx = WithProtocol(ctx, HTTP2)
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	reqBodyBytes := EncodeGrpcPayload(reqBytes)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceURL.String(), bytes.NewReader(reqBodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("content-type", "application/grpc")
	httpReq.Header.Set("te", "trailers")
	httpReq.Header.Set("grpc-encoding", "identity")
	httpReq.Header.Set("grpc-accept-encoding", "identity")

	if logger.Verbose() {
		logger.Printf("\t> %s %s HTTP/1.1\r\n", valueOrDefault(httpReq.Method, http.MethodGet), serviceURL.RequestURI())
		for k, v := range httpReq.Header {
			logger.Printf("\t> %s: %s", k, strings.Join(v, ","))
		}
		logger.Printf("\t> \r\n")
	}

	httpRes, err := f.Do(ctx, httpReq)
	if err != nil {
		return err
	}
	if logger.Verbose() {
		logger.Printf("\t< %s %s\r\n", httpRes.Status, httpRes.Proto)
		for k, v := range httpRes.Header {
			logger.Printf("\t< %s: %s\r\n", k, strings.Join(v, ","))
		}
		for k, v := range httpRes.Trailer {
			logger.Printf("\t< %s: %s\r\n", k, strings.Join(v, ","))
		}
	}

	grpcStatus := 0
	grpcMessage := ""
	if v := httpRes.Trailer.Get("grpc-status"); v != "" {
		grpcStatus, _ = strconv.Atoi(v)
	}
	if v := httpRes.Trailer.Get("grpc-message"); v != "" {
		grpcMessage = v
	}
	st := status.New(codes.Code(grpcStatus), grpcMessage)
	if st.Err() != nil {
		return st.Err()
	}

	resBodyBytes, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return err
	}
	resBytes, err := DecodeGrpcPayload(resBodyBytes)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(resBytes, res)
	if err != nil {
		return err
	}
	return nil
}

func (f *fetcher) doHTTP(ctx context.Context, req *http.Request) (*http.Response, error) {
	protocol := ProtocolFromContext(ctx)
	if protocol == HTTP2 {
		return f.doHTTP2(ctx, req)
	} else if protocol == HTTP1 || protocol == HTTP10 || protocol == HTTP11 {
		return f.doHTTP1(ctx, req)
	} else {
		res, err := f.doHTTP2(ctx, req)
		if !errors.Is(err, ErrHTTP2Unsupported) {
			return res, err
		}
		return f.doHTTP1(ctx, req)
	}
}

func (f *fetcher) doTLS(ctx context.Context, req *http.Request) (*http.Response, error) {
	conn, err := dialConn(req.Host, true)
	if err != nil {
		return nil, err
	}

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return nil, fmt.Errorf("expected *tls.Conn, got %T", conn)
	}

	protocol := ProtocolFromContext(ctx)
	if protocol == HTTP2 {
		return f.doHTTP2WithTLS(ctx, tlsConn, req)
	} else if protocol == HTTP1 || protocol == HTTP10 || protocol == HTTP11 {
		return f.doHTTP1WithConn(ctx, req, conn)
	} else {
		res, er := f.doHTTP2WithTLS(ctx, tlsConn, req)
		if !errors.Is(er, ErrHTTP2Unsupported) {
			return res, er
		}
		return f.doHTTP1WithConn(ctx, req, conn)
	}
}

func (f *fetcher) doHTTP2WithTLS(ctx context.Context, tlsConn *tls.Conn, req *http.Request) (*http.Response, error) {
	state := tlsConn.ConnectionState()
	logger := newVerboseLogger(ctx)
	logger.Printf("NegotiatedProtocol: %s\n", state.NegotiatedProtocol)

	if state.NegotiatedProtocol == "h2" {
		if _, err := tlsConn.Write(clientPreface); err != nil {
			return nil, err
		}
		cc, err := f.connPool.getConn(ctx, req.URL.Host, WithConn(tlsConn), DisableSendPreface())
		if err == nil {
			logger.Printf("[%s] Connect using NegotiatedProtocol", req.URL.RequestURI())
			return cc.do(ctx, req)
		}
		logger.Printf("[%s] Failed to connect using NegotiatedProtocol: %v", req.URL.RequestURI(), err)
	}
	return nil, ErrHTTP2Unsupported
}

func (f *fetcher) Close() error {
	return f.connPool.Close()
}

func dialConn(addr string, useTLS bool) (net.Conn, error) {
	if useTLS {
		if !hasPort(addr) {
			addr += ":443"
		}
		return tls.Dial("tcp", addr, insecureTLSConfig)
	}

	if !hasPort(addr) {
		addr += ":80"
	}
	return net.Dial("tcp", addr)
}

// hasPort checks if the host includes a port
func hasPort(host string) bool {
	_, _, err := net.SplitHostPort(host)
	return err == nil
}
