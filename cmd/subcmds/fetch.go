package subcmds

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/pysugar/wheels/cmd/base"
	"github.com/pysugar/wheels/http/client"
	"github.com/spf13/cobra"
	pb "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	fetchCmd = &cobra.Command{
		Use:   `fetch https://www.google.com`,
		Short: "fetch http2 response from url",
		Long: `
fetch http2 response from url

fetch http2 response from url: netool fetch https://www.google.com
call grpc service: netool fetch --grpc https://localhost:8443/grpc.health.v1.Health/Check
call grpc via context path: netool fetch --grpc http://localhost:8080/grpc/grpc.health.v1.Health/Check
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				log.Printf("you must specify the url")
				return
			}

			isGRPC, _ := cmd.Flags().GetBool("grpc")
			isHTTP2, _ := cmd.Flags().GetBool("http2")
			method, _ := cmd.Flags().GetString("method")

			targetURL, err := url.Parse(args[0])
			if err != nil {
				log.Printf("invalid url %s\n", args[0])
				return
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			isVerbose, _ := cmd.Flags().GetBool("verbose")
			if isVerbose {
				ctx = client.WithVerbose(ctx)
			}

			fetcher := client.NewFetcher()
			if isGRPC {
				req := &pb.HealthCheckRequest{}
				res := &pb.HealthCheckResponse{}
				if er := fetcher.CallGRPC(ctx, targetURL, req, res); er != nil {
					log.Printf("Call grpc %s error: %v\n", targetURL, err)
					return
				}
				fmt.Printf("grpc health check: %+v\n", res)
				return
			}

			if isHTTP2 {
				ctx = client.WithProtocol(ctx, client.HTTP2)
			}

			data, _ := cmd.Flags().GetString("data")
			var body io.Reader
			if data != "" {
				body = strings.NewReader(data)
			}

			req, err := http.NewRequestWithContext(ctx, method, targetURL.String(), body)
			res, er := fetcher.Do(ctx, req)
			if er != nil {
				log.Printf("Call %v %s error: %v\n", client.ProtocolFromContext(ctx), targetURL, err)
				return
			}
			fmt.Printf("http resp: %+v\n", res)
		},
	}
)

func init() {
	fetchCmd.Flags().StringP("user-agent", "A", "", "User Agent")
	fetchCmd.Flags().StringP("method", "M", "GET", "HTTP Method")
	grpcCmd.Flags().StringP("data", "d", "", "request data")
	fetchCmd.Flags().BoolP("grpc", "G", false, "Is GRPC Request Or Not")
	fetchCmd.Flags().BoolP("http2", "H", false, "Is HTTP2 Request Or Not")
	fetchCmd.Flags().BoolP("verbose", "V", false, "Verbose mode")
	base.AddSubCommands(fetchCmd)
}
