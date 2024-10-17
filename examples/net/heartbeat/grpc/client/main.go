package main

import (
	"fmt"
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

	// 启动心跳协程
	go startHeartbeat(conn)

	buf := make([]byte, 1024)
	for {
		n, er := conn.Read(buf)
		if er != nil {
			fmt.Printf("Read error: %v\n", er)
			continue
		}
		message := string(buf[:n])
		if message == "PING" {
			// 回复 Pong
			_, err := conn.Write([]byte("PONG"))
			if err != nil {
				fmt.Printf("Failed to send PONG: %v\n", err)
				return
			}
			fmt.Println("Received PING, sent PONG")
		} else {
			// 处理其他消息
			fmt.Printf("Received message: %s\n", message)
		}
	}
}

func startHeartbeat(conn net.Conn) {
	ticker := time.NewTicker(10 * time.Second)
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
