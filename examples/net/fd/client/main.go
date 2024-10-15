package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(network, addr, 10*time.Second)
			if err != nil {
				return nil, err
			}

			if tcpConn, ok := conn.(*net.TCPConn); ok {
				//rawConn, err := tcpConn.SyscallConn()
				//if err != nil {
				//	return nil, err
				//}
				//
				//rawConn.Control(func(fd uintptr) {
				//	fmt.Printf("Client socket file descriptor: %d\n", fd)
				//})

				file, er := tcpConn.File()
				if er != nil {
					fmt.Printf("Failed to get file from TCP connection: %v\n", er)
				}
				fd := file.Fd()
				fmt.Printf("listener fd-%d: %s\n", fd, file.Name())
			}
			return conn, nil
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGQUIT, syscall.SIGSTOP)
	go func() {
		sig := <-sigCh
		fmt.Printf("\nReceived signal: %v. Exiting...\n", sig)
		cancel()
	}()

loop:
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down gracefully...")
			break loop
		default:
			resp, err := client.Get("http://localhost:8765")
			if err != nil {
				fmt.Printf("Failed to make request: %v\n", err)
				break loop
			}
			defer resp.Body.Close()
			fmt.Printf("Response status: %s\n", resp.Status)
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Println("client exit....")
}
