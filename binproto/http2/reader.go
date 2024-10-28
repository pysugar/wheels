package http2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"

	"golang.org/x/net/http2/hpack"
	"google.golang.org/protobuf/proto"
)

const (
	http2frameHeaderLen = 9
	ack                 = uint8(0x01)
)

type http2FrameHeader struct {
	valid bool // caller can access []byte fields in the Frame

	// Type is the 1 byte frame type. There are ten standard frame types, but extension frame types may be written
	// by WriteRawFrame and will be returned by ReadFrame (as UnknownFrame).
	Type uint8

	// Flags are the 1 byte of 8 potential bit flags per frame. They are specific to the frame type.
	Flags uint8

	// Length is the length of the frame, not including the 9 byte header. The maximum size is one byte less than
	// 16MB (uint24), but only frames up to 16KB are allowed without peer agreement.
	Length uint32

	// StreamID is which stream this frame is for. Certain frames are not stream-specific, in which case this field is 0.
	StreamID uint32
}

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

func ReadFrames(rw io.ReadWriter, response proto.Message) error {
	pingCount := 0
	for {
		frameHeader, err := readFrameHeader(rw)
		if err != nil {
			return err
		}

		log.Printf("Received frame: Length=%d, Type=%s(%d), Flags=%d, StreamID=%d\n", frameHeader.Length,
			frameTypes[frameHeader.Type], frameHeader.Type, frameHeader.Flags, frameHeader.StreamID)

		if frameHeader.Type == 3 { // RST_STREAM
			return nil
		}

		payload, err := readFramePayload(rw, frameHeader.Length)
		if err != nil {
			return err
		}
		log.Printf("Payload(%d): %v(%s)\n", frameHeader.Length, payload, payload)

		if frameHeader.Type == 1 {
			buf := bytes.NewBuffer(payload)
			decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
				log.Printf("Decoded Header: %s: %s", f.Name, f.Value)
			})

			for buf.Len() > 0 {
				if n, er := decoder.Write(buf.Next(buf.Len())); err != nil {
					if er == io.EOF {
						break
					}
					log.Printf("failed to decode header field: %v, n = %d\n", err, n)
					break
				}
			}
		} else if frameHeader.Type == 6 {
			if pingCount%5 == 0 {
				if er := WritePingFrame(rw, payload); er != nil {
					return er
				}
			}
			pingCount++
		} else if frameHeader.Type == 4 {
			if (frameHeader.Flags & ack) == ack {
				log.Printf("Settings ACK: %v\n", frameHeader.Flags)
			} else {
				//flags := frameHeader.Flags | ack
				//if er := WriteSettingsFrame(rw, flags, payload); er != nil {
				//	return er
				//}
			}
		} else if frameHeader.Type == 0 {
			if er := DecodeGrpcFrame(payload, response); er == nil {
				log.Printf("receive data response: %v\n", response)
			}
		}
	}
}

func readFrameHeader(r io.Reader) (*http2FrameHeader, error) {
	var headerBuf [http2frameHeaderLen]byte
	n, err := io.ReadFull(r, headerBuf[:http2frameHeaderLen])
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Printf("Failed to read frame header, length: %d, err: %v\n", n, err)
		}
		return nil, err
	}
	return &http2FrameHeader{
		Length:   uint32(headerBuf[0])<<16 | uint32(headerBuf[1])<<8 | uint32(headerBuf[2]),
		Type:     headerBuf[3],
		Flags:    headerBuf[4],
		StreamID: binary.BigEndian.Uint32(headerBuf[5:]) & (1<<31 - 1), // binary.BigEndian.Uint32(frameHeader[5:9]) & 0x7FFFFFFF
		valid:    true,
	}, nil
}

func readFramePayload(r io.Reader, length uint32) ([]byte, error) {
	payload := make([]byte, length)
	bytesRead := 0
	for bytesRead < int(length) {
		n, err := r.Read(payload[bytesRead:])
		if err != nil {
			log.Println("Failed to read frame payload:", err)
			return payload, err
		}
		bytesRead += n
	}
	return payload, nil
}
