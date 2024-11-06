package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/pysugar/wheels/demos/protocols/demo1/codec"
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
		if length == 0 {
			fmt.Println("Invalid payload length:", length)
			return
		}

		// Read the payload
		payload := make([]byte, length)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			fmt.Println("Read Payload Error:", err)
			return
		}

		// Read the checksum
		checksumBytes := make([]byte, 2)
		_, err = io.ReadFull(reader, checksumBytes)
		if err != nil {
			fmt.Println("Read Checksum Error:", err)
			return
		}
		checksum := binary.BigEndian.Uint16(checksumBytes)

		// Combine header and payload for checksum verification
		data := append(header, payload...)

		// Decode the message
		msg, err := codec.Decode(append(data, checksumBytes...))
		if err != nil {
			fmt.Println("Decode Message Error:", err)
			return
		}

		if msg.Checksum != checksum {
			fmt.Println("Checksum Error")
			return
		}

		fmt.Printf("Received Message: Type=%d, ID=%d, Payload=%s\n", msg.Type, msg.MsgID, string(msg.Payload))

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
