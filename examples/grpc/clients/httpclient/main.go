package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	http2tool "github.com/pysugar/wheels/binproto/http2"
	"github.com/pysugar/wheels/protocol/http/extensions"
	"golang.org/x/net/http2"
	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

type (
	GRPCRequest struct {
		Scheme     string
		ServerAddr string
		GrpcMethod string
		Payload    []byte
	}

	GRPCResponse struct {
		StatusCode int
		Message    string
		Payload    []byte
	}
)

var (
	traceIdGen uint32
)

func main() {
	serverURL, _ := url.Parse("http://127.0.0.1:8080")
	// serverURL, _ := url.Parse("https://127.0.0.1:8443")
	// serverURL, _ := url.Parse("http://127.0.0.1:50051")
	var transport http.RoundTripper
	if serverURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2"},
		}
		transport = &http2.Transport{
			TLSClientConfig: tlsConfig,
		}
	} else {
		transport = &http2.Transport{
			AllowHTTP: true, // allow h2c
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		}
	}

	var client = &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := sendHealthCheckRequest(ctx, client, serverURL); err != nil {
				log.Printf("send health check request failed: %v\n", err)
			}
		}()
	}

	wg.Wait()
}

func sendHealthCheckRequest(ctx context.Context, client *http.Client, serverURL *url.URL) error {
	arg := &grpchealthv1.HealthCheckRequest{}
	reqArg, err := proto.Marshal(arg)
	if err != nil {
		return err
	}

	payload := http2tool.EncodeGrpcPayload(reqArg)
	log.Printf("request payload: %v\n", payload)
	req := &GRPCRequest{
		Scheme:     serverURL.Scheme,
		ServerAddr: serverURL.Host,
		GrpcMethod: "grpc.health.v1.Health/Check",
		Payload:    payload,
	}

	ctx = httptrace.WithClientTrace(ctx, extensions.NewDebugClientTrace(fmt.Sprintf("req-%03d", atomic.AddUint32(&traceIdGen, 1))))
	resp, err := callGRPCService(ctx, client, req)
	if err != nil {
		log.Printf("gRPC invoke failure: %v\n", err)
		return err
	}

	if resp.StatusCode != 0 {
		log.Printf("gRPC error: %s\n", resp.Message)
		return fmt.Errorf("gRPC error: %s", resp.Message)
	}

	log.Printf("response payload: %v\n", resp.Payload)
	res := &grpchealthv1.HealthCheckResponse{}
	if er := http2tool.DecodeGrpcFrame(resp.Payload, res); er != nil {
		log.Printf("response payload decode error: %v\n", er)
		return er
	}
	log.Printf("received health response: %+v\n", res)
	return nil
}

func callGRPCService(ctx context.Context, client *http.Client, grpcReq *GRPCRequest) (*GRPCResponse, error) {
	requestUrl := fmt.Sprintf("%s://%s/%s", grpcReq.Scheme, grpcReq.ServerAddr, grpcReq.GrpcMethod)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", requestUrl, bytes.NewReader(grpcReq.Payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/grpc+proto")
	httpReq.Header.Set("TE", "trailers")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//log.Printf("Req: %s\n", extensions.FormatRequest(httpReq))
	//log.Printf("Res: %s\n", extensions.FormatResponse(resp))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	grpcStatus := resp.Trailer.Get("grpc-status")
	grpcMessage := resp.Trailer.Get("grpc-message")

	statusCode, err := strconv.Atoi(grpcStatus)
	if err != nil {
		return nil, fmt.Errorf("invalid grpc-status: %s", grpcStatus)
	}

	return &GRPCResponse{
		StatusCode: statusCode,
		Message:    grpcMessage,
		Payload:    body,
	}, nil
}
