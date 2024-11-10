package http2client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	http2tool "github.com/pysugar/wheels/binproto/http2"
	"golang.org/x/net/http2"
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
		framer        *http2.Framer
		clientStreams sync.Map // streamID -> activeStream
		streamIdGen   uint32
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

	framer := http2.NewFramer(conn, conn)
	ctx, cancel := context.WithCancel(context.Background())

	settingsAcked := make(chan struct{})
	client := &grpcClient{
		serverURL:     serverURL,
		framer:        framer,
		conn:          conn,
		ctx:           ctx,
		cancel:        cancel,
		settingsAcked: settingsAcked,
	}
	client.encoder = hpack.NewEncoder(&client.encoderBuf)
	client.decoder = hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		log.Printf("emit: %+v", f)
	})

	if er := framer.WriteSettings(http2.Setting{
		ID:  http2.SettingMaxConcurrentStreams,
		Val: 1024,
	}); er != nil {
		client.Close()
		log.Printf("write settings failed: %v", er)
		return nil, er
	}
	log.Printf("Client Write Settings")
	go client.readLoop(ctx)

	select {
	case _, ok := <-settingsAcked:
		log.Printf("acked server settings done, ack channel available: %v", ok)
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
	if er := c.framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamId,
		BlockFragment: c.encodeHpackHeaders(headers),
		EndHeaders:    true,
	}); er != nil {
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
	if er := c.framer.WriteData(streamId, true, http2tool.EncodeGrpcPayload(reqBytes)); er != nil {
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

func (c *grpcClient) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := c.readFrame(ctx); err != nil {
				if err == io.EOF {
					log.Printf("Connection closed by remote host")
					return
				}
				log.Printf("Failed to read frame: %v", err)
			}
		}
	}
}

func (c *grpcClient) readFrame(ctx context.Context) error {
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	frame, err := c.framer.ReadFrame()
	if err != nil {
		log.Printf("Failed to read frame: %v", err)
		return err
	}

	log.Printf("[stream-%03d] Received %s frame: %+v", frame.Header().StreamID, frame.Header().Type, frame)

	switch f := frame.(type) {
	case *http2.DataFrame: // 0
		return c.processDataFrame(f)
	case *http2.HeadersFrame: // 1
		return c.processHeaderFrame(f)
	case *http2.PriorityFrame: // 2
	case *http2.RSTStreamFrame: // 3
		return c.processRSTStreamFrame(f)
	case *http2.SettingsFrame: // 4
		return c.processSettingsFrame(f)
	case *http2.PushPromiseFrame: // 5
	case *http2.PingFrame: // 6
		return c.processPingFrame(f)
	case *http2.GoAwayFrame: // 7
		return c.processGoAwayFrame(f)
	case *http2.WindowUpdateFrame: //8
	case *http2.ContinuationFrame: //9
	default:
		log.Printf("Received unknown frame: %T", f)
	}
	return nil
}

func (c *grpcClient) processDataFrame(f *http2.DataFrame) error {
	if v, loaded := c.clientStreams.Load(f.StreamID); loaded {
		if cs, ok := v.(*clientStream); ok {
			cs.mu.Lock()
			cs.payload = append(cs.payload, f.Data()...)
			cs.mu.Unlock()

			if f.StreamEnded() {
				c.clientStreams.Delete(f.StreamID)
				close(cs.doneCh)
			}
		}
	}
	return nil
}

func (c *grpcClient) processHeaderFrame(f *http2.HeadersFrame) error {
	if v, loaded := c.clientStreams.Load(f.StreamID); loaded {
		if cs, ok := v.(*clientStream); ok {
			c.decodeMu.Lock()
			headers, err := c.decoder.DecodeFull(f.HeaderBlockFragment())
			c.decodeMu.Unlock()

			if err != nil {
				return fmt.Errorf("[stream-%03d] Failed to decode headers: %w", f.StreamID, err)
			}

			for _, hf := range headers {
				log.Printf("received header (%s: %s)", hf.Name, hf.Value)
				if hf.Name == "grpc-status" {
					if statusCode, er := strconv.Atoi(hf.Value); er == nil {
						cs.grpcStatus = statusCode
					} else {
						log.Printf("[ERROR] failed to parse grpc status: %v", hf.Value)
					}
				} else if hf.Name == "grpc-message" {
					cs.grpcMessage = hf.Value
				} else if hf.Name == "grpc-encoding" {
					cs.compressionAlgo = hf.Value
				}
			}
			if f.StreamEnded() {
				c.clientStreams.Delete(f.StreamID)
				close(cs.doneCh)
			}
		}
	}
	return nil
}

func (c *grpcClient) processRSTStreamFrame(f *http2.RSTStreamFrame) error {
	if v, loaded := c.clientStreams.Load(f.StreamID); loaded {
		if cs, ok := v.(*clientStream); ok {
			cs.grpcStatus = int(f.ErrCode)
			cs.grpcMessage = "Stream reset by server"
			c.clientStreams.Delete(f.StreamID)
			close(cs.doneCh)
		}
	}
	return nil
}

func (c *grpcClient) processSettingsFrame(f *http2.SettingsFrame) error {
	log.Printf("Server Settings [%d]: ", f.NumSettings())
	for i := 0; i < f.NumSettings(); i++ {
		settings := f.Setting(i)
		log.Printf("\t%+v: %d\n", settings.ID, settings.Val)
	}

	if f.IsAck() {
		return nil
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	err := c.framer.WriteSettingsAck()
	if err != nil {
		return err
	}
	close(c.settingsAcked)
	return nil
}

func (c *grpcClient) processPingFrame(f *http2.PingFrame) error {
	if f.IsAck() {
		return nil
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.framer.WritePing(true, f.Data)
}

func (c *grpcClient) processGoAwayFrame(f *http2.GoAwayFrame) error {
	log.Printf("Received GOAWAY frame: LastStreamID=%d, SteamID=%d, ErrorCode=%d, DebugData=%s", f.LastStreamID,
		f.StreamID, f.ErrCode, f.DebugData())
	c.clientStreams.Range(func(key, value any) bool {
		if cs, ok := value.(*clientStream); ok {
			cs.grpcStatus = int(f.ErrCode)
			cs.grpcMessage = "stream go away"
			c.clientStreams.Delete(cs.streamId)
			close(cs.doneCh)
		}
		return true
	})
	return nil //io.EOF
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
	//before := c.encoderBuf.Len()
	//for _, header := range headers {
	//	if err := c.encoder.WriteField(header); err != nil {
	//		log.Printf("failed to encode header field: %v", err)
	//		continue
	//	}
	//	log.Printf("Send Header %s: %s\n", header.Name, header.Value)
	//}
	//after := c.encoderBuf.Len()
	//return c.encoderBuf.Bytes()[before:after]
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
