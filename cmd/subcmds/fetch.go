package subcmds

import (
	"crypto/tls"
	"fmt"
	"github.com/pysugar/wheels/binproto/http2"
	"log"
	"net/url"
	"strings"

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
		if state.NegotiatedProtocol != "h2" {
			log.Println("Failed to negotiate HTTP/2")
			return
		}
		log.Println("Successfully negotiated HTTP/2")

		err = sendRequest(conn, targetURL)
		if err != nil {
			fmt.Println("Failed to send request:", err)
			return
		}

		err = http2.ReadFrames(conn)
		if err != nil {
			log.Printf("read conn err: %v\n", err)
		}
	},
}

func init() {
	base.AddSubCommands(fetchCmd)
}

func sendRequest(conn *tls.Conn, parsedURL *url.URL) error {
	path := parsedURL.RequestURI()
	host := parsedURL.Host

	// Create HTTP/2 request headers
	headers := []string{
		fmt.Sprintf(":method: %s", "GET"),
		fmt.Sprintf(":scheme: %s", parsedURL.Scheme),
		fmt.Sprintf(":path: %s", path),
		fmt.Sprintf(":authority: %s", host),
		"user-agent: golang-http2-client",
	}

	// Encode headers as a simple format (not full HPACK encoding)
	var headersBuffer strings.Builder
	for _, header := range headers {
		headersBuffer.WriteString(header)
		headersBuffer.WriteString("\r\n")
	}
	headersBuffer.WriteString("\r\n")

	// Send headers
	if _, err := conn.Write([]byte(headersBuffer.String())); err != nil {
		return err
	}
	fmt.Println("Sent HTTP/2 request headers")
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
