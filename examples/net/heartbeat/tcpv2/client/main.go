package main

import (
	"errors"
	"fmt"
	"github.com/pysugar/wheels/examples/net/heartbeat/tcpv2/ldm"
	"io"
	"net"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:50051")
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to server")

	messageChan := make(chan string)
	heartbeatResponseChan := make(chan bool)
	heartbeatDone := make(chan struct{})

	go reader(conn, messageChan, heartbeatResponseChan)

	go func() {
		startHeartbeat(conn, heartbeatResponseChan)
		close(heartbeatDone)
	}()

	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				fmt.Println("Message channel closed")
				return
			}
			if message == "PING" {
				if er := ldm.SendMessage(conn, "PONG"); er != nil {
					fmt.Printf("Failed to send PONG: %v\n", er)
					return
				}
				fmt.Println("Received PING, sent PONG")
			} else {
				fmt.Printf("Received message: %s\n", message)
			}
		case <-heartbeatDone:
			fmt.Println("Heartbeat failed, exiting main loop")
			return
		}
	}
}

func reader(conn net.Conn, messageChan chan<- string, heartbeatResponseChan chan<- bool) {
	defer func() {
		close(messageChan)
		close(heartbeatResponseChan)
	}()

	for {
		message, err := ldm.ReceiveMessage(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("Connection closed by remote")
				break
			} else if nerr, ok := err.(net.Error); ok {
				if nerr.Temporary() {
					fmt.Printf("Temporary read error: %v\n", err)
					continue // 临时错误，继续读取
				} else if nerr.Timeout() {
					fmt.Printf("Read timeout: %v\n", err)
					time.Sleep(100 * time.Millisecond)
					continue
				}
				fmt.Printf("Unexpect network error: %v", err)
				break
			} else {
				fmt.Printf("Read error: %v\n", err)
				break
			}
		}

		if message == "PONG" {
			heartbeatResponseChan <- true
		} else {
			messageChan <- message
		}
	}
}

func startHeartbeat(conn net.Conn, heartbeatResponseChan <-chan bool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ldm.SendMessage(conn, "PING"); err != nil {
				fmt.Printf("Failed to send heartbeat: %v\n", err)
				return
			}
			fmt.Println("Sent heartbeat PING")

			select {
			case <-heartbeatResponseChan:
				fmt.Println("Heartbeat successful - Received PONG")
			case <-time.After(5 * time.Second):
				fmt.Println("Heartbeat timeout")
				return
			}
		}
	}
}
