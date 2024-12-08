package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pysugar/wheels/http/extensions"
	"github.com/pysugar/wheels/snippets/httproto/grpc"
	"github.com/pysugar/wheels/snippets/httproto/sse"
	"github.com/pysugar/wheels/snippets/httproto/ws"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// export GODEBUG=http2debug=1
// GODEBUG=http2debug=2 go run server.go
func main() {
	mux := http.NewServeMux()

	// netool fetch --verbose http://127.0.0.1:8080/health
	// netool fetch --http1 --verbose http://127.0.0.1:8080/health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	// x/net/websocket not support gorilla client
	// netool fetch --websocket --verbose http://127.0.0.1:8080/websocket
	mux.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			ws.SimpleEchoHandler(w, r)
		} else {
			http.Error(w, "Unsupported upgrade protocol", http.StatusUpgradeRequired)
		}
	})

	// netool fetch --websocket --verbose http://127.0.0.1:8080/ws
	// netool fetch --ws --verbose http://127.0.0.1:8080/ws
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			ws.GorillaEchoHandler(w, r)
		} else {
			http.Error(w, "Unsupported upgrade protocol", http.StatusUpgradeRequired)
		}
	})

	// netool fetch --verbose http://127.0.0.1:8080/h2c
	// netool fetch --http1 --verbose http://127.0.0.1:8080/h2c
	// netool fetch --http2 --verbose http://127.0.0.1:8080/h2c
	// netool fetch --http2 --verbose --method=POST  http://127.0.0.1:8080/h2c
	// netool fetch --upgrade --verbose http://127.0.0.1:8080/h2c
	// netool fetch --upgrade --method=POST --verbose http://127.0.0.1:8080/h2c
	h2cHandler := http.HandlerFunc(extensions.DebugHandler)
	mux.HandleFunc("/h2c", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "h2c" {
			h2cHandler.ServeHTTP(w, r)
		} else {
			http.Error(w, "Unsupported upgrade protocol", http.StatusUpgradeRequired)
		}
	})

	// netool fetch --verbose http://127.0.0.1:8080/http2
	// netool fetch --http1 --verbose http://127.0.0.1:8080/http2
	// netool fetch --http2 --verbose http://127.0.0.1:8080/http2
	// netool fetch --upgrade --verbose http://127.0.0.1:8080/http2
	http2Handler := http.HandlerFunc(extensions.DebugHandler)
	mux.HandleFunc("/http2", func(w http.ResponseWriter, r *http.Request) {
		http2Handler.ServeHTTP(w, r)
	})

	// netool fetch --grpc --verbose http://localhost:8080/grpc/proto.EchoService/Echo --proto-path=../grpc/proto/echo.proto -d'{"message": "netool"}'
	echoHandler := grpc.NewEchoHandler()
	mux.HandleFunc("/grpc/", func(w http.ResponseWriter, r *http.Request) {
		originalPath := r.URL.Path
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/grpc")
		r.RequestURI = r.URL.RequestURI()
		log.Printf("Adjusted path: %s -> %s\n", originalPath, r.URL.Path)

		echoHandler.ServeHTTP(w, r)
	})

	// netool fetch --verbose http://127.0.0.1:8080/metrics
	// netool fetch --http1 --verbose http://127.0.0.1:8080/metrics
	// netool fetch --http2 --verbose http://127.0.0.1:8080/metrics
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/sse", sse.SSEHandler)

	h2s := &http2.Server{}
	handler := h2c.NewHandler(extensions.LoggingMiddleware(mux), h2s)
	// handler := h2c.NewHandler(mux, h2s)
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	log.Println("Server started，listen port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server start failure：%v", err)
	}
}
