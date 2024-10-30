package http2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/net/http2/hpack"
)

var (
	specialHeaders = map[string]bool{
		"Set-Cookie":                    true,
		"Authorization":                 true,
		"WWW-Authenticate":              true,
		"Proxy-Authenticate":            true,
		"Proxy-Authorization":           true,
		"Warning":                       true,
		"Link":                          true,
		"Vary":                          true,
		"Access-Control-Allow-Headers":  true,
		"Access-Control-Expose-Headers": true,
		"Digest":                        true,
	}
)

func WriteDataFrame(w io.Writer, streamID uint32, body []byte) error {
	length := len(body)
	var buf [http2frameHeaderLen]byte
	buf[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	buf[1] = byte((length >> 8) & 0xFF)
	buf[2] = byte(length & 0xFF)
	buf[3] = 0x0                                  // Type: DATA (0x0)
	buf[4] = 0x0                                  // Flags
	binary.BigEndian.PutUint32(buf[5:], streamID) // Stream ID: 1

	if n, err := w.Write(buf[:]); err != nil {
		return err
	} else {
		log.Printf("Write data header success, length = %d\n", n)
	}
	if length > 0 {
		if n, err := w.Write(body); err != nil {
			return err
		} else {
			log.Printf("Write data payload success, length = %d\n", n)
		}
	}
	return nil
}

func sendDataFrame(conn net.Conn, data []byte) {
	length := len(data)
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], uint32(length))
	frameHeader[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	frameHeader[1] = byte((length >> 8) & 0xFF)
	frameHeader[2] = byte(length & 0xFF)
	frameHeader[3] = 0x0                             // Type: DATA (0x0)
	frameHeader[4] = 0x1                             // Flags: END_STREAM (0x1)
	binary.BigEndian.PutUint32(frameHeader[5:], 0x1) // Stream ID: 1

	conn.Write(frameHeader)
	conn.Write(data)
	log.Println("Sent DATA frame")
}

func WriteHeadersFrame(w io.Writer, streamID uint32, headers []hpack.HeaderField) error {
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

	headersPayload := headersBuffer.Bytes()
	length := len(headersPayload)

	var buf [http2frameHeaderLen]byte
	// binary.BigEndian.PutUint32(buf[:4], uint32(length))
	buf[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	buf[1] = byte((length >> 8) & 0xFF)
	buf[2] = byte(length & 0xFF)
	buf[3] = 0x1                                  // Type: HEADERS (0x1)
	buf[4] = 0x4                                  // Flags: END_HEADERS
	binary.BigEndian.PutUint32(buf[5:], streamID) // Stream ID (client-initiated, must be odd)

	if n, err := w.Write(buf[:]); err != nil {
		return err
	} else {
		log.Printf("Write headers header success, length = %d\n", n)
	}
	if n, err := w.Write(headersPayload); err != nil {
		return err
	} else {
		log.Printf("Write headers payload success, length = %d\n", n)
	}
	return nil
}

func sendHeadersFrame(conn net.Conn) {
	// Frame Header for HEADERS frame
	headersPayload := []byte{
		0x88, // :status 200 OK (pre-encoded with HPACK)
	}
	length := len(headersPayload)

	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], uint32(length))
	frameHeader[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	frameHeader[1] = byte((length >> 8) & 0xFF)
	frameHeader[2] = byte(length & 0xFF)
	frameHeader[3] = 0x1                             // Type: HEADERS (0x1)
	frameHeader[4] = 0x4                             // Flags: END_HEADERS (0x4)
	binary.BigEndian.PutUint32(frameHeader[5:], 0x1) // Stream ID: 1

	conn.Write(frameHeader)
	conn.Write(headersPayload)
	log.Println("Sent HEADERS frame")
}

func sendPriorityFrame(conn net.Conn) {
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], 0x5) // Length (5 bytes)
	frameHeader[3] = 0x2                             // Type: PRIORITY (0x2)
	frameHeader[4] = 0x0                             // Flags
	binary.BigEndian.PutUint32(frameHeader[5:], 0x1) // Stream ID: 1

	priorityPayload := make([]byte, 5)
	priorityPayload[0] = 0x0                              // Exclusive bit and Stream Dependency (31 bits)
	binary.BigEndian.PutUint32(priorityPayload[1:], 0x10) // Weight (1-256, actual weight is value + 1)
	conn.Write(frameHeader)
	conn.Write(priorityPayload)
	log.Println("Sent PRIORITY frame")
}

func sendResetStreamFrame(conn net.Conn) {
	// Frame Header for RST_STREAM frame
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], 0x4) // Length (4 bytes)
	frameHeader[3] = 0x3                             // Type: RST_STREAM (0x3)
	frameHeader[4] = 0x0                             // Flags
	binary.BigEndian.PutUint32(frameHeader[5:], 0x1) // Stream ID: 1

	// Error Code (4 bytes)
	errorCode := make([]byte, 4)
	binary.BigEndian.PutUint32(errorCode, 0x0) // NO_ERROR (0x0)

	conn.Write(frameHeader)
	conn.Write(errorCode)
	log.Println("Sent RST_STREAM frame")
}

func sendSettingsFrame(conn net.Conn) {
	// Frame Header: Length (3 bytes), Type (1 byte), Flags (1 byte), Stream Identifier (4 bytes)
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], 0x0) // Length (0 bytes)
	frameHeader[3] = 0x4                             // Type: SETTINGS (0x4)
	frameHeader[4] = 0x0                             // Flags
	binary.BigEndian.PutUint32(frameHeader[5:], 0x0) // Stream ID

	conn.Write(frameHeader)
	log.Println("Sent SETTINGS frame")
}

