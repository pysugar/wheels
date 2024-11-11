package client

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

	"github.com/pysugar/wheels/concurrent"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const (
	ClientPreface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

var (
	clientPreface = []byte(ClientPreface)
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

	clientConn struct {
		serverURL     *url.URL
		conn          net.Conn
		framer        *http2.Framer
		serializer    *concurrent.CallbackSerializer
		cancel        context.CancelFunc
		streamIdGen   uint32
		clientStreams sync.Map // streamID -> activeStream
		encoder       *hpack.Encoder
		decoder       *hpack.Decoder
		encoderBuf    bytes.Buffer // encoder buffer
		closed        bool
	}
)

func newClientConn(serverURL *url.URL) (*clientConn, error) {
	conn, err := dialConn(serverURL)
	if err != nil {
		return nil, err
	}

	log.Printf("[%T] Send HTTP/2 Client Preface: %s\n", conn, clientPreface)
	if _, er := conn.Write(clientPreface); er != nil {
		conn.Close()
		err = er
	}

	framer := http2.NewFramer(conn, conn)
	ctx, cancel := context.WithCancel(context.Background())
	cc := &clientConn{
		serverURL:  serverURL,
		conn:       conn,
		framer:     framer,
		serializer: concurrent.NewCallbackSerializer(ctx),
		cancel:     cancel,
	}
	cc.encoder = hpack.NewEncoder(&cc.encoderBuf)
	cc.decoder = hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		log.Printf("emit: %+v", f)
	})

	if er := framer.WriteSettings(http2.Setting{
		ID:  http2.SettingMaxConcurrentStreams,
		Val: 1024,
	}); er != nil {
		cc.close()
		log.Printf("[clientconn] write settings failed: %v", er)
		return nil, er
	}
	log.Printf("[clientconn] client write settings")
	go cc.readLoop(ctx)

	return cc, nil
}

func (c *clientConn) call(ctx context.Context, serverURL *url.URL, reqBytes []byte) ([]byte, error) {
	headers := http2Headers(serverURL)
	errCh := make(chan error)
	streamIdCh := make(chan uint32)
	c.serializer.TrySchedule(func(ctx context.Context) {
		if ctx.Err() != nil {
			return
		}
		streamId := atomic.AddUint32(&c.streamIdGen, 2) - 1
		log.Printf("Generated stream ID: %d\n", streamId)
		err := c.framer.WriteHeaders(http2.HeadersFrameParam{
			StreamID:      streamId,
			BlockFragment: c.encodeHpackHeaders(headers),
			EndHeaders:    true,
		})
		if err != nil {
			errCh <- err
		} else {
			streamIdCh <- streamId
		}
	})

	var streamId uint32
	select {
	case err := <-errCh:
		return nil, err
	case v := <-streamIdCh:
		streamId = v
	}

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

	errCh = make(chan error)
	c.serializer.TrySchedule(func(ctx context.Context) {
		err := c.framer.WriteData(streamId, true, reqBytes)
		errCh <- err
	})
	err := <-errCh
	if err != nil {
		return nil, err
	}

	select {
	case <-cs.doneCh:
		log.Printf("[clientconn] grpc-status: %d", cs.grpcStatus)
		log.Printf("[clientconn] grpc-message: %s", cs.grpcMessage)
		return cs.payload, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *clientConn) readLoop(ctx context.Context) {
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

func (cc *clientConn) close() {
	cc.cancel()
	if cc.conn != nil {
		cc.conn.Close()
	}
}

func (c *clientConn) readFrame(ctx context.Context) error {
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

func (c *clientConn) processDataFrame(f *http2.DataFrame) error {
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

func (c *clientConn) processHeaderFrame(f *http2.HeadersFrame) error {
	if v, loaded := c.clientStreams.Load(f.StreamID); loaded {
		if cs, ok := v.(*clientStream); ok {
			headers, err := c.decoder.DecodeFull(f.HeaderBlockFragment())

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

func (c *clientConn) processRSTStreamFrame(f *http2.RSTStreamFrame) error {
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

func (c *clientConn) processSettingsFrame(f *http2.SettingsFrame) error {
	log.Printf("Server Settings [%d]: ", f.NumSettings())
	for i := 0; i < f.NumSettings(); i++ {
		settings := f.Setting(i)
		log.Printf("\t%+v: %d\n", settings.ID, settings.Val)
	}

	if f.IsAck() {
		return nil
	}

	errCh := make(chan error)
	c.serializer.TrySchedule(func(ctx context.Context) {
		if ctx.Err() != nil {
			return
		}
		err := c.framer.WriteSettingsAck()
		// 	close(c.settingsAcked)
		errCh <- err
	})
	return <-errCh
}

func (c *clientConn) processPingFrame(f *http2.PingFrame) error {
	if f.IsAck() {
		return nil
	}

	errCh := make(chan error)
	c.serializer.TrySchedule(func(ctx context.Context) {
		if ctx.Err() != nil {
			return
		}
		err := c.framer.WritePing(true, f.Data)
		// 	close(c.settingsAcked)
		errCh <- err
	})
	return <-errCh
}

func (c *clientConn) processGoAwayFrame(f *http2.GoAwayFrame) error {
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

func (c *clientConn) encodeHpackHeaders(headers []hpack.HeaderField) []byte {
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

func http2Headers(serverURL *url.URL) []hpack.HeaderField {
	return []hpack.HeaderField{
		{Name: ":method", Value: "POST"},
		{Name: ":scheme", Value: serverURL.Scheme},
		{Name: ":authority", Value: serverURL.Host},
		{Name: ":path", Value: serverURL.RequestURI()},
		{Name: "content-type", Value: "application/grpc"},
		{Name: "te", Value: "trailers"},
		{Name: "grpc-encoding", Value: "identity"},
		{Name: "grpc-accept-encoding", Value: "identity"},
	}
}

func dialConn(serverURL *url.URL) (net.Conn, error) {
	addr := getHostAddress(serverURL)
	if serverURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // NOTE: For testing only. Do not use in production.
			NextProtos:         []string{"h2", "http/1.1"},
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return nil, err
		}
		return conn, nil
	} else {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Printf("dial conn err: %v\n", err)
			return nil, err
		}
		return conn, nil
	}
}

func getHostAddress(parsedURL *url.URL) string {
	parsedURL.RequestURI()
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
