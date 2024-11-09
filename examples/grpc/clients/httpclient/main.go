package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	http2tool "github.com/pysugar/wheels/binproto/http2"
	"golang.org/x/net/http2"
	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

type (
	GRPCRequest struct {
		ServerAddr string
		GrpcMethod string
		Payload    []byte
	}

	GRPCResponse struct {
		StatusCode int
		Message    string
		Payload    []byte
	}

	GRPCClient struct {
		client    *http.Client
		scheme    string
		mutex     sync.Mutex
		streamMap map[uint32]*GRPCResponse
	}
)

func main() {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2"},
	}

	client := NewGRPCClient(tlsConfig)
	serverAddr := "127.0.0.1:8443"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := sendHealthCheckRequest(ctx, client, serverAddr); err != nil {
				log.Printf("send health check request failed: %v\n", err)
			}
		}()
	}

	wg.Wait()
}

func sendHealthCheckRequest(ctx context.Context, client *GRPCClient, serverAddr string) error {
	arg := &grpchealthv1.HealthCheckRequest{}
	reqArg, err := proto.Marshal(arg)
	if err != nil {
		return err
	}

	payload := http2tool.EncodeGrpcPayload(reqArg)
	log.Printf("request payload: %v\n", payload)
	req := GRPCRequest{
		ServerAddr: serverAddr,
		GrpcMethod: "grpc.health.v1.Health/Check",
		Payload:    payload,
	}

	resp, err := client.CallGRPCService(ctx, req)
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

func (c *GRPCClient) CallGRPCService(ctx context.Context, req GRPCRequest) (*GRPCResponse, error) {
	requestUrl := fmt.Sprintf("%s://%s/%s", c.scheme, req.ServerAddr, req.GrpcMethod)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", requestUrl, bytes.NewReader(req.Payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/grpc+proto")
	httpReq.Header.Set("TE", "trailers")

	resp, err := c.client.Do(httpReq)
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

func NewGRPCClient(tlsConfig *tls.Config) *GRPCClient {
	transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
	}

	return &GRPCClient{
		client:    client,
		scheme:    "https",
		streamMap: make(map[uint32]*GRPCResponse),
	}
}
