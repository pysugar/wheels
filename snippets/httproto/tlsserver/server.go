package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pysugar/wheels/http/extensions"
	"github.com/pysugar/wheels/snippets/httproto/grpc"
	"github.com/pysugar/wheels/snippets/httproto/ws"
	"google.golang.org/grpc/grpclog"
)

// GODEBUG=http2debug=2 go run server.go
func main() {
	os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "DEBUG")
	os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "DEBUG")
	os.Setenv("GRPC_TRACE", "all")
	os.Setenv("GRPC_VERBOSITY", "DEBUG")

	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
	grpclog.SetLoggerV2(logger)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			ws.SimpleEchoHandler(w, r)
		} else {
			http.Error(w, "Unsupported upgrade protocol", http.StatusUpgradeRequired)
		}
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
			ws.GorillaEchoHandler(w, r)
		} else {
			http.Error(w, "Unsupported upgrade protocol", http.StatusUpgradeRequired)
		}
	})

	echoHandler := grpc.NewEchoHandler()
	mux.HandleFunc("/grpc/", func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			originalPath := r.URL.Path
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/grpc")
			r.RequestURI = r.URL.RequestURI()
			log.Printf("Adjusted path: %s -> %s\n", originalPath, r.URL.Path)
			echoHandler.ServeHTTP(w, r)
		} else {
			extensions.DebugHandler(w, r)
		}
	})

	server := &http.Server{
		Addr:    ":8443",
		Handler: mux,
	}

	log.Println("server listen on :8443")
	if err := server.ListenAndServeTLS("server.crt", "server.key"); err != nil {
		log.Fatalf("start server failure: %v", err)
	}
}
