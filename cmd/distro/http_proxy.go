package distro

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

var httpProxyCmd = &cobra.Command{
	Use:   `httpproxy -p 8080`,
	Short: "Start a Transparent HTTP Proxy",
	Long: `
Start a Transparent HTTP Proxy.

Start a Transparent HTTP Proxy: netool httpproxy --port=8080
`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")

		RunHTTPProxy(port)
	},
}

func init() {
	httpProxyCmd.Flags().IntP("port", "p", 8080, "http proxy	 port")
}

func RunHTTPProxy(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Error starting listener: %v\n", err)
	}

	log.Println("Starting HTTP transparent proxy on ", lis.Addr())
	for {
		clientConn, er := lis.Accept()
		if er != nil {
			log.Printf("Failed to accept connection: %v\n", er)
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
		log.Printf("Failed to read client request: %v", err)
		return
	}

	if request.Method == http.MethodConnect {
		handleConnectMethod(clientConn, request)
	} else {
		handleHTTPRequest(clientConn, request)
	}
}

func handleConnectMethod(clientConn net.Conn, request *http.Request) {
	targetHost := request.Host
	if !strings.Contains(targetHost, ":") {
		if request.URL.Scheme == "https" {
			targetHost = fmt.Sprintf("%s:443", targetHost)
		} else {
			targetHost = fmt.Sprintf("%s:80", targetHost)
		}
	}

	targetConn, err := net.DialTimeout("tcp", targetHost, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to target: %v\n", err)
		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
		fmt.Fprintf(clientConn, "HTTP/1.1 503 Service Unavailable"+errorHeaders+err.Error())
		return
	}
	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	defer targetConn.Close()

	log.Printf("Tunnel: %s -> (%s-%s) -> %s", clientConn.RemoteAddr(), clientConn.LocalAddr(),
		targetConn.LocalAddr(), targetConn.RemoteAddr())

	go func() {
		if _, er := io.Copy(targetConn, clientConn); er != nil {
			log.Printf("Error copying from client to target: %v\n", er)
		}
	}()

	_, err = io.Copy(clientConn, targetConn)
	if err != nil {
		log.Printf("Error copying from target to client: %v\n", err)
	}
}

func handleHTTPRequest(clientConn net.Conn, request *http.Request) {
	targetHost := request.Host
	if !strings.Contains(targetHost, ":") {
		if request.URL.Scheme == "https" {
			targetHost = fmt.Sprintf("%s:443", targetHost)
		} else {
			targetHost = fmt.Sprintf("%s:80", targetHost)
		}
	}

	targetConn, err := net.DialTimeout("tcp", targetHost, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to target: %v\n", err)
		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
		fmt.Fprintf(clientConn, "HTTP/1.1 "+"503 Service Unavailable"+errorHeaders+err.Error())
		return
	}
	defer targetConn.Close()

	remoteAddr := clientConn.RemoteAddr()
	log.Printf("RemoteAddr by proxyproto: %s\n", remoteAddr)

	// 获取原始客户端 IP 地址
	clientIP, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		log.Printf("Failed to parse client IP from remoteAddr: %v", err)
		clientIP = "Unknown"
	}

	// 添加 X-Forwarded-For 头部
	if prior, ok := request.Header["X-Forwarded-For"]; ok {
		clientIP = strings.Join(prior, ", ") + ", " + clientIP
	}
	request.Header.Set("X-Forwarded-For", clientIP)

	err = request.Write(targetConn)
	if err != nil {
		log.Printf("Failed to forward request to target: %v\n", err)
		const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"
		fmt.Fprintf(clientConn, "HTTP/1.1 "+"503 Service Unavailable"+errorHeaders+err.Error())
		return
	}

	go func() {
		if _, er := io.Copy(targetConn, clientConn); er != nil {
			log.Printf("Error copying from client to target: %v\n", err)
		}
	}()

	if _, er := io.Copy(clientConn, targetConn); er != nil {
		log.Printf("Error copying from target to client: %v\n", err)
	}
}
