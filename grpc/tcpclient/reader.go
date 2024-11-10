package tcpclient

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
)

type (
	Flags uint8

	frameHeader struct {
		valid    bool
		Type     uint8
		Flags    Flags
		Length   uint32
		StreamID uint32
	}

	frameHandler func(*frameHeader, []byte) error
)

const (
	http2frameHeaderLen = 9

	// Data Frame
	FlagDataEndStream Flags = 0x1
	FlagDataPadded    Flags = 0x8

	// Headers Frame
	FlagHeadersEndStream  Flags = 0x1
	FlagHeadersEndHeaders Flags = 0x4
	FlagHeadersPadded     Flags = 0x8
	FlagHeadersPriority   Flags = 0x20

	// Settings Frame
	FlagSettingsAck Flags = 0x1

	// Ping Frame
	FlagPingAck Flags = 0x1

	// Continuation Frame
	FlagContinuationEndHeaders Flags = 0x4

	FlagPushPromiseEndHeaders Flags = 0x4
	FlagPushPromisePadded     Flags = 0x8
)

var (
	frameTypes = map[byte]string{
		0: "DATA",
		1: "HEADERS",
		2: "PRIORITY",
		3: "RST_STREAM",
		4: "SETTINGS",
		5: "PUSH_PROMISE",
		6: "PING",
		7: "GOAWAY",
		8: "WINDOW_UPDATE",
		9: "CONTINUATION",
	}
)

// Has reports whether f contains all (0 or more) flags in v.
func (f Flags) Has(v Flags) bool {
	return (f & v) == v
}

func (c *grpcClient) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := c.readFrame(ctx, c.conn); err != nil {
				if err == io.EOF {
					log.Printf("Connection closed by remote host")
					return
				}
				log.Printf("Failed to read frame: %v", err)
			}
		}
	}
}

func (c *grpcClient) readFrame(ctx context.Context, r io.Reader) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	fh, err := readFrameHeader(r)
	if err != nil {
		return fmt.Errorf("failed to read frame header: %v", err)
	}

	log.Printf("[Stream-%d] Received frame: Length=%d, Type=%s(%d), Flags=%d\n", fh.StreamID,
		fh.Length, frameTypes[fh.Type], fh.Type, fh.Flags)
	payload, err := readFramePayload(r, fh.Length)
	if err != nil {
		return fmt.Errorf("failed to read frame payload: %v", err)
	}

	if handle, ok := c.frameHandlers[fh.Type]; ok {
		if er := handle(fh, payload); er != nil {
			return fmt.Errorf("failed to handle frame: %v", er)
		}
	} else {
		log.Printf("[ERROR] unknown frame type: %v", fh.Type)
	}

	return nil
}

func (c *grpcClient) handleDataFrame(fh *frameHeader, payload []byte) error {
	if v, loaded := c.clientStreams.Load(fh.StreamID); loaded {
		if cs, ok := v.(*clientStream); ok {
			cs.mu.Lock()
			cs.payload = append(cs.payload, payload...)
			cs.mu.Unlock()

			if fh.Flags.Has(FlagDataEndStream) {
				c.clientStreams.Delete(fh.StreamID)
				close(cs.doneCh)
			}
		}
	}
	return nil
}

func (c *grpcClient) handleHeadersFrame(fh *frameHeader, payload []byte) error {
	if v, loaded := c.clientStreams.Load(fh.StreamID); loaded {
		if cs, ok := v.(*clientStream); ok {
			c.decodeMu.Lock()
			headers, err := c.decoder.DecodeFull(payload)
			c.decodeMu.Unlock()

			if err != nil {
				return fmt.Errorf("[stream-%03d] Failed to decode headers: %w", fh.StreamID, err)
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
			if fh.Flags.Has(FlagHeadersEndStream) {
				c.clientStreams.Delete(fh.StreamID)
				close(cs.doneCh)
			}
		}
	}
	return nil
}

func (c *grpcClient) handleRSTStreamFrame(fh *frameHeader, payload []byte) error {
	if v, loaded := c.clientStreams.Load(fh.StreamID); loaded {
		if cs, ok := v.(*clientStream); ok {
			cs.grpcStatus = int(binary.BigEndian.Uint32(payload[:4]))
			cs.grpcMessage = "Stream reset by server"
			c.clientStreams.Delete(fh.StreamID)
			close(cs.doneCh)
		}
	}
	return nil
}

func (c *grpcClient) handleSettingsFrame(fh *frameHeader, payload []byte) error {
	if fh.Flags.Has(FlagSettingsAck) {
		return nil
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := WriteSettingsFrame(c.conn, FlagSettingsAck, nil); err != nil {
		return err
	}
	close(c.settingsAcked)
	return nil
}

func (c *grpcClient) handlePingFrame(fh *frameHeader, payload []byte) error {
	if fh.Flags.Has(FlagPingAck) {
		return nil
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return WritePingFrame(c.conn, FlagPingAck, payload)
}

func (c *grpcClient) handleGoAwayFrame(fh *frameHeader, payload []byte) error {
	c.clientStreams.Range(func(key, value any) bool {
		if cs, ok := value.(*clientStream); ok {
			cs.grpcStatus = int(binary.BigEndian.Uint32(payload[4:8]))
			cs.grpcMessage = "stream go away"
			c.clientStreams.Delete(cs.streamId)
			close(cs.doneCh)
		}
		return true
	})
	return nil //io.EOF
}

func readFrameHeader(r io.Reader) (*frameHeader, error) {
	var headerBuf [http2frameHeaderLen]byte
	n, err := io.ReadFull(r, headerBuf[:http2frameHeaderLen])
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Printf("Failed to read frame header, length: %d, err: %v\n", n, err)
		}
		return nil, err
	}
	return &frameHeader{
		Length:   uint32(headerBuf[0])<<16 | uint32(headerBuf[1])<<8 | uint32(headerBuf[2]),
		Type:     headerBuf[3],
		Flags:    Flags(headerBuf[4]),
		StreamID: binary.BigEndian.Uint32(headerBuf[5:]) & (1<<31 - 1), // binary.BigEndian.Uint32(frameHeader[5:9]) & 0x7FFFFFFF
		valid:    true,
	}, nil
}

func readFramePayload(r io.Reader, length uint32) ([]byte, error) {
	payload := make([]byte, length)
	n, err := io.ReadFull(r, payload)
	if err != nil {
		log.Printf("Failed to read frame payload: %v, bytes read: %d", err, n)
		return payload[:n], err
	}
	return payload, nil
}
