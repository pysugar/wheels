package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/pysugar/wheels/http/extensions"
	"github.com/pysugar/wheels/snippets/httproto/http2"
	"github.com/pysugar/wheels/snippets/httproto/tls10"
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
			tls10.TLS10Handler(func(w http.ResponseWriter, r *http.Request) {
				log.Printf("[tls/1.0] %s %s %s", r.RemoteAddr, r.Method, r.URL)
				fmt.Fprintln(w, "Has already upgrade to tls/1.0")
			}).ServeHTTP(w, r)
		} else {
			fmt.Fprintln(w, "Welcome to upgrade service")
		}
	}
	handler := extensions.LoggingMiddleware(http.HandlerFunc(handlerFunc))

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	log.Println("Server started，listen port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server start failure：%v", err)
	}
}
