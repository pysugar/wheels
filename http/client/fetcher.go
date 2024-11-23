package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type (
	VerboseLogger interface {
		Printf(format string, v ...any)
	}

	Fetcher interface {
		Do(context.Context, *http.Request) (*http.Response, error)
		Close() error
	}

	fetcher struct {
		userAgent string
		connPool  *connPool
	}

	verboseLogger struct {
		verbose bool
	}
)

func newVerboseLogger(ctx context.Context) VerboseLogger {
	return &verboseLogger{
		verbose: VerboseFromContext(ctx),
	}
}

func (vl *verboseLogger) Printf(format string, v ...any) {
	if vl.verbose {
		log.Printf(format, v...)
	}
}

var (
	ErrHTTP2Unsupported = errors.New("unsupported protocol http2")
)

func NewFetcher() Fetcher {
	return &fetcher{
		connPool: newConnPool(),
	}
}

func (f *fetcher) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	useTLS := req.URL.Scheme == "https"
	if useTLS {
		return f.doTLS(ctx, req)
	}
	return f.doHTTP(ctx, req)
}

func (f *fetcher) Close() error {
	return f.connPool.Close()
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
		cc, err := f.connPool.getConn(ctx, req.URL.Host, WithConn(tlsConn))
		if err == nil {
			logger.Printf("[%s] Connect using NegotiatedProtocol", req.URL.RequestURI())
			return cc.do(ctx, req)
		}
		logger.Printf("[%s] Failed to connect using NegotiatedProtocol: %v", req.URL.RequestURI(), err)
	}
	return nil, ErrHTTP2Unsupported
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

func sendUpgradeRequestHTTP1(conn net.Conn, method string, url *url.URL) error {
	writer := bufio.NewWriter(conn)
	requestLine := fmt.Sprintf("%s %s HTTP/1.1\r\n", method, url.RequestURI())
	if _, err := writer.WriteString(requestLine); err != nil {
		return err
	}
	hostHeader := fmt.Sprintf("Host: %s\r\n", url.Host)
	if _, err := writer.WriteString(hostHeader); err != nil {
		return err
	}
	connectionHeader := "Connection: Upgrade, HTTP2-Settings\r\n"
	if _, err := writer.WriteString(connectionHeader); err != nil {
		return err
	}
	upgradeHeader := "Upgrade: h2c\r\n"
	if _, err := writer.WriteString(upgradeHeader); err != nil {
		return err
	}

	settingPayload := []byte{
		0x00, 0x03, 0x00, 0x00, 0x00, 0x64, // SETTINGS_MAX_CONCURRENT_STREAMS = 100
	}
	http2Settings := base64.StdEncoding.EncodeToString(settingPayload)
	http2SettingsHeader := fmt.Sprintf("HTTP2-Settings: %s\r\n", http2Settings)
	if _, err := writer.WriteString(http2SettingsHeader); err != nil {
		return err
	}
	if _, err := writer.WriteString("\r\n"); err != nil {
		return err
	}
	return writer.Flush()
}

func readUpgradeResponse(conn net.Conn) (bool, error) {
	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	log.Printf("Upgrade Status Line: %s", statusLine)
	if !strings.Contains(statusLine, "101 Switching Protocols") {
		log.Printf("Fail upgraded to HTTP/2 (h2c) >\n")
		return false, nil
	}

	for {
		line, er := reader.ReadString('\n')
		if er != nil {
			return false, er
		}
		log.Print(line)
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	log.Printf("Successfully upgraded to HTTP/2 (h2c) >\n")
	return true, nil
}
