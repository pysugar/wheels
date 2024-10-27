package http2

import (
	"encoding/binary"
	"io"
	"log"
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

func ReadFrames(r io.Reader) error {
	// Read frames
	for {
		frameHeader := make([]byte, 9)
		n, err := io.ReadFull(r, frameHeader)
		if err != nil {
			log.Printf("Failed to read frame header, length: %d, err: %v\n", n, err)
			return err
		}

		length := binary.BigEndian.Uint32(append([]byte{0}, frameHeader[:3]...))
		frameType := frameHeader[3]
		flags := frameHeader[4]
		streamID := binary.BigEndian.Uint32(frameHeader[5:9]) & 0x7FFFFFFF

		log.Printf("Received frame: Length=%d, Type=%s, Flags=%d, StreamID=%d\n", length, frameTypes[frameType], flags, streamID)

		payload, err := readPayload(r, length)
		if err != nil {
			return err
		}

		log.Printf("Payload(%d): %s\n", length, string(payload))
	}
}

func readPayload(r io.Reader, length uint32) ([]byte, error) {
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
