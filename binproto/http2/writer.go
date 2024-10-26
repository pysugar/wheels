package http2

import (
	"encoding/binary"
	"log"
	"net"
)

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

func sendDataFrame(conn net.Conn) {
	data := []byte("Hello, HTTP/2 cleartext")
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
