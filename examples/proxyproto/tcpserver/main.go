package main

import (
	"fmt"
	"log"
	"net"

	"github.com/pires/go-proxyproto"
)

func main() {
	listener, err := net.Listen("tcp", ":9876")
	if err != nil {
		log.Fatalf("无法监听端口 8080：%v", err)
	}
	defer listener.Close()

	proxyListener := &proxyproto.Listener{Listener: listener}
	defer proxyListener.Close()

	log.Println("服务器正在监听端口 8080...")

	for {
		conn, err := proxyListener.Accept()
		if err != nil {
			log.Printf("接受连接失败：%v", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	localAddr := conn.LocalAddr().String()
	fmt.Printf("local addr: %s\n", localAddr)

	remoteAddr := conn.RemoteAddr().String()
	fmt.Printf("remote addr: %s\n", remoteAddr)
}
