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
		writeMu       sync.Mutex
	}

	GRPCClient interface {
		Call(ctx context.Context, serviceMethod string, req, res proto.Message) error
		Close()
	}
)

func NewGRPCClient(serverURL *url.URL) (GRPCClient, error) {
	conn, err := openConn(serverURL)
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

	go client.readLoop(ctx)
	if er := framer.WriteSettings(); er != nil {
		client.Close()
		log.Printf("write settings failed: %v", er)
		return nil, er
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

	streamId := atomic.AddUint32(&c.streamIdGen, 2) - 1
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

	headers := grpcHeaders(c.serverURL, serviceMethod)
	c.writeMu.Lock()
	if er := c.framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamId,
		BlockFragment: c.encodeHpackHeaders(headers),
		EndHeaders:    true,
	}); er != nil {
		c.writeMu.Unlock()
		return er
	}

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
		return err
	}

	log.Printf("[stream-%03d] Received %s frame: %+v", frame.Header().StreamID, frame.Header().Type, frame)

	switch f := frame.(type) {
	case *http2.DataFrame: // 0
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
	case *http2.HeadersFrame: // 1
		if v, loaded := c.activeStreams.Load(f.StreamID); loaded {
			if active, ok := v.(*activeStream); ok {
				headers, er := c.decoder.DecodeFull(f.HeaderBlockFragment())
				if er != nil {
					return er
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
	case *http2.PriorityFrame: // 2
	case *http2.RSTStreamFrame: // 3
	case *http2.SettingsFrame: // 4
		if f.IsAck() {
			return nil
		}
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		return c.framer.WriteSettingsAck()
	case *http2.PushPromiseFrame: // 5
	case *http2.PingFrame: // 6
		if f.IsAck() {
			return nil
		}
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		return c.framer.WritePing(true, f.Data)
	case *http2.GoAwayFrame: // 7
		log.Printf("Received GOAWAY frame: LastStreamID=%d, ErrorCode=%d, DebugData=%s", f.LastStreamID,
			f.ErrCode, f.DebugData())
		if active, ok := c.activeStreams.Load(f.StreamID); ok {
			close(active.(*activeStream).doneCh)
		}
		return io.EOF
	case *http2.WindowUpdateFrame: //8
	case *http2.ContinuationFrame: //9
	default:
		log.Printf("Received unknown frame: %T", f)
	}
	return nil
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

//func encodeHpackHeaders(headers []hpack.HeaderField) []byte {
//	var headersBuffer bytes.Buffer
//	encoder := hpack.NewEncoder(&headersBuffer)
//	for _, header := range headers {
//		err := encoder.WriteField(hpack.HeaderField{
//			Name:  header.Name,
//			Value: header.Value,
//		})
//		if err != nil {
//			log.Printf("failed to encode header field: %v", err)
//			continue
//		}
//		log.Printf("Send Header %s: %s\n", header.Name, header.Value)
//	}
//	return headersBuffer.Bytes()
//}

//func decodeHpackHeaders(headerPayload []byte) []hpack.HeaderField {
//	headers := make([]hpack.HeaderField, 0)
//
//	buf := bytes.NewBuffer(headerPayload)
//	decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
//		headers = append(headers, f)
//	})
//	for buf.Len() > 0 {
//		if n, err := decoder.Write(buf.Next(buf.Len())); err != nil {
//			if err == io.EOF {
//				break
//			}
//			log.Printf("failed to decode header field: %v, n = %d\n", err, n)
//			continue
//		}
//	}
//	return headers
//}

//func newFramer(serverURL *url.URL) (*http2.Framer, error) {
//	conn, err := net.Dial("tcp", serverURL.Host)
//	if err != nil {
//		log.Printf("Failed to connect: %v\n", err)
//		return nil, err
//	}
//
//	if serverURL.Scheme == "https" {
//		tlsConfig := &tls.Config{
//			InsecureSkipVerify: true,
//			NextProtos:         []string{"h2"},
//		}
//		conn = tls.Client(conn, tlsConfig)
//	} else {
//		clientPreface := []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
//		log.Printf("Send HTTP/2 Client Preface: %s\n", clientPreface)
//		if _, er := conn.Write(clientPreface); er != nil {
//			return nil, er
//		}
//	}
//	return http2.NewFramer(conn, conn), nil
//}

func openConn(serverURL *url.URL) (net.Conn, error) {
	conn, err := net.Dial("tcp", serverURL.Host)
	if err != nil {
		log.Printf("Failed to connect: %v\n", err)
		return nil, err
	}

	if serverURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2"},
		}
		conn = tls.Client(conn, tlsConfig)
	} else {
		clientPreface := []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
		log.Printf("Send HTTP/2 Client Preface: %s\n", clientPreface)
		if _, er := conn.Write(clientPreface); er != nil {
			return nil, er
		}
	}
	return conn, nil
}
