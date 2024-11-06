package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/pysugar/wheels/demos/protocols/demo1/codec"
	"io"
	"net"
	"os"
)

// Define server address
const (
	ServerAddress = "localhost:9000"
)

func main() {
	conn, err := net.Dial("tcp", ServerAddress)
	if err != nil {
		fmt.Println("Connection Error:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to server:", ServerAddress)

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

			if msg.Type == codec.MSG_TYPE_ACK {
				fmt.Printf("Received ACK: ID=%d, Payload=%s\n", msg.MsgID, string(msg.Payload))
			} else {
				fmt.Printf("Received Message: Type=%d, ID=%d, Payload=%s\n", msg.Type, msg.MsgID, string(msg.Payload))
			}
		}
	}()

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
		return
	}
	_, err = conn.Write(encodedAuth)
	if err != nil {
		fmt.Println("Write Auth Error:", err)
		return
	}
	fmt.Println("Sent Auth Message")

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
		fmt.Println("Sent Text Message")
		msgID++
	}

	fmt.Println("Client exiting")
}
