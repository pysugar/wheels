package client

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func (f *fetcher) doHTTP2(ctx context.Context, req *http.Request) (*http.Response, error) {
	logger := newVerboseLogger(ctx)
	var netOpErr *net.OpError

	upgrade := UpgradeFromContext(ctx)
	if !upgrade {
		cc, err := f.tryHTTP2Direct(ctx, req)
		if errors.As(err, &netOpErr) {
			logger.Printf("[%s] try http2 direct failure: %v", req.URL.RequestURI(), netOpErr)
			return nil, netOpErr
		}

		if err == nil && cc != nil {
			logger.Printf("try http2 direct success: %v", req.URL.RequestURI())
			return cc.do(ctx, req)
		}

		logger.Printf("[%s] try http2 direct failure: %v", req.URL.RequestURI(), err)
	}

	cc, err := f.tryHTTP2Upgrade(ctx, req)
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
	return f.connPool.getConn(ctx, req.URL.Host, WithConn(conn), DisableSendPreface())
}

func (f *fetcher) tryHTTP2Upgrade(ctx context.Context, req *http.Request) (*clientConn, error) {
	conn, err := dialConn(req.Host, false)
	if err != nil {
		return nil, err
	}

	logger := newVerboseLogger(ctx)
	logger.Printf("< [%s] Attempting HTTP/2 Upgrade", req.URL.RequestURI())
	err = sendUpgradeRequestHTTP1(ctx, conn, req)
	if err != nil {
		logger.Printf("[%s] Failed to send HTTP/1.1 Upgrade request: %v", req.URL.RequestURI(), err)
		conn.Close()
		return nil, err
	}

	upgraded, err := readUpgradeResponse(ctx, conn)
	if err != nil {
		logger.Printf("[%s] Failed to read response from Upgrade: %v", req.URL.RequestURI(), err)
		conn.Close()
		return nil, err
	}

	if upgraded {
		logger.Printf("[%s] Successfully upgraded to HTTP/2 (h2c)", req.URL.RequestURI())
		return f.connPool.getConn(ctx, req.URL.Host, WithConn(conn), WithH2CUpgrade())
	}

	logger.Printf("[%s] Server does not support HTTP/2 Upgrade", req.URL.RequestURI())
	conn.Close()
	return nil, errors.New("server does not support HTTP/2 Upgrade")
}

func sendUpgradeRequestHTTP1(ctx context.Context, conn net.Conn, r *http.Request) error {
	logger := newVerboseLogger(ctx)

	w := bufio.NewWriter(conn)

	ruri := r.URL.RequestURI()
	logger.Printf("\t> %s %s HTTP/1.1\r\n", valueOrDefault(r.Method, http.MethodGet), ruri)
	if _, err := fmt.Fprintf(w, "%s %s HTTP/1.1\r\n", valueOrDefault(r.Method, http.MethodGet), ruri); err != nil {
		return err
	}

	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	host = removeZone(host)

	r.Header.Set("Host", host)
	r.Header.Set("Connection", "Upgrade, HTTP2-Settings")
	r.Header.Set("Upgrade", "h2c")
	settingPayload := []byte{
		0x00, 0x01, 0x00, 0x00, 0x04, 0x00, // SETTINGS_HEADER_TABLE_SIZE = 1024
		0x00, 0x02, 0x00, 0x00, 0x00, 0x00, // SETTINGS_ENABLE_PUSH = 0
		0x00, 0x03, 0x00, 0x00, 0x00, 0x64, // SETTINGS_MAX_CONCURRENT_STREAMS = 100
	}
	http2Settings := base64.StdEncoding.EncodeToString(settingPayload)
	r.Header.Set("HTTP2-Settings", http2Settings)
	if r.Body != nil {
		r.Header.Set("Content-Length", strconv.FormatInt(outgoingLength(r), 10))
	}

	for k, v := range r.Header {
		logger.Printf("\t> %s: %s", k, strings.Join(v, ","))
	}
	if err := r.Header.Write(w); err != nil {
		return err
	}

	logger.Printf("\t> \r\n")
	if _, err := w.WriteString("\r\n"); err != nil {
		return err
	}

	if r.Body != nil {
		logger.Printf("\t> <request body>")
		if _, err := io.Copy(w, r.Body); err != nil {
			return err
		}
	}

	return w.Flush()
}

func readUpgradeResponse(ctx context.Context, conn net.Conn) (bool, error) {
	logger := newVerboseLogger(ctx)

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	logger.Printf("\t< %s", statusLine)
	if !strings.Contains(statusLine, "101 Switching Protocols") {
		log.Printf("Fail upgraded to HTTP/2 (h2c) >\n")
		return false, nil
	}

	for {
		line, er := reader.ReadString('\n')
		if er != nil {
			return false, er
		}
		logger.Printf("\t< %s", line)
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	logger.Printf("Successfully upgraded to HTTP/2 (h2c) >\n")
	return true, nil
}

func outgoingLength(r *http.Request) int64 {
	if r.Body == nil || r.Body == http.NoBody {
		return 0
	}
	if r.ContentLength != 0 {
		return r.ContentLength
	}
	return -1
}

func removeZone(host string) string {
	if !strings.HasPrefix(host, "[") {
		return host
	}
	i := strings.LastIndex(host, "]")
	if i < 0 {
		return host
	}
	j := strings.LastIndex(host[:i], "%")
	if j < 0 {
		return host
	}
	return host[:j] + host[i:]
}

func valueOrDefault(value, def string) string {
	if value != "" {
		return value
	}
	return def
}
