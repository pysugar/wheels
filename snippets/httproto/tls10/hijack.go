package tls10

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
)

func TLS10Handler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			http.Error(w, "Hijack failure", http.StatusInternalServerError)
			return
		}

		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS10,
		}

		tlsConn := tls.Server(conn, tlsConfig)
		if er := tlsConn.Handshake(); er != nil {
			log.Printf("TLS handshake failure: %v", er)
			return
		}

		err = http.Serve(&singleListener{tlsConn}, h)
		if err != nil {
			log.Printf("tls/1.0 Upgrade Failure: %v", err)
			http.Error(w, "tls/1.0 Upgrade Failure", http.StatusInternalServerError)
			return
		}
	}
}

type singleListener struct {
	net.Conn
}

func (sl *singleListener) Accept() (net.Conn, error) {
	if sl.Conn == nil {
		return nil, fmt.Errorf("conn is nil")
	}
	conn := sl.Conn
	sl.Conn = nil
	return conn, nil
}

func (sl *singleListener) Close() error {
	if sl.Conn != nil {
		return sl.Conn.Close()
	}
	return nil
}

func (sl *singleListener) Addr() net.Addr {
	return sl.Conn.LocalAddr()
}
