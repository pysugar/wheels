package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/pysugar/wheels/demos/protocols/demo1/codec"
	"io"
	"net"
	"os"
	"time"
)

// Define server address
const (
	ServerAddress = "localhost:9000"
)

// SendHeartbeat sends a heartbeat message periodically
func SendHeartbeat(conn net.Conn, stopChan chan struct{}) {
	ticker := time.NewTicker(30 * time.Second) // Send heartbeat every 30 seconds
	defer ticker.Stop()
	msgID := uint32(100) // Starting MsgID for heartbeat

	for {
		select {
		case <-ticker.C:
			heartbeatMsg := &codec.Message{
				Version: 1,
				Type:    codec.MSG_TYPE_HEARTBEAT,
				MsgID:   msgID,
				Payload: []byte{},
			}
			encodedHeartbeat, err := heartbeatMsg.Encode()
			if err != nil {
				fmt.Println("Encode Heartbeat Error:", err)
				continue
			}
			_, err = conn.Write(encodedHeartbeat)
			if err != nil {
				fmt.Println("Write Heartbeat Error:", err)
				return
			}
			fmt.Println("Sent HEARTBEAT Message")
			msgID++
		case <-stopChan:
			fmt.Println("Stopping Heartbeat")
			return
		}
	}
}

func main() {
	conn, err := net.Dial("tcp", ServerAddress)
	if err != nil {
		fmt.Println("Connection Error:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to server:", ServerAddress)

	// Start a goroutine to receive messages from the server
	stopHeartbeat := make(chan struct{})
	go func() {
		reader := bufio.NewReader(conn)
		for {
			// Read the first 8 bytes (header)
			header := make([]byte, 8)
			_, err := io.ReadFull(reader, header)
			if err != nil {
				if err != io.EOF {
					fmt.Println("Read Header Error:", err)
				}
				close(stopHeartbeat) // Stop heartbeat on connection close
				return
			}

			// Parse Length field (payload length)
			length := binary.BigEndian.Uint16(header[2:4])
			// No longer check for length == 0 as some message types may have zero-length payloads

			// Read the payload
			payload := make([]byte, length)
			_, err = io.ReadFull(reader, payload)
			if err != nil {
				fmt.Println("Read Payload Error:", err)
				close(stopHeartbeat)
				return
			}

			// Read the checksum
			checksumBytes := make([]byte, 2)
			_, err = io.ReadFull(reader, checksumBytes)
			if err != nil {
				fmt.Println("Read Checksum Error:", err)
				close(stopHeartbeat)
				return
			}
			checksum := binary.BigEndian.Uint16(checksumBytes)

			data := append(header, payload...)
			calculatedChecksum := codec.CalculateChecksum(data)
			if calculatedChecksum != checksum {
				fmt.Printf("Invalid checksum: expected %d, got %d\n", checksum, calculatedChecksum)
				close(stopHeartbeat)
				return
			}

			// Decode the message
			msg, err := codec.Decode(append(data, checksumBytes...))
			if err != nil {
				fmt.Println("Decode Message Error:", err)
				close(stopHeartbeat)
				return
			}

			// Handle different message types
			switch msg.Type {
			case codec.MSG_TYPE_ACK:
				fmt.Printf("Received ACK: ID=%d, Payload=%s\n", msg.MsgID, string(msg.Payload))
			default:
				fmt.Printf("Received Unknown Message Type: %d\n", msg.Type)
			}
		}
	}()

	// Start sending heartbeat messages
	go SendHeartbeat(conn, stopHeartbeat)

	// Send authentication message
	authPayload := []byte("user:password")
	authMsg := &codec.Message{
		Version: 1,
		Type:    codec.MSG_TYPE_AUTH,
		MsgID:   1,
		Payload: authPayload,
	}
	encodedAuth, err := authMsg.Encode()
	if err != nil {
		fmt.Println("Encode Auth Error:", err)
		close(stopHeartbeat)
		return
	}
	_, err = conn.Write(encodedAuth)
	if err != nil {
		fmt.Println("Write Auth Error:", err)
		close(stopHeartbeat)
		return
	}
	fmt.Println("Sent AUTH Message")

	// Read user input and send text messages
	scanner := bufio.NewScanner(os.Stdin)
	msgID := uint32(2)
	for {
		fmt.Print("Enter message (type 'exit' to quit): ")
		if !scanner.Scan() {
			break
		}
		text := scanner.Text()
		if text == "exit" {
			break
		}
		textMsg := &codec.Message{
			Version: 1,
			Type:    codec.MSG_TYPE_TEXT,
			MsgID:   msgID,
			Payload: []byte(text),
		}
		encodedText, err := textMsg.Encode()
		if err != nil {
			fmt.Println("Encode Text Error:", err)
			continue
		}
		_, err = conn.Write(encodedText)
		if err != nil {
			fmt.Println("Write Text Error:", err)
			break
		}
		fmt.Println("Sent TEXT Message")
		msgID++
	}

	// Close the heartbeat goroutine
	close(stopHeartbeat)
	fmt.Println("Client exiting")
}
