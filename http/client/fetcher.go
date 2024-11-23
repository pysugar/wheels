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
	Fetcher interface {
		Close()
	}

	fetcher struct {
		userAgent string
		verbose   bool
		connPool  *connPool
	}
)

func (f *fetcher) printf(format string, v ...any) {
	if f.verbose {
		log.Printf(format, v...)
	}
}

func (f *fetcher) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	useTLS := req.URL.Scheme == "https"
	if useTLS {
		return f.doTLS(ctx, req)
	}
	return f.doHTTP(ctx, req)
}

func (f *fetcher) doHTTP(ctx context.Context, req *http.Request) (*http.Response, error) {
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

	return f.doHTTP1(ctx, req)
}

func (f *fetcher) doTLS(ctx context.Context, req *http.Request) (*http.Response, error) {
	conn, err := dialConn(req.Host, true)
	if err != nil {
		return nil, err
	}

	if c, ok := conn.(*tls.Conn); ok {
		state := c.ConnectionState()
		fmt.Printf("NegotiatedProtocol: %s\n", state.NegotiatedProtocol)
		if state.NegotiatedProtocol == "h2" {
			if _, er := conn.Write(clientPreface); er != nil {
				return nil, er
			}

			cc, e := f.connPool.getConn(ctx, req.URL.Host, WithConn(conn))
			if e == nil {
				f.printf("[%s] Connect using NegotiatedProtocol", req.URL.RequestURI())
				return cc.do(ctx, req)
			}
			f.printf("[%s] Failed to connect using NegotiatedProtocol: %v", req.URL.RequestURI(), err)
		}
	}

	return nil, nil
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

func sendUpgradeRequestHTTP1(conn net.Conn, method string, parsedURL *url.URL) error {
	host := parsedURL.Host
	path := parsedURL.RequestURI()

	// Generate HTTP2-Settings header value with specific SETTINGS frame (base64 encoded)
	settingPayload := []byte{
		// SETTINGS payload:
		0x00, 0x03, 0x00, 0x00, 0x00, 0x64, // SETTINGS_MAX_CONCURRENT_STREAMS = 100
	}
	http2Settings := base64.StdEncoding.EncodeToString(settingPayload)

	log.Println("< Sent HTTP/1.1 Upgrade request")
	// Create HTTP/1.1 Upgrade request
	request := fmt.Sprintf(
		"%s %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"User-Agent: curl/8.7.1\r\n"+
			"Accept: */*\r\n"+
			"Connection: Upgrade, HTTP2-Settings\r\n"+
			"Upgrade: h2c\r\n"+
			"HTTP2-Settings: %s\r\n\r\n",
		method, path, host, http2Settings)

	//request := fmt.Sprintf(
	//	"OPTIONS * HTTP/1.1\r\n"+
	//		"Host: %s\r\n"+
	//		"Connection: Upgrade, HTTP2-Settings\r\n"+
	//		"Upgrade: h2c\r\n"+
	//		"HTTP2-Settings: %s\r\n"+
	//		"\r\n",
	//	host, http2Settings)

	log.Printf("%s\n", request)
	if _, err := conn.Write([]byte(request)); err != nil {
		return err
	}
	return nil
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

	// Read headers until an empty line
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
