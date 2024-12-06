package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/pysugar/wheels/http/extensions"
	"github.com/pysugar/wheels/snippets/httproto/h2c"
	"github.com/pysugar/wheels/snippets/httproto/tls10"
	"github.com/pysugar/wheels/snippets/httproto/ws"
)

// export GODEBUG=http2debug=1
// GODEBUG=http2debug=2 go run server.go
func main() {
	h2cHandler1 := h2c.SimpleH2cHandler(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[h2c] %s %s %s", r.RemoteAddr, r.Method, r.URL)
		fmt.Fprintln(w, "[1] Has already upgrade to HTTP/2")
	})
	h2cHandler2 := h2c.NewH2CHandler(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[h2c] %s %s %s", r.RemoteAddr, r.Method, r.URL)
		fmt.Fprintln(w, "[2] Has already upgrade to HTTP/2")
	})

	// ./netool fetch --verbose http://localhost:8080
	// ./netool fetch --http1 --verbose http://localhost:8080
	// ./netool fetch --upgrade --verbose http://localhost:8080
	// ./netool fetch --verbose http://localhost:8080?simple=1
	// ./netool fetch --http1 --verbose http://localhost:8080?simple=1
	// ./netool fetch --upgrade --verbose http://localhost:8080?simple=1
	// ./netool fetch --websocket --verbose http://localhost:8080
	// ./netool fetch --websocket --verbose http://localhost:8080?gorilla=1
	// ./netool fetch --gorilla --verbose http://localhost:8080?gorilla=1
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		upgradeHeader := r.Header.Get("Upgrade")
		if strings.Contains(strings.ToLower(upgradeHeader), "websocket") {
			if r.URL.Query().Has("gorilla") {
				ws.GorillaEchoHandler(w, r)
			} else {
				ws.SimpleEchoHandler(w, r)
			}
		} else if strings.Contains(strings.ToLower(upgradeHeader), "h2c") {
			if r.URL.Query().Has("simple") {
				h2cHandler1.ServeHTTP(w, r)
			} else {
				h2cHandler2.ServeHTTP(w, r)
			}
		} else if strings.Contains(strings.ToLower(upgradeHeader), "tls/1.0") {
			// deprecated
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