func WriteSettingsFrame(w io.Writer, flags byte, payload []byte) error {
	length := len(payload)
	// Frame Header: Length (3 bytes), Type (1 byte), Flags (1 byte), Stream Identifier (4 bytes)
	var buf [http2frameHeaderLen]byte
	buf[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	buf[1] = byte((length >> 8) & 0xFF)
	buf[2] = byte(length & 0xFF)
	buf[3] = 0x4                             // Type: SETTINGS (0x4)
	buf[4] = flags                           // Flags
	binary.BigEndian.PutUint32(buf[5:], 0x0) // Stream ID

	if n, err := w.Write(buf[:]); err != nil {
		return err
	} else {
		log.Printf("Write settings header success, length = %d\n", n)
	}
	if len(payload) > 0 {
		if n, err := w.Write(payload); err != nil {
			return err
		} else {
			log.Printf("Write settings payload success, length = %d\n", n)
		}
	}
	return nil
}

func sendPushPromiseFrame(conn net.Conn) {
	promisedStreamID := 2
	headersPayload := []byte{
		0x88, // :status 200 OK (pre-encoded with HPACK)
	}
	length := 4 + len(headersPayload) // 4 bytes for promised stream ID + headers
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], uint32(length))
	frameHeader[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	frameHeader[1] = byte((length >> 8) & 0xFF)
	frameHeader[2] = byte(length & 0xFF)
	frameHeader[3] = 0x5                             // Type: PUSH_PROMISE (0x5)
	frameHeader[4] = 0x4                             // Flags: END_HEADERS (0x4)
	binary.BigEndian.PutUint32(frameHeader[5:], 0x1) // Stream ID: 1

	promisedStreamIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(promisedStreamIDBytes, uint32(promisedStreamID)&0x7FFFFFFF)
	conn.Write(frameHeader)
	conn.Write(promisedStreamIDBytes)
	conn.Write(headersPayload)
	fmt.Println("Sent PUSH_PROMISE frame")
}

func WritePingFrame(w io.Writer, payload []byte) error {
	length := len(payload)
	if length != 8 {
		return fmt.Errorf("ping payload should be 8")
	}
	var buf [http2frameHeaderLen]byte
	buf[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	buf[1] = byte((length >> 8) & 0xFF)
	buf[2] = byte(length & 0xFF)
	buf[3] = 0x6                             // Type: PING (0x6)
	buf[4] = 0x0                             // Flags
	binary.BigEndian.PutUint32(buf[5:], 0x0) // Stream ID

	if n, err := w.Write(buf[:]); err != nil {
		return err
	} else {
		log.Printf("Write ping header success, length = %d\n", n)
	}
	if n, err := w.Write(payload[:]); err != nil {
		return err
	} else {
		log.Printf("Write ping payload success, length = %d\n", n)
	}

	return nil
}

func sendPingFrame(conn net.Conn) {
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], 0x8) // Length (8 bytes)
	frameHeader[3] = 0x6                             // Type: PING (0x6)
	frameHeader[4] = 0x0                             // Flags
	binary.BigEndian.PutUint32(frameHeader[5:], 0x0) // Stream ID

	pingPayload := []byte{0, 0, 0, 0, 0, 0, 0, 1} // Ping opaque data
	conn.Write(frameHeader)
	conn.Write(pingPayload)
	log.Println("Sent PING frame")
}

func sendGoAwayFrame(conn net.Conn) {
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], 0x8) // Length (8 bytes)
	frameHeader[3] = 0x7                             // Type: GOAWAY (0x7)
	frameHeader[4] = 0x0                             // Flags
	binary.BigEndian.PutUint32(frameHeader[5:], 0x0) // Stream ID

	lastStreamID := make([]byte, 4)
	binary.BigEndian.PutUint32(lastStreamID, 0x0)
	errorCode := make([]byte, 4)
	binary.BigEndian.PutUint32(errorCode, 0x0) // NO_ERROR (0x0)

	conn.Write(frameHeader)
	conn.Write(lastStreamID)
	conn.Write(errorCode)
	log.Println("Sent GOAWAY frame")
}

func sendWindowUpdateFrame(conn net.Conn) {
	frameHeader := make([]byte, 9)
	binary.BigEndian.PutUint32(frameHeader[:4], 0x4) // Length (4 bytes)
	frameHeader[3] = 0x8                             // Type: WINDOW_UPDATE (0x8)
	frameHeader[4] = 0x0                             // Flags
	binary.BigEndian.PutUint32(frameHeader[5:], 0x0) // Stream ID

	windowSizeIncrement := make([]byte, 4)
	binary.BigEndian.PutUint32(windowSizeIncrement, 0x4000) // 16,384 bytes
	conn.Write(frameHeader)
	conn.Write(windowSizeIncrement)
	log.Println("Sent WINDOW_UPDATE frame")
}

func sendContinuationFrame(conn net.Conn) {
	frameHeader := make([]byte, 9)
	continuationPayload := []byte{0x00} // Continuation payload example
	length := len(continuationPayload)
	binary.BigEndian.PutUint32(frameHeader[:4], uint32(length))
	frameHeader[0] = byte((length >> 16) & 0xFF) // Length (3 bytes)
	frameHeader[1] = byte((length >> 8) & 0xFF)
	frameHeader[2] = byte(length & 0xFF)
	frameHeader[3] = 0x9                             // Type: CONTINUATION (0x9)
	frameHeader[4] = 0x4                             // Flags: END_HEADERS (0x4)
	binary.BigEndian.PutUint32(frameHeader[5:], 0x1) // Stream ID: 1

	conn.Write(frameHeader)
	conn.Write(continuationPayload)
	fmt.Println("Sent CONTINUATION frame")
}
