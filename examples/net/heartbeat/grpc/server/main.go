package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server is listening...")

	for {
		conn, er := listener.Accept()
		if er != nil {
			fmt.Printf("Failed to accept: %v\n", er)
			continue
		}
		fmt.Println("Accepted new connection")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	go startHeartbeat(conn)

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			return
		}
		message := string(buf[:n])
		if message == "PING" {
			if _, er := conn.Write([]byte("PONG")); er != nil {
				fmt.Printf("Failed to send PONG: %v\n", er)
				return
			}
			fmt.Println("Received PING, sent PONG")
		} else {
			fmt.Printf("Received message: %s\n", message)
		}
	}
}

func startHeartbeat(conn net.Conn) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 发送心跳
			_, err := conn.Write([]byte("PING"))
			if err != nil {
				fmt.Printf("Failed to send heartbeat: %v\n", err)
				return
			}
			// 等待回应
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			buf := make([]byte, 4)
			_, err = conn.Read(buf)
			if err != nil {
				fmt.Printf("Heartbeat timeout: %v\n", err)
				return
			}
			if string(buf) != "PONG" {
				fmt.Println("Invalid heartbeat response")
				return
			}
			conn.SetReadDeadline(time.Time{})
			fmt.Println("Heartbeat successful")
		}
	}
}
