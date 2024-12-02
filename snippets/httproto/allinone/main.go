package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pysugar/wheels/http/extensions"
	"github.com/pysugar/wheels/snippets/httproto/grpc"
	"github.com/pysugar/wheels/snippets/httproto/http2"
	"github.com/pysugar/wheels/snippets/httproto/ws"
)

// export GODEBUG=http2debug=1
func main() {
	mux := http.NewServeMux()

	// netool fetch --verbose http://127.0.0.1:8080/health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	// netool fetch --gorilla --verbose http://127.0.0.1:8080/ws
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			ws.GorillaEchoHandler(w, r)
		} else {
			http.Error(w, "Unsupported upgrade protocol", http.StatusUpgradeRequired)
		}
	})

	h2cHandler := http2.SimpleH2cHandler(extensions.DebugHandler)
	// netool fetch --upgrade --http2 --verbose http://127.0.0.1:8080/h2c
	// netool fetch --upgrade --http2 --method=POST --verbose http://127.0.0.1:8080/h2c
	mux.HandleFunc("/h2c", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "h2c" {
			h2cHandler.ServeHTTP(w, r)
		} else {
			http.Error(w, "Unsupported upgrade protocol", http.StatusUpgradeRequired)
		}
	})

	// netool fetch --http2 --upgrade --verbose http://127.0.0.1:8080/http2
	mux.HandleFunc("/http2", func(w http.ResponseWriter, r *http.Request) {
		http2.NewH2CHandler(extensions.DebugHandler).ServeHTTP(w, r)
	})

	echoHandler := grpc.NewEchoHandler()
	// netool fetch --grpc --upgrade --verbose http://localhost:8080/grpc/proto.EchoService/Echo --proto-path=../grpc/proto/echo.proto -d'{"message": "netool"}'
	mux.HandleFunc("/grpc/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received gRPC request: %s %s %s\n", r.Method, r.URL.Path, r.Proto)
		for k, v := range r.Header {
			log.Printf("Header[%q] = %q\n", k, v)
		}

		originalPath := r.URL.Path
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/grpc")
		r.RequestURI = r.URL.RequestURI()
		log.Printf("Adjusted path: %s -> %s\n", originalPath, r.URL.Path)

		echoHandler.ServeHTTP(w, r)
	})
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
