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
	activeStream struct {
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
		activeStreams sync.Map // streamID -> activeStream
		streamIdGen   uint32
		ctx           context.Context
		cancel        context.CancelFunc
		encoder       *hpack.Encoder
		decoder       *hpack.Decoder
		encoderBuf    bytes.Buffer // encoder buffer
		encodeMu      sync.Mutex   // protect encoder and encoderBuf
		decodeMu      sync.Mutex
		writeMu       sync.Mutex
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
	client := &grpcClient{
		serverURL: serverURL,
		framer:    framer,
		ctx:       ctx,
		cancel:    cancel,
	}
	client.encoder = hpack.NewEncoder(&client.encoderBuf)
	client.decoder = hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		log.Printf("emit: %+v", f)
	})

	//if er := framer.WriteSettings(); er != nil {
	//	client.Close()
	//	log.Printf("write settings failed: %v", er)
	//	return nil, er
	//}

	go client.readLoop(ctx)
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

	active := &activeStream{
		streamId:   streamId,
		activeAt:   time.Now(),
		doneCh:     make(chan struct{}),
		grpcStatus: -1,
	}
	c.activeStreams.Store(streamId, active)
	defer func() {
		c.activeStreams.Delete(streamId)
	}()

	c.writeMu.Lock()
	if er := c.framer.WriteData(streamId, true, http2tool.EncodeGrpcPayload(reqBytes)); er != nil {
		c.writeMu.Unlock()
		return er
	}
	c.writeMu.Unlock()

	select {
	case <-active.doneCh:
		if active.grpcStatus != 0 {
			return fmt.Errorf("[%d] grpc error: %s", active.grpcStatus, active.grpcMessage)
		}
		if er := http2tool.DecodeGrpcFrameWithDecompress(active.payload, active.compressionAlgo, res); er != nil {
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
	if v, loaded := c.activeStreams.Load(f.StreamID); loaded {
		if active, ok := v.(*activeStream); ok {
			active.mu.Lock()
			active.payload = append(active.payload, f.Data()...)
			active.mu.Unlock()

			if f.StreamEnded() {
				c.activeStreams.Delete(f.StreamID)
				close(active.doneCh)
			}
		}
	}
	return nil
}

func (c *grpcClient) processHeaderFrame(f *http2.HeadersFrame) error {
	if v, loaded := c.activeStreams.Load(f.StreamID); loaded {
		if active, ok := v.(*activeStream); ok {
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
						active.grpcStatus = statusCode
					} else {
						log.Printf("[ERROR] failed to parse grpc status: %v", hf.Value)
					}
				} else if hf.Name == "grpc-message" {
					active.grpcMessage = hf.Value
				} else if hf.Name == "grpc-encoding" {
					active.compressionAlgo = hf.Value
				}
			}
			if f.StreamEnded() {
				c.activeStreams.Delete(f.StreamID)
				close(active.doneCh)
			}
		}
	}
	return nil
}

func (c *grpcClient) processRSTStreamFrame(f *http2.RSTStreamFrame) error {
	if v, loaded := c.activeStreams.Load(f.StreamID); loaded {
		if active, ok := v.(*activeStream); ok {
			active.grpcStatus = int(f.ErrCode)
			active.grpcMessage = "Stream reset by server"
			c.activeStreams.Delete(f.StreamID)
			close(active.doneCh)
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
	return c.framer.WriteSettingsAck()
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
	//if active, ok := c.activeStreams.Load(f.LastStreamID); ok {
	//	close(active.(*activeStream).doneCh)
	//}
	c.activeStreams.Range(func(key, value any) bool {
		if active, ok := value.(*activeStream); ok {
			active.grpcStatus = int(f.ErrCode)
			active.grpcMessage = "stream go away"
			c.activeStreams.Delete(f.StreamID)
			close(active.doneCh)
		}
		return true
	})
	return nil //io.EOF
}

func grpcHeaders(serverURL *url.URL, fullMethod string) []hpack.HeaderField {
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

	//c.encoderBuf.Reset()
	//for _, header := range headers {
	//	if err := c.encoder.WriteField(header); err != nil {
	//		log.Printf("failed to encode header field: %v", err)
	//		continue
	//	}
	//	log.Printf("Send Header %s: %s\n", header.Name, header.Value)
	//}
	//return c.encoderBuf.Bytes()

	before := c.encoderBuf.Len()
	for _, header := range headers {
		if err := c.encoder.WriteField(header); err != nil {
			log.Printf("failed to encode header field: %v", err)
			continue
		}
		log.Printf("Send Header %s: %s\n", header.Name, header.Value)
	}
	after := c.encoderBuf.Len()
	return c.encoderBuf.Bytes()[before:after]
}

func dialConn(serverURL *url.URL) (net.Conn, error) {
	addr := getHostAddress(serverURL)
	if serverURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // NOTE: For testing only. Do not use in production.
			NextProtos:         []string{"h2"},
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return nil, err
		}

		//if er := conn.Handshake(); er != nil {
		//	conn.Close()
		//	return nil, fmt.Errorf("TLS handshake failed: %w", er)
		//}

		//if np := conn.ConnectionState().NegotiatedProtocol; np != "h2" {
		//	conn.Close()
		//	return nil, fmt.Errorf("failed to negotiate HTTP/2 via ALPN, got %s", np)
		//}

		//log.Printf("Send HTTP/2 Client Preface: %s\n", clientPreface)
		//if _, er := conn.Write(clientPreface); er != nil {
		//	conn.Close()
		//	return nil, er
		//}

		return conn, nil
	} else {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return nil, err
		}

		log.Printf("Send HTTP/2 Client Preface: %s\n", clientPreface)
		if _, er := conn.Write(clientPreface); er != nil {
			conn.Close()
			return nil, er
		}
		return conn, nil
	}
	//
	//conn, err := net.Dial("tcp", serverURL.Host)
	//if err != nil {
	//	log.Printf("Failed to connect: %v\n", err)
	//	return nil, err
	//}
	//
	//if serverURL.Scheme == "https" {
	//	tlsConfig := &tls.Config{
	//		InsecureSkipVerify: true,
	//		NextProtos:         []string{"h2"},
	//	}
	//
	//	tlsConn := tls.Client(conn, tlsConfig)
	//	if er := tlsConn.Handshake(); er != nil {
	//		conn.Close()
	//		return nil, fmt.Errorf("TLS handshake failed: %w", er)
	//	}
	//
	//	// 检查 ALPN 协商结果
	//	if np := tlsConn.ConnectionState().NegotiatedProtocol; np != "h2" {
	//		tlsConn.Close()
	//		return nil, fmt.Errorf("failed to negotiate HTTP/2 via ALPN, got %s", np)
	//	}
	//	conn = tlsConn
	//} else {
	//	log.Printf("Send HTTP/2 Client Preface: %s\n", clientPreface)
	//	if _, er := conn.Write(clientPreface); er != nil {
	//		conn.Close()
	//		return nil, er
	//	}
	//}
	//return conn, nil
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
