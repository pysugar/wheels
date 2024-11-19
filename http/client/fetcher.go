package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

//type connPool struct {
//	mu    sync.Mutex               // TODO: maybe switch to RWMutex
//	conns map[string][]*clientConn // key is host:port
//}

// import (
//
//	"context"
//	"github.com/pysugar/wheels/binproto/http2"
//	"github.com/pysugar/wheels/concurrent"
//	pb "google.golang.org/grpc/health/grpc_health_v1"
//	"log"
//	"net"
//	"net/url"
//	"sync/atomic"
//
// )
//
// const (
//
//	ClientPreface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
//
// )
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
	conn, err := dialConn(req.Host, false)
	if err != nil {
		return nil, err
	}

	if _, er := conn.Write(clientPreface); er == nil {
		f.printf("[%s] Connect using HTTP/2 Prior Knowledge", req.URL.RequestURI())

		cc, e := f.connPool.getConn(ctx, req.URL.Host, WithConn(conn))
		if e == nil {
			return cc.do(ctx, req)
		} else {
			f.printf("[%s] get client conn failure: %v", req.URL.RequestURI(), e)
			f.printf("close conn, err: %v ", conn.Close())
		}
	} else {
		f.printf("[%s] Failed to connect using HTTP/2 Prior Knowledge: %v", req.URL.RequestURI(), er)
	}

	conn, err = dialConn(req.Host, false)
	if err != nil {
		return nil, err
	}

	err = sendUpgradeRequestHTTP1(conn, req.Method, req.URL)
	if err != nil {
		log.Println("Failed to send HTTP/1.1 Upgrade request:", err)
		return nil, err
	}

	// Read the server's response to the upgrade request
	upgraded, err := readUpgradeResponse(conn)
	if err != nil {
		log.Printf("Fail to read response from Upgrade: %v\n", err)
	}

	if upgraded {
		cc, er := f.connPool.getConn(ctx, req.URL.Host, WithConn(conn))
		if er != nil {
			return nil, err
		}

		return cc.do(ctx, req)
	}

	return nil, nil
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
