package extensions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

func DebugHandlerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Client-IP", getClientIP(r))
	w.Header().Set("Server-IP", getServerIP(r))

	var body string
	if r.Body != nil {
		defer r.Body.Close()
		reqBody := http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
		b, err := io.ReadAll(reqBody)
		if err != nil {
			body = fmt.Sprintf("Error reading body: %v", err)
		} else {
			body = string(b)
		}
	}

	response := map[string]interface{}{
		":method":   r.Method,
		":path":     r.RequestURI,
		":protocol": r.Proto,
		"headers":   r.Header,
		"body":      body,
		"client_ip": getClientIP(r),
		"server_ip": getServerIP(r),
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, "Error generating JSON", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(jsonData)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func DebugHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Client-IP", getClientIP(r))
	w.Header().Set("Server-IP", getServerIP(r))

	bw := bufio.NewWriter(w)
	defer func() {
		if err := bw.Flush(); err != nil {
			log.Printf("Error flushing buffer: %v", err)
		}
	}()

	if n, err := fmt.Fprintf(bw, "%s %s %s\n", r.Method, r.RequestURI, r.Proto); err != nil {
		log.Printf("Error writing request line, size: %d, err: %v", n, err)
	}

	if err := r.Header.Write(bw); err != nil {
		log.Printf("Error writing headers: %v", err)
	}

	if _, err := fmt.Fprintln(bw); err != nil {
		log.Printf("Error writing newline: %v", err)
	}

	if r.Body != nil {
		defer r.Body.Close()
		reqBody := http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
		n, err := io.Copy(bw, reqBody)
		if err != nil {
			log.Printf("Error reading body, size: %d, err: %v", n, err)
		}
	}

	if err := r.Trailer.Write(bw); err != nil {
		log.Printf("Error writing trailer: %v", err)
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return strings.TrimSpace(xRealIP)
	}

	return strings.TrimSpace(r.RemoteAddr)
}

func getServerIP(r *http.Request) string {
	addr, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if ok {
		log.Printf("LocalAddr: %s:%s", addr.Network(), addr.String())
		return addr.String()
	}

	var err error
	host := r.Host
	if host == "" {
		host, err = os.Hostname()
		if err != nil {
			return ""
		}
	}

	addrs, err := net.LookupHost(host)
	if err != nil || len(addrs) == 0 {
		return "Unknown"
	}
	return addrs[0]
}
