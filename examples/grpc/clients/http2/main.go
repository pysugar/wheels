package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	http2tool "github.com/pysugar/wheels/binproto/http2"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

var streamIdGen int32 = -1

type (
	activeStream struct {
		streamId    uint32
		doneCh      chan struct{}
		grpcStatus  int
		grpcMessage string
		payload     []byte
	}

	grpcClient struct {
		framer        *http2.Framer
		activeStreams sync.Map // streamID -> activeStream
	}
)

func main() {
	// serverURL, _ := url.Parse("https://127.0.0.1:8443")
	serverURL, _ := url.Parse("http://127.0.0.1:50051")

	framer, err := NewFramer(serverURL)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go readLoop(ctx, framer)

	framer.WriteSettings()
	streamId := uint32(atomic.AddInt32(&streamIdGen, 2))
	headers := grpcHeaders(serverURL, "grpc.health.v1.Health/Check")
	framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamId,
		BlockFragment: encodeGrpcHeaders(headers),
		EndHeaders:    true,
	})

	framer.WriteData(streamId, true, http2tool.EncodeGrpcPayload([]byte{}))

	time.Sleep(10 * time.Second)
}

//func (c *grpcClient) grpcCall(ctx context.Context, req, res proto.Message) error {
//
//}

func encodeGrpcHeaders(headers []hpack.HeaderField) []byte {
	var headersBuffer bytes.Buffer
	encoder := hpack.NewEncoder(&headersBuffer)
	for _, header := range headers {
		err := encoder.WriteField(hpack.HeaderField{
			Name:  header.Name,
			Value: header.Value,
		})
		if err != nil {
			log.Printf("failed to encode header field: %v", err)
			continue
		}
		log.Printf("Send Header %s: %s\n", header.Name, header.Value)
	}
	return headersBuffer.Bytes()
}

func grpcHeaders(serverURL *url.URL, fullMethod string) []hpack.HeaderField {
	return []hpack.HeaderField{
		{Name: ":method", Value: "POST"},
		{Name: ":scheme", Value: serverURL.Scheme},
		{Name: ":authority", Value: serverURL.Host},
		{Name: ":path", Value: fullMethod},
		{Name: "content-type", Value: "application/grpc"},
		{Name: "te", Value: "trailers"},
	}
}

func NewFramer(serverURL *url.URL) (*http2.Framer, error) {
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
		if n, er := conn.Write(clientPreface); er != nil {
			log.Printf("Failed to send HTTP/2 Client Preface, err = %v, n = %d\n", er, n)
			return nil, er
		}
	}

	return http2.NewFramer(conn, conn), nil
}

func readLoop(ctx context.Context, framer *http2.Framer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := processFrame(framer); err != nil {
				if err == io.EOF {
					log.Printf("Connection closed by remote host")
					return
				}
				log.Printf("Failed to read frame: %v", err)
			}
		}
	}
}

func processFrame(framer *http2.Framer) error {
	frame, err := framer.ReadFrame()
	if err != nil {
		return err
	}

	log.Printf("[stream-%03d] Received %s frame: %+v", frame.Header().StreamID, frame.Header().Type, frame)

	switch f := frame.(type) {
	case *http2.DataFrame: // 0
		log.Printf("Response Data: %+v", f.Data())
	case *http2.HeadersFrame: // 1
	case *http2.PriorityFrame: // 2
	case *http2.RSTStreamFrame: // 3
	case *http2.SettingsFrame: // 4
		if f.IsAck() {
			return nil
		}
		return framer.WriteSettingsAck()
	case *http2.PushPromiseFrame: // 5
	case *http2.PingFrame: // 6
		if f.IsAck() {
			return nil
		}
		return framer.WritePing(true, f.Data)
	case *http2.GoAwayFrame: // 7
		return io.EOF
	case *http2.WindowUpdateFrame: //8
	case *http2.ContinuationFrame: //9
	default:
		log.Printf("Received unknown frame: %T", f)
	}
	return nil
}
