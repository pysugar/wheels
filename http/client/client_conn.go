package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
		statusCode      int
		responseHeaders http.Header
		payload         bytes.Buffer
		trailers        http.Header
		mu              sync.Mutex
		doneCh          chan struct{}
		doneOnce        sync.Once
	}

	clientConn struct {
		serverURL              *url.URL
		conn                   net.Conn
		framer                 *http2.Framer
		serializer             *concurrent.CallbackSerializer
		streamIdGen            uint32
		clientStreams          sync.Map // streamID -> activeStream
		maxConcurrentStreams   uint32
		maxConcurrentSemaphore chan struct{}
		encoder                *hpack.Encoder
		decoder                *hpack.Decoder
		encoderBuf             bytes.Buffer // encoder buffer
		encodeMu               sync.Mutex
		closed                 bool
		mu                     sync.Mutex
		cancel                 context.CancelFunc
		settingsAcked          chan struct{}
	}
)

func (cs *clientStream) done() {
	cs.doneOnce.Do(func() {
		close(cs.doneCh)
	})
}

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
		serverURL:            serverURL,
		conn:                 conn,
		framer:               framer,
		serializer:           concurrent.NewCallbackSerializer(ctx),
		cancel:               cancel,
		maxConcurrentStreams: 1000,
		settingsAcked:        make(chan struct{}),
	}
	cc.encoder = hpack.NewEncoder(&cc.encoderBuf)
	cc.decoder = hpack.NewDecoder(4096, nil)

	if er := framer.WriteSettings(http2.Setting{
		ID:  http2.SettingMaxConcurrentStreams,
		Val: 1024,
	}); er != nil {
		cc.close()
		log.Printf("[clientConn] write settings failed: %v", er)
		return nil, er
	}
	log.Printf("[clientConn] client write settings")
	go cc.readLoop(ctx)

	select {
	case <-cc.settingsAcked:
		log.Printf("[clientConn] Settings acknowledged by server")
	case <-time.After(5 * time.Second):
		cc.close()
		return nil, fmt.Errorf("timeout waiting for settings ack")
	}

	cc.maxConcurrentSemaphore = make(chan struct{}, cc.maxConcurrentStreams)
	return cc, nil
}

func (c *clientConn) do(ctx context.Context, req *http.Request) (res *http.Response, err error) {
	if er := validateRequest(req); er != nil {
		return nil, er
	}

	headerFields, err := c.createHeaderFields(req)
	if err != nil {
		return nil, err
	}

	c.maxConcurrentSemaphore <- struct{}{}
	defer func() {
		<-c.maxConcurrentSemaphore
	}()

	clientStreamCh := make(chan struct {
		cs  *clientStream
		err error
	})
	c.serializer.TrySchedule(func(ctx context.Context) {
		if ctx.Err() != nil {
			return
		}

		cs, er := c.startClientStream(headerFields, req.Body == nil)
		clientStreamCh <- struct {
			cs  *clientStream
			err error
		}{cs, er}
	})

	var cs *clientStream
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case ret := <-clientStreamCh:
		if ret.err != nil {
			return nil, ret.err
		}
		cs = ret.cs
		defer c.clientStreams.Delete(ret.cs.streamId)
	}

	if req.Body != nil {
		reqBytes, er := io.ReadAll(req.Body)
		if er != nil {
			return nil, er
		}
		errCh := make(chan error)
		c.serializer.TrySchedule(func(ctx context.Context) {
			errCh <- c.framer.WriteData(cs.streamId, true, reqBytes)
		})
		err = <-errCh

		if err != nil {
			return nil, err
		}
	}

	select {
	case <-cs.doneCh:
		resp := &http.Response{
			StatusCode:    cs.statusCode,
			Status:        fmt.Sprintf("%d %s", cs.statusCode, http.StatusText(cs.statusCode)),
			Proto:         "HTTP/2.0",
			ProtoMajor:    2,
			ProtoMinor:    0,
			Header:        cs.responseHeaders,
			Trailer:       cs.trailers,
			Body:          io.NopCloser(&cs.payload),
			ContentLength: int64(cs.payload.Len()),
			Request:       req,
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *clientConn) startClientStream(headerFields []hpack.HeaderField, endStream bool) (*clientStream, error) {
	headersPayload := c.encodeHpackHeaders(headerFields)

	streamId := atomic.AddUint32(&c.streamIdGen, 2) - 1
	err := c.framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamId,
		BlockFragment: headersPayload,
		EndHeaders:    true,
		EndStream:     endStream,
	})

	if err != nil {
		return nil, err
	}

	cs := &clientStream{
		streamId:        streamId,
		activeAt:        time.Now(),
		doneCh:          make(chan struct{}),
		responseHeaders: make(http.Header),
		trailers:        make(http.Header),
	}
	c.clientStreams.Store(streamId, cs)

	return cs, nil
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

