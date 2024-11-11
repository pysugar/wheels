package extensions

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("< Received HTTP Request: %s\n", FormatRequest(r))
		next.ServeHTTP(w, r)
		if r.Response != nil {
			log.Printf("Sending HTTP Response: %s\nCost: %v >\n", FormatResponse(r.Response), time.Since(start))
		} else {
			log.Printf("Sending HTTP Error: \n%s\nCost: %v >\n", FormatResponseWriter(w), time.Since(start))
		}
	})
}

func FormatRequest(r *http.Request) string {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	fmt.Fprintf(writer, "\n%s %s %s\r\n", r.Method, r.URL.RequestURI(), r.Proto)
	r.Header.Write(writer)

	if r.RemoteAddr != "" {
		fmt.Fprintf(writer, "Remote-Addr: %s\r\n", r.RemoteAddr)
	}

	if len(r.Trailer) > 0 {
		fmt.Fprintf(writer, "Trailer: ")
		first := true
		for name := range r.Trailer {
			if !first {
				fmt.Fprintf(writer, ", ")
			}
			fmt.Fprintf(writer, "%s", name)
			first = false
		}
		fmt.Fprintf(writer, "\r\n")
	}
	writer.Flush()
	return buf.String()
}

func FormatResponse(r *http.Response) string {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	fmt.Fprintf(writer, "\n%s %s\r\n", r.Proto, r.Status)

	r.Header.Write(writer)

	if len(r.Trailer) > 0 {
		fmt.Fprintf(writer, "Trailer: ")
		first := true
		for name := range r.Trailer {
			if !first {
				fmt.Fprintf(writer, ", ")
			}
			fmt.Fprintf(writer, "%s", name)
			first = false
		}
		fmt.Fprintf(writer, "\r\n")
	}
	fmt.Fprintf(writer, "\r\n")
	writer.Flush()
	return buf.String()
}

func FormatResponseWriter(w http.ResponseWriter) string {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	w.Header().Write(writer)

	writer.Flush()
	return buf.String()
}
