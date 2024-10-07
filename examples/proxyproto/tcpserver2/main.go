package main

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/pires/go-proxyproto"
)

func main() {
	// 创建原始的 TCP 监听器
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("无法监听端口 8080：%v", err)
	}
	defer listener.Close()

	// 使用 go-proxyproto 包装监听器
	proxyListener := &proxyproto.Listener{
		Listener: listener,
	}
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

	// 获取客户端的真实地址
	clientAddr := conn.RemoteAddr().String()
	fmt.Printf("收到来自 %s 的连接\n", clientAddr)

	// 读取客户端发送的数据
	reader := bufio.NewReader(conn)
	message, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("读取数据失败：%v", err)
		return
	}

	fmt.Printf("收到消息：%s", message)

	// 回应客户端
	_, err = conn.Write([]byte("消息已收到\n"))
	if err != nil {
		log.Printf("发送数据失败：%v", err)
	}
}
