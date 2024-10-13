package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error starting listener: %v\n", err)
	}

	log.Println("Starting HTTP transparent proxy on :8080")
	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		go handleHTTPProxy(clientConn)
	}
}

func handleHTTPProxy(clientConn net.Conn) {
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to read client request: %v\n", err)
		return
	}

	targetHost := request.Host
	if !strings.Contains(targetHost, ":") {
		targetHost = fmt.Sprintf("%s:80", targetHost)
	}

	targetConn, err := net.DialTimeout("tcp", targetHost, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to target: %v\n", err)
		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
		fmt.Fprintf(clientConn, "HTTP/1.1 "+"503 Service Unavailable"+errorHeaders+err.Error())
		return
	}
	defer targetConn.Close()

	err = request.Write(targetConn)
	if err != nil {
		log.Printf("Failed to forward request to target: %v\n", err)
		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
		fmt.Fprintf(clientConn, "HTTP/1.1 "+"503 Service Unavailable"+errorHeaders+err.Error())
		return
	}

	go func() {
		err := copyTo(targetConn, reader)
		// _, err := io.Copy(targetConn, clientConn)
		// _, err := io.Copy(targetConn, reader)
		if err != nil {
			log.Printf("Error copying from client to target: %v\n", err)
		}
	}()

	// _, err = io.Copy(clientConn, targetConn)
	err = copyTo(clientConn, targetConn)
	if err != nil {
		log.Printf("Error copying from target to client: %v\n", err)
	}
}

func copyTo(w io.Writer, r io.Reader) error {
	buffer := make([]byte, 4096)
	for {
		n, err := r.Read(buffer)
		log.Printf("read buffer, n: %d, err: %v", n, err)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				log.Printf("Error writing to target: %v", writeErr)
				return writeErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			log.Printf("Error reading from source: %v", err)
			return err
		}
	}
}
