package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const serverVersion = "1.0.0"

func main() {
	http.HandleFunc("/", handleRequest)

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Starting HTTP server on ", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// 获取客户端 IP 地址
	clientIP := getClientIP(r)

	// 获取服务端 IP 地址
	serverIP, err := getServerIP(r)
	if err != nil {
		serverIP = "Unknown"
	}

	// 打印请求信息到控制台
	log.Printf("Received request from %s: %s %s", clientIP, r.Method, r.URL.Path)

	// 构建响应内容
	response := buildResponse(r, clientIP, serverIP)

	// 设置响应头并返回信息
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, response)
}

func getClientIP(r *http.Request) string {
	// 尝试从 X-Forwarded-For 获取客户端 IP 地址（适用于代理的场景）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// 尝试从 X-Real-IP 获取客户端 IP 地址（适用于一些代理的场景）
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return strings.TrimSpace(xRealIP)
	}

	// 使用 RemoteAddr 获取客户端 IP 地址
	ip, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "Unknown"
	}
	log.Printf("RemoteAddr: %s:%s\n", ip, port)

	return ip
}

func getServerIP(r *http.Request) (string, error) {
	// 从请求中获取 TCP 连接的本地地址
	addr, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if ok {
		log.Printf("LocalAddr: %s:%s", addr.Network(), addr.String())
	}

	// 获取本机的主机名
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}

	// 获取主机的地址列表
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}

	if len(addrs) > 0 {
		return addrs[0], nil
	}

	return "", fmt.Errorf("could not determine server IP")
}

func buildResponse(r *http.Request, clientIP, serverIP string) string {
	// 构建请求的详细信息
	var sb strings.Builder

	sb.WriteString("HTTP Request Information:\n")
	sb.WriteString(fmt.Sprintf("Client IP: %s\n", clientIP))
	sb.WriteString(fmt.Sprintf("Server IP: %s\n", serverIP))
	sb.WriteString(fmt.Sprintf("Server Version: %s\n", serverVersion))
	sb.WriteString(fmt.Sprintf("Request Method: %s\n", r.Method))
	sb.WriteString(fmt.Sprintf("Request URL: %s\n", r.URL.String()))
	sb.WriteString(fmt.Sprintf("HTTP Protocol: %s\n", r.Proto))
	sb.WriteString("Headers:\n")

	for name, values := range r.Header {
		for _, value := range values {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", name, value))
		}
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		sb.WriteString("\nBody:\n")
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		sb.WriteString(string(body))
	}

	return sb.String()
}
