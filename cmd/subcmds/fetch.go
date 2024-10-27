package subcmds

import (
	"crypto/tls"
	"github.com/pysugar/wheels/binproto/http2"
	"log"
	"net/url"

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

		err = http2.ReadFrames(conn)
		if err != nil {
			log.Printf("read conn err: %v\n", err)
		}
	},
}

func init() {
	base.AddSubCommands(fetchCmd)
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
