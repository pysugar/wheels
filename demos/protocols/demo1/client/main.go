package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/pysugar/wheels/demos/protocols/demo1/codec"
	"hash/crc32"
	"io"
	"net"
	"os"
	"time"
)

// SendHeartbeat sends a heartbeat message periodically
func SendHeartbeat(ctx context.Context, conn net.Conn, msgID *uint32) {
	ticker := time.NewTicker(30 * time.Second) // Send heartbeat every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			heartbeatMsg := &codec.Message{
				Version: 1,
				Type:    codec.MSG_TYPE_HEARTBEAT,
				MsgID:   *msgID,
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
			*msgID++
		case <-ctx.Done():
			fmt.Println("Stopping Heartbeat")
			return
		}
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("Connection Error:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to server: localhost:9000")

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a goroutine to receive messages from the server
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
				cancel() // Signal to stop heartbeat
				return
			}

			// Parse Length field (payload length)
			length := binary.BigEndian.Uint16(header[2:4])

			// Read the payload
			payload := make([]byte, length)
			_, err = io.ReadFull(reader, payload)
			if err != nil {
				fmt.Println("Read Payload Error:", err)
				cancel()
				return
			}

			// Read the checksum
			checksumBytes := make([]byte, 4)
			_, err = io.ReadFull(reader, checksumBytes)
			if err != nil {
				fmt.Println("Read Checksum Error:", err)
				cancel()
				return
			}
			checksum := binary.BigEndian.Uint32(checksumBytes)

			// Combine header and payload for checksum verification
			data := append(header, payload...)
			calculatedChecksum := crc32.ChecksumIEEE(data)
			if calculatedChecksum != checksum {
				fmt.Printf("Invalid checksum: expected %d, got %d\n", checksum, calculatedChecksum)
				cancel()
				return
			}

			// Decode the message
			msg, err := codec.Decode(append(data, checksumBytes...))
			if err != nil {
				fmt.Println("Decode Message Error:", err)
				cancel()
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

	// Initialize message ID counter
	var msgID uint32 = 2

	// Start sending heartbeat messages
	go SendHeartbeat(ctx, conn, &msgID)

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
		cancel()
		return
	}
	_, err = conn.Write(encodedAuth)
	if err != nil {
		fmt.Println("Write Auth Error:", err)
		cancel()
		return
	}
	fmt.Println("Sent AUTH Message")

	// Read user input and send text messages
	scanner := bufio.NewScanner(os.Stdin)
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

	// Signal heartbeat to stop and wait briefly to ensure goroutines exit
	cancel()
	time.Sleep(1 * time.Second)
	fmt.Println("Client exiting")
}