func (c *clientConn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}

	c.closed = true
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
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

	log.Printf("[stream-%03d] Received %s Frame: %+v", frame.Header().StreamID, frame.Header().Type, frame)

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
			defer cs.mu.Unlock()
			cs.payload.Write(f.Data())
			if f.StreamEnded() {
				cs.done()
			}
		}
	}
	return nil
}

func (c *clientConn) processHeaderFrame(f *http2.HeadersFrame) error {
	v, loaded := c.clientStreams.Load(f.StreamID)
	if !loaded {
		log.Printf("Stream %d not found", f.StreamID)
		return nil
	}
	cs, ok := v.(*clientStream)
	if !ok {
		log.Printf("Stream %d is not a clientStream", f.StreamID)
		return nil
	}

	headers, err := c.decoder.DecodeFull(f.HeaderBlockFragment())
	if err != nil {
		return fmt.Errorf("[stream-%03d] Failed to decode headers: %w", f.StreamID, err)
	}

	if f.StreamEnded() {
		for _, hf := range headers {
			log.Printf("Received trailer (%s: %s)", hf.Name, hf.Value)
			cs.trailers.Add(hf.Name, hf.Value)
			if hf.Name == ":status" {
				if statusCode, er := strconv.Atoi(hf.Value); er == nil {
					cs.statusCode = statusCode
				}
			}
		}
		cs.done()
	} else {
		for _, hf := range headers {
			log.Printf("Received header (%s: %s)", hf.Name, hf.Value)
			cs.responseHeaders.Add(hf.Name, hf.Value)
			if hf.Name == ":status" {
				if statusCode, er := strconv.Atoi(hf.Value); er == nil {
					cs.statusCode = statusCode
				}
			}
		}
	}
	return nil
}

func (c *clientConn) processRSTStreamFrame(f *http2.RSTStreamFrame) error {
	if v, loaded := c.clientStreams.Load(f.StreamID); loaded {
		if cs, ok := v.(*clientStream); ok {
			cs.trailers.Add("grpc-status", strconv.Itoa(int(f.ErrCode)))
			cs.trailers.Add("grpc-message", "stream reset by server")
			cs.done()
		}
	}
	return nil
}

func (c *clientConn) processSettingsFrame(f *http2.SettingsFrame) error {
	log.Printf("Server Settings [%d]: ", f.NumSettings())
	for i := 0; i < f.NumSettings(); i++ {
		settings := f.Setting(i)
		if settings.ID == http2.SettingMaxConcurrentStreams {
			c.maxConcurrentStreams = settings.Val
		}
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
		close(c.settingsAcked)
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
		errCh <- err
	})
	return <-errCh
}

func (c *clientConn) processGoAwayFrame(f *http2.GoAwayFrame) error {
	log.Printf("Received GOAWAY frame: LastStreamID=%d, SteamID=%d, ErrorCode=%d, DebugData=%s", f.LastStreamID,
		f.StreamID, f.ErrCode, f.DebugData())
	c.clientStreams.Range(func(key, value any) bool {
		if cs, ok := value.(*clientStream); ok {
			cs.trailers.Add("grpc-status", strconv.Itoa(int(f.ErrCode)))
			cs.trailers.Add("grpc-message", string(f.DebugData()))
			c.clientStreams.Delete(cs.streamId)
			cs.done()
		}
		return true
	})
	return nil //io.EOF
}

func (c *clientConn) encodeHpackHeaders(headers []hpack.HeaderField) []byte {
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

func (c *clientConn) createHeaderFields(req *http.Request) ([]hpack.HeaderField, error) {
	scheme := req.URL.Scheme
	if scheme == "" {
		scheme = "https"
	}
	authority := req.Host
	if authority == "" {
		authority = req.URL.Host
	}

	headers := []hpack.HeaderField{
		{Name: ":method", Value: req.Method},
		{Name: ":scheme", Value: scheme},
		{Name: ":authority", Value: authority},
		{Name: ":path", Value: req.URL.RequestURI()},
	}

	for key, values := range req.Header {
		for _, value := range values {
			headers = append(headers, hpack.HeaderField{Name: strings.ToLower(key), Value: strings.ToLower(value)})
		}
	}

	return headers, nil
}

func validateRequest(req *http.Request) error {
	if req.URL == nil {
		return fmt.Errorf("request URL is nil")
	}
	if req.Method == "" {
		return fmt.Errorf("request method is empty")
	}
	return nil
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

// getHostAddress constructs the host address from the URL
func getHostAddress(parsedURL *url.URL) string {
	host := parsedURL.Host
	if !hasPort(host) {
		switch parsedURL.Scheme {
		case "https":
			host += ":443"
		case "http":
			host += ":80"
		}
	}
	return host
}

// hasPort checks if the host includes a port
func hasPort(host string) bool {
	_, _, err := net.SplitHostPort(host)
	return err == nil
}
