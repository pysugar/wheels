package subcmds

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"github.com/pysugar/wheels/binproto/http2"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/pysugar/wheels/cmd/base"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
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

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // NOTE: For testing only. Do not use in production.
			NextProtos:         []string{"h2"},
		}

		addr := getHostAddress(targetURL)
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return
		}
		defer conn.Close()

		state := conn.ConnectionState()
		fmt.Printf("TLS Handshake state: %+v", state)
		if state.NegotiatedProtocol != "h2" {
			fmt.Println("Failed to negotiate HTTP/2, ALPN Negotiated Protocol:", state.NegotiatedProtocol)
			return
		}
		log.Println("Successfully negotiated HTTP/2")

		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			reader(conn)
		}()

		err = sendRequest(conn, targetURL)
		if err != nil {
			fmt.Println("Failed to send request:", err)
			return
		}

		log.Printf("Send request done, url: %v\n", targetURL)
		<-doneCh
	},
}

func init() {
	base.AddSubCommands(fetchCmd)
}

func reader(conn net.Conn) {
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("SetReadDeadline err: %v\n", err)
	}
	if err := http2.ReadFrames(conn); err != nil {
		log.Printf("read conn err: %v\n", err)
	}
}

func sendRequest(conn *tls.Conn, parsedURL *url.URL) error {
	path := parsedURL.RequestURI()
	host := parsedURL.Host

	// Create HTTP/2 request headers
	headers := []struct {
		Name  string
		Value string
	}{
		{":method", "GET"},
		{":scheme", parsedURL.Scheme},
		{":path", path},
		{":authority", host},
		{"user-agent", "netool-fetch"},
		{"content-type", "application/json"},
	}

	var headersBuffer bytes.Buffer
	for _, header := range headers {
		// Write each header field as: length of name, name, length of value, value
		headersBuffer.WriteByte(byte(len(header.Name)))
		headersBuffer.WriteString(header.Name)
		headersBuffer.WriteByte(byte(len(header.Value)))
		headersBuffer.WriteString(header.Value)
	}

	headersPayload := headersBuffer.Bytes()
	length := len(headersPayload)
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], uint32(length))
	frameHeader[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	frameHeader[1] = byte((length >> 8) & 0xFF)
	frameHeader[2] = byte(length & 0xFF)
	frameHeader[3] = 0x1                             // Type: HEADERS (0x1)
	frameHeader[4] = 0x4                             // Flags: END_HEADERS (0x4)
	binary.BigEndian.PutUint32(frameHeader[5:], 0x1) // Stream ID: 1

	// Send the HEADERS frame
	_, err := conn.Write(frameHeader)
	if err != nil {
		log.Printf("Send the HEADERS frame failure: %v\n", err)
		return err
	}
	_, err = conn.Write(headersPayload)
	if err != nil {
		log.Printf("Send the HEADERS payload failure: %v\n", err)
		return err
	}

	log.Println("Sent HTTP/2 request headers successful")
	return nil
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
