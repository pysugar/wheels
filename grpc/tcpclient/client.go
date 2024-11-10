package tcpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	http2tool "github.com/pysugar/wheels/binproto/http2"
	"golang.org/x/net/http2/hpack"
	"google.golang.org/protobuf/proto"
)

const (
	ClientPreface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

type (
	clientStream struct {
		streamId        uint32
		activeAt        time.Time
		doneCh          chan struct{}
		grpcStatus      int
		grpcMessage     string
		compressionAlgo string
		payload         []byte
		mu              sync.Mutex
	}

	grpcClient struct {
		serverURL     *url.URL
		conn          net.Conn
		streamIdGen   uint32
		clientStreams sync.Map // streamID -> clientStream
		frameHandlers map[uint8]frameHandler
		ctx           context.Context
		cancel        context.CancelFunc
		encoder       *hpack.Encoder
		decoder       *hpack.Decoder
		encoderBuf    bytes.Buffer // encoder buffer
		encodeMu      sync.Mutex   // protect encoder and encoderBuf
		decodeMu      sync.Mutex
		writeMu       sync.Mutex
		settingsAcked chan struct{}
	}

	GRPCClient interface {
		Call(ctx context.Context, serviceMethod string, req, res proto.Message) error
		Close()
	}
)

var (
	clientPreface = []byte(ClientPreface)
)

func NewGRPCClient(serverURL *url.URL) (GRPCClient, error) {
	conn, err := dialConn(serverURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	settingsAcked := make(chan struct{})
	client := &grpcClient{
		serverURL:     serverURL,
		ctx:           ctx,
		conn:          conn,
		cancel:        cancel,
		settingsAcked: settingsAcked,
	}

	client.encoder = hpack.NewEncoder(&client.encoderBuf)
	client.decoder = hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		log.Printf("emit: %+v", f)
	})

	// switch frameHeader.Type {
	// case 0: // Data
	// case 1: // Headers
	// case 2: // Priority
	// case 3: // RSTStream
	// case 4: // Settings
	// case 5: // PushPromise
	// case 6: // Ping
	// case 7: // GoAway
	// case 8: // WindowUpdate
	// case 9: // Continuation
	// default:
	// }
	client.frameHandlers = map[uint8]frameHandler{
		0: client.handleDataFrame,
		1: client.handleHeadersFrame,
		3: client.handleRSTStreamFrame,
		4: client.handleSettingsFrame,
		6: client.handlePingFrame,
		7: client.handleGoAwayFrame,
	}

	if er := WriteSettingsFrame(conn, 0, nil); er != nil {
		client.Close()
		log.Printf("write settings failed: %v", er)
		return nil, er
	}
	log.Printf("Client Settings Write")
	go client.readLoop(ctx)

	select {
	case _, ok := <-settingsAcked:
		log.Printf("Server Settings Acked done, ack channel available: %v", ok)
	case <-time.After(time.Second * 5):
		return nil, fmt.Errorf("timeout waiting for settings ack")
	}
	return client, nil
}

func (c *grpcClient) Close() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *grpcClient) Call(ctx context.Context, serviceMethod string, req, res proto.Message) error {
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	headers := grpcHeaders(c.serverURL, serviceMethod)

	c.writeMu.Lock()
	streamId := atomic.AddUint32(&c.streamIdGen, 2) - 1
	// log.Printf("Generated stream ID: %d\n", streamId)
	if er := c.writeHeadersFrame(streamId, headers); er != nil {
		c.writeMu.Unlock()
		return er
	}
	c.writeMu.Unlock()

	cs := &clientStream{
		streamId:   streamId,
		activeAt:   time.Now(),
		doneCh:     make(chan struct{}),
		grpcStatus: -1,
	}
	c.clientStreams.Store(streamId, cs)
	defer func() {
		c.clientStreams.Delete(streamId)
	}()

	c.writeMu.Lock()
	if er := WriteDataFrame(c.conn, streamId, FlagDataEndStream, http2tool.EncodeGrpcPayload(reqBytes)); er != nil {
		c.writeMu.Unlock()
		return er
	}
	c.writeMu.Unlock()

	select {
	case <-cs.doneCh:
		if cs.grpcStatus != 0 {
			return fmt.Errorf("[%d] grpc error: %s", cs.grpcStatus, cs.grpcMessage)
		}
		if er := http2tool.DecodeGrpcFrameWithDecompress(cs.payload, cs.compressionAlgo, res); er != nil {
			return fmt.Errorf("failed to decode response: %w", er)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func grpcHeaders(serverURL *url.URL, fullMethod string) []hpack.HeaderField {
	if !strings.HasPrefix(fullMethod, "/") {
		fullMethod = "/" + fullMethod
	}
	return []hpack.HeaderField{
		{Name: ":method", Value: "POST"},
		{Name: ":scheme", Value: serverURL.Scheme},
		{Name: ":authority", Value: serverURL.Host},
		{Name: ":path", Value: fullMethod},
		{Name: "content-type", Value: "application/grpc"},
		{Name: "te", Value: "trailers"},
		{Name: "grpc-encoding", Value: "identity"},
		{Name: "grpc-accept-encoding", Value: "identity"},
	}
}

func (c *grpcClient) encodeHpackHeaders(headers []hpack.HeaderField) []byte {
	c.encodeMu.Lock()
	defer c.encodeMu.Unlock()

	c.encoderBuf.Reset()
	for _, header := range headers {
		if err := c.encoder.WriteField(header); err != nil {
			log.Printf("failed to encode header field: %v", err)
			continue
		}
		log.Printf("Send Header %s: %s\n", header.Name, header.Value)
	}
	return c.encoderBuf.Bytes()
}

func dialConn(serverURL *url.URL) (conn net.Conn, err error) {
	addr := getHostAddress(serverURL)
	if serverURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // NOTE: For testing only. Do not use in production.
			NextProtos:         []string{"h2"},
		}
		conn, err = tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return
		}
	} else {
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return
		}
	}

	log.Printf("[%T] Send HTTP/2 Client Preface: %s\n", conn, clientPreface)
	if _, er := conn.Write(clientPreface); er != nil {
		conn.Close()
		err = er
	}
	return
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
