package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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

	maxPingRetryTimes = 3
)

var (
	clientPreface = []byte(ClientPreface)

	insecureTLSConfig = &tls.Config{
		InsecureSkipVerify: true, // NOTE: For testing only. Do not use in production.
		NextProtos:         []string{"h2", "http/1.1"},
	}

	clientConnIdGen uint32
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
		id                     uint32
		dopts                  *dialOptions
		target                 string
		conn                   net.Conn
		framer                 *http2.Framer
		serializer             *concurrent.CallbackSerializer
		cancel                 context.CancelFunc
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
		settingsAcked          chan struct{}
		pingRequests           map[uint64]chan uint64
		nextRequest            uint64 // Next key to use in pingRequests.
	}
)

func (cs *clientStream) done() {
	cs.doneOnce.Do(func() {
		close(cs.doneCh)
	})
}

func dialContext(ctx context.Context, target string, opts ...DialOption) (cc *clientConn, err error) {
	dopts := evaluateOptions(opts)
	conn := dopts.conn
	if conn == nil {
		conn, err = dialConn(target, dopts.useTLS)
		if err != nil {
			return nil, err
		}

		log.Printf("[%T] Send HTTP/2 Client Preface: %s\n", conn, clientPreface)
		if _, er := conn.Write(clientPreface); er != nil {
			conn.Close()
			err = er
		}
	}

	framer := http2.NewFramer(conn, conn)
	ctx, cancel := context.WithCancel(context.Background())
	cc = &clientConn{
		id:                   atomic.AddUint32(&clientConnIdGen, 1),
		dopts:                dopts,
		target:               target,
		conn:                 conn,
		framer:               framer,
		serializer:           concurrent.NewCallbackSerializer(ctx),
		cancel:               cancel,
		maxConcurrentStreams: 1000,
		settingsAcked:        make(chan struct{}),
		pingRequests:         make(map[uint64]chan uint64),
	}

	cc.encoder = hpack.NewEncoder(&cc.encoderBuf)
	cc.decoder = hpack.NewDecoder(4096, nil)

	if er := framer.WriteSettings(http2.Setting{
		ID:  http2.SettingMaxConcurrentStreams,
		Val: 100,
	}); er != nil {
		cc.Close()
		log.Printf("[clientConn] write settings failed: %v", er)
		return nil, er
	}
	cc.verbose("[clientConn] client write settings")
	go cc.readLoop(ctx)

	select {
	case <-cc.settingsAcked:
		cc.verbose("[clientConn] Settings acknowledged by server")
		cc.maxConcurrentSemaphore = make(chan struct{}, cc.maxConcurrentStreams)
		return cc, nil
	case <-time.After(dopts.timeout):
		cc.Close()
		return nil, fmt.Errorf("[clientConn] timeout waiting for settings ack")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *clientConn) do(ctx context.Context, req *http.Request) (res *http.Response, err error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("clientConn closed")
	}
	c.mu.Unlock()

	if er := validateRequest(req); er != nil {
		return nil, er
	}

	c.maxConcurrentSemaphore <- struct{}{}
	defer func() {
		<-c.maxConcurrentSemaphore
	}()

	cs, err := c.writeHeaders(ctx, req)
	if err != nil {
		return nil, err
	}
	defer c.clientStreams.Delete(cs.streamId)

	if req.Body != nil {
		if er := c.writeBody(ctx, cs.streamId, req.Body); er != nil {
			return nil, er
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

func (c *clientConn) writeHeaders(ctx context.Context, req *http.Request) (*clientStream, error) {
	headerFields, err := c.createHeaderFields(req)
	if err != nil {
		return nil, err
	}

	clientStreamCh := make(chan struct {
		cs  *clientStream
		err error
	})
	c.serializer.TrySchedule(func(ctx context.Context) {
		if ctx.Err() != nil {
			return
		}

		cs, er := c.startNewClientStream(headerFields, req.Body == nil)
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
		c.verbose("[clientConn] write headers done")
		cs = ret.cs
		return cs, nil
	}
}

func (c *clientConn) writeBody(ctx context.Context, streamId uint32, body io.ReadCloser) error {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	errCh := make(chan error)
	c.serializer.TrySchedule(func(ccCtx context.Context) {
		if ccCtx.Err() != nil {
			return
		}
		if ctx.Err() != nil {
			errCh <- ctx.Err()
			return
		}
		errCh <- c.framer.WriteData(streamId, true, bodyBytes)
	})

	c.verbose("[clientConn] write body done")
	return <-errCh
}

func (c *clientConn) startNewClientStream(headerFields []hpack.HeaderField, endStream bool) (*clientStream, error) {
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

func (c *clientConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}

	c.closed = true
	c.cancel()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *clientConn) isValid(ctx context.Context) bool {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		log.Printf("clientConn is invalid, conn is closed, target: %s\n", c.target)
		return false
	}

	req := make(chan uint64, 1)
	reqKey := c.nextRequestKeyLocked()
	if len(c.pingRequests) == 0 {
		c.pingRequests[reqKey] = req
		c.mu.Unlock()

		ok := c.sendRecvPingWithRetries(ctx, reqKey, req)
		if !ok {
			c.mu.Lock()
			delete(c.pingRequests, reqKey)
			c.mu.Unlock()
		}
		return ok
	}

	c.pingRequests[reqKey] = req
	c.mu.Unlock()

	waitStart := time.Now()
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
	}

	select {
	case ackKey, ok := <-req:
		log.Printf("[clientConn] ping acked-: %v, target: %s, req: %d, ack: %d, cost: %dμs", ok, c.target,
			reqKey, ackKey, time.Since(waitStart).Microseconds())
		return true
	case <-ctx.Done():
		log.Printf("clientConn is invalid, read ping ack timeout, target: %s, cost: %dμs\n", c.target,
			time.Since(waitStart).Microseconds())
		c.mu.Lock()
		delete(c.pingRequests, reqKey)
		c.mu.Unlock()
		return false
	}
}

func (c *clientConn) readLoop(ctx context.Context) {
	for {
		c.mu.Lock()
		closed := c.closed
		c.mu.Unlock()
		if closed {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
			if err := c.readFrame(ctx); err != nil {
				if errors.Is(err, io.EOF) {
					log.Printf("Connection closed by remote host")
					c.cancel()
					return
				}
				log.Printf("Failed to read frame: %v", err)
			}
		}
	}
}

func (c *clientConn) sendRecvPingWithRetries(ctx context.Context, reqKey uint64, req <-chan uint64) bool {
	var data [8]byte
	binary.BigEndian.PutUint64(data[:], reqKey)
	pingTimeout := time.Second
	waitStart := time.Now()
	for i := 0; i < maxPingRetryTimes; i++ {
		if err := c.sendPing(false, data); err != nil {
			return false
		}

		select {
		case ackKey, ok := <-req:
			log.Printf("[clientConn] ping acked+: %v, target: %s, req: %d, ack: %d, cost: %dμs", ok, c.target,
				reqKey, ackKey, time.Since(waitStart).Microseconds())
			return true
		case <-time.After(pingTimeout):
			log.Printf("[clientConn] ping timeout, req: %v, retrying... (%d/%d)", reqKey, i+1, maxPingRetryTimes)
			continue
		case <-ctx.Done():
			log.Printf("[clientConn] ping timeout, context done, target: %s, req: %d, cost: %dμs, err: %v",
				c.target, reqKey, time.Since(waitStart).Microseconds(), ctx.Err())
			return false
		}
	}
	log.Printf("Ping failed after %d retries, cost: %dμs", maxPingRetryTimes, time.Since(waitStart).Microseconds())
	return false
}

func (c *clientConn) sendPing(ack bool, data [8]byte) error {
	errCh := make(chan error)
	c.serializer.TrySchedule(func(ctx context.Context) {
		if ctx.Err() != nil {
			return
		}
		err := c.framer.WritePing(ack, data)
		errCh <- err
	})
	return <-errCh
}

func (c *clientConn) putPingAck(data [8]byte) {
	reqKey := binary.BigEndian.Uint64(data[:])
	c.mu.Lock()
	defer c.mu.Unlock()
	c.putPingAckLocked(reqKey)
}

func (c *clientConn) putPingAckLocked(ackKey uint64) bool {
	if c.closed {
		return false
	}

	waitingLen := len(c.pingRequests)
	if n := waitingLen; n > 0 {
		for reqKey, req := range c.pingRequests {
			if _, has := c.pingRequests[reqKey]; !has {
				log.Printf("[clientConn] invalid ack key: %d, maybe it has already processed", ackKey)
				continue
			}
			if reqKey >= ackKey {
				c.verbose("[clientConn] receive ping ack, ack key: %d, req key %d, waiting count: %d",
					ackKey, reqKey, waitingLen)
				req <- ackKey
				delete(c.pingRequests, reqKey) // Remove from pending requests.
				close(req)
			}
		}
		return true
	}
	return false
}

func (c *clientConn) nextRequestKeyLocked() uint64 {
	next := c.nextRequest
	c.nextRequest++
	return next
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

	c.verbose("[stream-%03d] Received %s Frame: %+v", frame.Header().StreamID, frame.Header().Type, frame)

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
			c.verbose("Received trailer (%s: %s)", hf.Name, hf.Value)
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
			c.verbose("Received header (%s: %s)", hf.Name, hf.Value)
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
	c.verbose("Server Settings [%d], isAck: %v: ", f.NumSettings(), f.IsAck())

	if f.IsAck() {
		close(c.settingsAcked)
		return nil
	}

	for i := 0; i < f.NumSettings(); i++ {
		settings := f.Setting(i)
		if settings.ID == http2.SettingMaxConcurrentStreams {
			c.maxConcurrentStreams = settings.Val
		}
		c.verbose("\t%+v: %d\n", settings.ID, settings.Val)
	}

	errCh := make(chan error)
	c.serializer.TrySchedule(func(ctx context.Context) {
		if ctx.Err() != nil {
			return
		}
		err := c.framer.WriteSettingsAck()
		errCh <- err
	})
	return <-errCh
}

func (c *clientConn) processPingFrame(f *http2.PingFrame) error {
	if f.IsAck() {
		c.putPingAck(f.Data)
		return nil
	}

	return c.sendPing(true, f.Data)
}

func (c *clientConn) processGoAwayFrame(f *http2.GoAwayFrame) error {
	c.verbose("Received GOAWAY frame: LastStreamID=%d, SteamID=%d, ErrorCode=%d, DebugData=%s", f.LastStreamID,
		f.StreamID, f.ErrCode, f.DebugData())
	defer c.Close()
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
		c.verbose("Send Header %s: %s\n", header.Name, header.Value)
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

func (c *clientConn) verbose(format string, v ...any) {
	if c.dopts.verbose {
		log.Printf(format, v...)
	}
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
