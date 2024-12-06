package h2c

import (
	"log"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func NewH2CHandler(h http.HandlerFunc) http.HandlerFunc {
	handler := h2c.NewHandler(h, &http2.Server{})
	log.Printf("New H2C Handler: %T", handler)
	return func(w http.ResponseWriter, req *http.Request) {
		handler.ServeHTTP(w, req)
	}
}
