package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8765")
	if err != nil {
		fmt.Printf("Error starting listener: %v\n", err)
		return
	}
	defer listener.Close()

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		log.Fatalf("Not a TCP listener")
	}

	file, err := tcpListener.File()
	if err != nil {
		log.Fatalf("Failed to get file from listener: %v\n", err)
	}
	defer file.Close()

	fd := file.Fd()
	fmt.Printf("listener fd-%d: %s\n", fd, file.Name())

	fmt.Println("Listening on ", listener.Addr())

	for {
		conn, er := listener.Accept()
		if er != nil {
			fmt.Printf("Failed to accept connection: %v\n", er)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		fmt.Println("Not a TCP connection")
		return
	}
	//defer tcpConn.Close()
	defer tcpConn.CloseRead()
	defer tcpConn.CloseWrite()

	file, err := tcpConn.File()
	if err != nil {
		fmt.Printf("Failed to get file from TCP connection: %v\n", err)
		return
	}
	defer file.Close()

	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Connection: close\r\n" +
		"\r\n" +
		"Hello, World!\n"
	_, err = file.Write([]byte(response))
	if err != nil {
		fmt.Printf("Failed to write response: %v\n", err)
		return
	}

	fd := file.Fd()
	fmt.Printf("fd-%d: %s\n", fd, file.Name())
}
