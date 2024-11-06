package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/pysugar/wheels/demos/protocols/demo1/codec"
	"hash/crc32"
	"io"
	"net"
)

const (
	ServerPort = ":9000"
)

// Handle incoming connections
func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		// Read the first 8 bytes (header)
		header := make([]byte, 8)
		_, err := io.ReadFull(reader, header)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Read Header Error:", err)
			}
			return
		}

		// Parse Length field (payload length)
		length := binary.BigEndian.Uint16(header[2:4])

		// Read the payload
		payload := make([]byte, length)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			fmt.Println("Read Payload Error:", err)
			return
		}

		// Read the checksum
		checksumBytes := make([]byte, 4)
		_, err = io.ReadFull(reader, checksumBytes)
		if err != nil {
			fmt.Println("Read Checksum Error:", err)
			return
		}

		checksum := binary.BigEndian.Uint32(checksumBytes)
		data := append(header, payload...)
		calculatedChecksum := crc32.ChecksumIEEE(data)
		if calculatedChecksum != checksum {
			fmt.Println("Invalid checksum:", calculatedChecksum)
			return
		}

		data = append(data, checksumBytes...)

		// Decode the message
		msg, err := codec.Decode(data)
		if err != nil {
			fmt.Println("Decode Message Error:", err)
			return
		}

		// Handle different message types
		switch msg.Type {
		case codec.MSG_TYPE_AUTH:
			fmt.Printf("Received AUTH Message: ID=%d, Payload=%s\n", msg.MsgID, string(msg.Payload))
			// Here you can add authentication logic
			// For simplicity, assume authentication is always successful

		case codec.MSG_TYPE_HEARTBEAT:
			fmt.Printf("Received HEARTBEAT Message: ID=%d\n", msg.MsgID)
			// Handle heartbeat (e.g., update last active time)

		case codec.MSG_TYPE_TEXT:
			fmt.Printf("Received TEXT Message: ID=%d, Payload=%s\n", msg.MsgID, string(msg.Payload))
			// Here you can add logic to forward the message to other clients

		default:
			fmt.Printf("Received Unknown Message Type: %d\n", msg.Type)
		}

		// Send ACK message
		ackMsg := &codec.Message{
			Version: 1,
			Type:    codec.MSG_TYPE_ACK,
			MsgID:   msg.MsgID,
			Payload: []byte("ACK"),
		}
		encodedAck, err := ackMsg.Encode()
		if err != nil {
			fmt.Println("Encode ACK Error:", err)
			return
		}
		_, err = conn.Write(encodedAck)
		if err != nil {
			fmt.Println("Write ACK Error:", err)
			return
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", ServerPort)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server listening on", ServerPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept Error:", err)
			continue
		}
		fmt.Println("New connection established:", conn.RemoteAddr())
		go handleConnection(conn)
	}
}
