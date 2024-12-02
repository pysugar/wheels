package http2

import (
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// todo there is a bug
func NewH2CHandler(h http.HandlerFunc) http.Handler {
	var h2s = &http2.Server{}
	return h2c.NewHandler(h, h2s)
}
