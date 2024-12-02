package main

import (
	"fmt"
	"github.com/pysugar/wheels/http/extensions"
	"log"
	"net/http"
	"strings"

	"github.com/pysugar/wheels/snippets/httproto/http2"
	"github.com/pysugar/wheels/snippets/httproto/ws"
)

func main() {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		upgradeHeader := r.Header.Get("Upgrade")
		if strings.Contains(strings.ToLower(upgradeHeader), "websocket") {
			ws.SimpleEchoHandler(w, r)
		} else if strings.Contains(strings.ToLower(upgradeHeader), "h2c") {
			http2.SimpleH2cHandler(func(w http.ResponseWriter, r *http.Request) {
				log.Printf("[h2c] %s %s %s", r.RemoteAddr, r.Method, r.URL)
				fmt.Fprintln(w, "Has already upgrade to HTTP/2")
			}).ServeHTTP(w, r)
		} else if strings.Contains(strings.ToLower(upgradeHeader), "tls/1.0") {
			// 处理 TLS 升级
			//conn, _, err := w.(http.Hijacker).Hijack()
			//if err != nil {
			//	http.Error(w, "连接劫持失败", http.StatusInternalServerError)
			//	return
			//}
			//tlsConfig := &tls.Config{
			//	MinVersion: tls.VersionTLS10,
			//}
			//tlsConn := tls.Server(conn, tlsConfig)
			//if err := tlsConn.Handshake(); err != nil {
			//	log.Println("TLS 握手失败：", err)
			//	return
			//}
			//http.Serve(tlsConn, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//	fmt.Fprintln(w, "已升级到 TLS")
			//}))
		} else {
			// 默认处理
			fmt.Fprintln(w, "Welcome to upgrade service")
		}
	}
	handler := extensions.LoggingMiddleware(http.HandlerFunc(handlerFunc))

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	log.Println("服务器启动，监听端口 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("服务器启动失败：%v", err)
	}
}

func handleUpgrade(w http.ResponseWriter, r *http.Request) {
	upgradeHeader := r.Header.Get("Upgrade")
	if strings.Contains(strings.ToLower(upgradeHeader), "websocket") {
		ws.SimpleEchoHandler(w, r)
	} else {

	}
}
