package main

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/pires/go-proxyproto"
)

func main() {
	// 连接到服务器
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatalf("无法连接到服务器：%v", err)
	}
	defer conn.Close()

	// 构建 PROXY 协议头
	header := &proxyproto.Header{
		Version:           1,
		Command:           proxyproto.PROXY,
		TransportProtocol: proxyproto.TCPv4,
		SourceAddr:        &net.TCPAddr{IP: net.ParseIP("10.1.1.1"), Port: 12345},
		DestinationAddr:   conn.RemoteAddr().(*net.TCPAddr),
	}

	// 发送 PROXY 协议头
	_, err = header.WriteTo(conn)
	if err != nil {
		log.Fatalf("发送 PROXY 协议头失败：%v", err)
	}

	// 发送实际数据
	_, err = conn.Write([]byte("Hello, Server!\n"))
	if err != nil {
		log.Fatalf("发送数据失败：%v", err)
	}

	// 读取服务器响应
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("读取服务器响应失败：%v", err)
	}

	fmt.Printf("服务器响应：%s", response)
}
