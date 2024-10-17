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

	// 用于接收从连接读取的消息
	messageChan := make(chan string)
	// 用于心跳响应的通知
	heartbeatResponseChan := make(chan bool)

	// 启动读取协程
	go reader(conn, messageChan, heartbeatResponseChan)

	// 启动心跳协程
	go startHeartbeat(conn, heartbeatResponseChan)

	// 主循环处理消息
	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				fmt.Println("Connection closed")
				return
			}
			if message == "PING" {
				// 回复 PONG
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
}

func reader(conn net.Conn, messageChan chan<- string, heartbeatResponseChan chan<- bool) {
	defer func() {
		close(messageChan)
		close(heartbeatResponseChan)
	}()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			return
		}
		message := string(buf[:n])
		if message == "PONG" {
			// 心跳响应
			heartbeatResponseChan <- true
		} else {
			// 其他消息
			messageChan <- message
		}
	}
}

func startHeartbeat(conn net.Conn, heartbeatResponseChan <-chan bool) {
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
			fmt.Println("Sent heartbeat PING")

			// 等待回应，设置超时时间
			select {
			case <-heartbeatResponseChan:
				fmt.Println("Heartbeat successful")
			case <-time.After(5 * time.Second):
				fmt.Println("Heartbeat timeout")
				return
			}
		}
	}
}
