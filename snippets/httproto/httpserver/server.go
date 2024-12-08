package main

import (
	"fmt"
	"github.com/pysugar/wheels/snippets/httproto/sse"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pysugar/wheels/http/extensions"
	"github.com/pysugar/wheels/snippets/httproto/h2c"
	"github.com/pysugar/wheels/snippets/httproto/ws"
)

// export GODEBUG=http2debug=2
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
	// netool fetch --upgrade--verbose http://127.0.0.1:8080/h2c
	// netool fetch --upgrade --http2 --verbose http://127.0.0.1:8080/h2c
	// netool fetch --upgrade --http2 --method=POST --verbose http://127.0.0.1:8080/h2c
	h2cHandler := h2c.SimpleH2cHandler(extensions.DebugHandler)
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
	h2Handler := extensions.LoggingMiddleware(h2c.NewH2CHandler(extensions.DebugHandler))
	mux.HandleFunc("/http2", func(w http.ResponseWriter, r *http.Request) {
		h2Handler.ServeHTTP(w, r)
	})

	mux.HandleFunc("/sse", sse.SSEHandler)

	// netool fetch --verbose http://127.0.0.1:8080/metrics
	// netool fetch --http1 --verbose http://127.0.0.1:8080/metrics
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server started，listen port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server start failure：%v", err)
	}
}
