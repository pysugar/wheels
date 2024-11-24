package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pysugar/wheels/http/extensions"
)

func main() {
	debugHandler := http.HandlerFunc(extensions.DebugHandler)
	debugHandlerJSON := http.HandlerFunc(extensions.DebugHandlerJSON)

	http.Handle("/", extensions.CORSMiddleware(extensions.LoggingMiddleware(debugHandler)))
	http.Handle("/json", extensions.CORSMiddleware(extensions.LoggingMiddleware(debugHandlerJSON)))

	addr := ":8080"
	fmt.Printf("Starting debug server at http://localhost%s\n", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
