package main

import (
	"fmt"
	"github.com/pires/go-proxyproto"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("无法监听端口 8080：%v", err)
	}
	defer listener.Close()

	proxyListener := &proxyproto.Listener{
		Listener:          listener,
		ReadHeaderTimeout: 3 * time.Second,
		ValidateHeader: func(header *proxyproto.Header) error {
			log.Printf("validate header: %v", header)
			return nil
		},
		Policy: func(upstream net.Addr) (proxyproto.Policy, error) {
			log.Printf("policy: %v", upstream)
			return proxyproto.USE, nil
		},
	}
	defer proxyListener.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		fmt.Fprintf(w, "客户端 IP：%s\n", clientIP)
	})

	log.Println("HTTP 服务器正在监听端口 8080...")
	if err := http.Serve(proxyListener, nil); err != nil {
		log.Fatalf("HTTP 服务器错误：%v", err)
	}
}
