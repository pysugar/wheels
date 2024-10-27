package subcmds

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pysugar/wheels/binproto/http2"
	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
)

type (
	fetchClient struct {
		userAgent string
	}
)

var (
	streamID = uint32(1)
	fetchCmd = &cobra.Command{
		Use:   `fetch https://www.google.com`,
		Short: "fetch http2 response from url",
		Long: `
fetch http2 response from url

fetch http2 response from url: netool fetch https://www.google.com
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				log.Printf("you must specify the url")
				return
			}

			targetURL, err := url.Parse(args[0])
			if err != nil {
				log.Printf("invalid url %s\n", args[0])
				return
			}

			conn, err := dialConn(targetURL)
			if err != nil {
				log.Printf("dial conn err: %v\n", err)
				return
			}
			defer conn.Close()

			if c, ok := conn.(*tls.Conn); ok {
				state := c.ConnectionState()
				log.Printf("* TLS Handshake state: \n")
				log.Printf("* \tVersion: %v\n", state.Version)
				log.Printf("* \tServerName: %v\n", state.ServerName)
				log.Printf("* \tNegotiatedProtocol: %v\n", state.NegotiatedProtocol)
				for _, cert := range state.PeerCertificates {
					log.Printf("* \tCertificate Version: %v\n", cert.Version)
					log.Printf("* \tCertificate Subject: %v\n", cert.Subject)
					log.Printf("* \tCertificate Issuer: %v\n", cert.Issuer)
					log.Printf("* \tCertificate SignatureAlgorithm: %v\n", cert.SignatureAlgorithm)
					log.Printf("* \tCertificate PublicKeyAlgorithm: %v\n", cert.PublicKeyAlgorithm)
					log.Printf("* \tCertificate NotBefore: %v\n", cert.NotBefore)
					log.Printf("* \tCertificate NotAfter: %v\n", cert.NotAfter)
				}

				if state.NegotiatedProtocol == "h2" {
					log.Println("Successfully negotiated HTTP/2")

					doneCh := make(chan struct{})
					go func() {
						defer close(doneCh)
						readLoop(conn)
					}()

					//err = sendSettingsFrame(conn)
					//if err != nil {
					//	log.Printf("Failed to send SETTINGS frame: %v\n", err)
					//	return
					//}

					// Send HTTP/2 request after successful upgrade
					err = sendRequestHTTP2(conn, targetURL)
					if err != nil {
						log.Println("Failed to send HTTP/2 request:", err)
						return
					}

					log.Printf("Send request done, url: %v\n", targetURL)
					<-doneCh
					return
				} else if state.NegotiatedProtocol == "http/1.1" {
					log.Println("Falling back to HTTP/1.1")
				} else {
					log.Println("Failed to negotiate HTTP/2, ALPN Negotiated Protocol:", state.NegotiatedProtocol)
					return
				}
			} else {
				// Attempt to upgrade to HTTP/2 (h2c)
				err = sendUpgradeRequestHTTP1(conn, targetURL)
				if err != nil {
					log.Println("Failed to send HTTP/1.1 Upgrade request:", err)
					return
				}

				// Read the server's response to the upgrade request
				upgraded, err := readUpgradeResponse(conn)
				if err != nil {
					log.Println("Failed to read upgrade response:", err)
					return
				}

				if upgraded {
					doneCh := make(chan struct{})
					go func() {
						defer close(doneCh)
						readLoop(conn)
					}()

					// Send HTTP/2 request after successful upgrade
					err = sendRequestHTTP2(conn, targetURL)
					if err != nil {
						log.Println("Failed to send HTTP/2 request:", err)
						return
					}

					log.Printf("Send h2c request done, url: %v\n", targetURL)
					<-doneCh
					return
				}
			}

			log.Println("< start sendRequestHTTP1")
			// Send HTTP/1.1 request if no upgrade
			err = sendRequestHTTP1(conn, targetURL)
			if err != nil {
				fmt.Println("Failed to send HTTP/1.1 request:", err)
				return
			}

			// Read HTTP/1.1 response
			readResponseHTTP1(conn)
			log.Println("finish readResponseHTTP1 >")
		},
	}
)

func init() {
	fetchCmd.Flags().StringP("user-agent", "A", "netool-fetch", "proto binary file")
	base.AddSubCommands(fetchCmd)
}

func sendUpgradeRequestHTTP1(conn net.Conn, parsedURL *url.URL) error {
	host := parsedURL.Host
	path := parsedURL.RequestURI()

	// Generate HTTP2-Settings header value with specific SETTINGS frame (base64 encoded)
	settings := []byte{
		0x00, 0x00, 0x0c, // Length (12 bytes)
		0x04,                   // Type: SETTINGS (0x4)
		0x00,                   // Flags
		0x00, 0x00, 0x00, 0x00, // Stream ID: 0 (connection control frame)
		// SETTINGS payload:
		0x00, 0x03, 0x00, 0x00, 0x00, 0x64, // SETTINGS_MAX_CONCURRENT_STREAMS = 100
		0x00, 0x04, 0x00, 0x00, 0x40, 0x00, // SETTINGS_INITIAL_WINDOW_SIZE = 16384
	}
	http2Settings := base64.StdEncoding.EncodeToString(settings)

	log.Println("< Sent HTTP/1.1 Upgrade request")
	// Create HTTP/1.1 Upgrade request
	request := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"User-Agent: curl/8.7.1\r\n"+
			"Accept: */*\r\n"+
			"Connection: Upgrade, HTTP2-Settings\r\n"+
			"Upgrade: h2c\r\n"+
			"HTTP2-Settings: %s\r\n\r\n",
		path, host, http2Settings)

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

func readLoop(conn net.Conn) {
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("SetReadDeadline err: %v\n", err)
	}
	if err := http2.ReadFrames(conn); err != nil {
		log.Printf("read conn err: %v\n", err)
	}
}

func sendSettingsFrame(conn net.Conn) error {
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], 0x0) // Length (0 bytes)
	frameHeader[3] = 0x4                             // Type: SETTINGS (0x4)
	frameHeader[4] = 0x0                             // Flags
	binary.BigEndian.PutUint32(frameHeader[5:], 0x0) // Stream ID: 0 (connection control frame)

	// Send the SETTINGS frame
	_, err := conn.Write(frameHeader)
	if err != nil {
		return err
	}

	log.Println("Sent SETTINGS frame successful")
	return nil
}

func sendRequestHTTP2(conn net.Conn, parsedURL *url.URL) error {
	path := parsedURL.RequestURI()
	host := parsedURL.Host

	headers := http.Header{
		":method":    {"GET"},
		":scheme":    {parsedURL.Scheme},
		":authority": {host},
		":path":      {path},
		":accept":    {"*/*"},
	}

	if err := http2.WriteHeadersFrame(conn, atomic.AddUint32(&streamID, 2), headers); err != nil {
		return err
	}

	log.Println("Sent HTTP/2 request headers")
	return nil
}

func sendRequestHTTP1(conn net.Conn, parsedURL *url.URL) error {
	host := parsedURL.Host
	path := parsedURL.RequestURI()

	// Create HTTP/1.1 request
	request := fmt.Sprintf("GET %s HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Accept: */*\r\n"+
		"Connection: close\r\n"+
		"\r\n", path, host)

	log.Println("Sent HTTP/1.1 request")
	log.Println(request)
	_, err := conn.Write([]byte(request))
	if err != nil {
		return err
	}
	return nil
}

func readResponseHTTP1(conn net.Conn) {
	reader := bufio.NewReader(conn)

	log.Println("Reading response headers:")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Failed to read response:", err)
			return
		}
		log.Print(line)
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	log.Println("Reading response body:")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Print(line)
				break
			}
			log.Println("Failed to read response body:", err)
			return
		}
		log.Print(line)
	}
}

func dialConn(parsedURL *url.URL) (net.Conn, error) {
	addr := getHostAddress(parsedURL)
	if parsedURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // NOTE: For testing only. Do not use in production.
			NextProtos:         []string{"h2", "http/1.1"},
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return nil, err
		}
		return conn, nil
	} else {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return nil, err
		}
		return conn, nil
	}
}

func getHostAddress(parsedURL *url.URL) string {
	host := parsedURL.Host
	if parsedURL.Port() == "" {
		switch parsedURL.Scheme {
		case "https":
			host += ":443"
		case "http":
			host += ":80"
		}
	}
	return host
}
