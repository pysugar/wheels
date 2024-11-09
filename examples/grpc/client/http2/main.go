package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"golang.org/x/net/http2"
	grpchealthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

type (
	GRPCRequest struct {
		Method  string
		Payload []byte
	}

	GRPCResponse struct {
		StatusCode int
		Message    string
		Payload    []byte
	}

	GRPCClient struct {
		client    *http.Client
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

	arg := &grpchealthv1.HealthCheckRequest{}
	payload, err := proto.Marshal(arg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("payload: %v", payload)

	req := GRPCRequest{
		Method:  "https://127.0.0.1:8443/grpc.health.v1.Health/Check",
		Payload: payload,
	}

	resp, err := client.CallGRPCService(req)
	if err != nil {
		log.Fatalf("gRPC invoke failure: %v", err)
	}

	if resp.StatusCode != 0 {
		log.Fatalf("gRPC error: %s", resp.Message)
	}

	fmt.Printf("gRPC resp: %s\n", string(resp.Payload))
}

func (c *GRPCClient) CallGRPCService(req GRPCRequest) (*GRPCResponse, error) {
	httpReq, err := http.NewRequest("POST", req.Method, bytes.NewReader(req.Payload))
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	grpcStatus := resp.Header.Get("grpc-status")
	grpcMessage := resp.Header.Get("grpc-message")

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
		streamMap: make(map[uint32]*GRPCResponse),
	}
}
