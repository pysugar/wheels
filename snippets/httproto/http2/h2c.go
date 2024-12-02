package http2

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

func SimpleH2cHandler(h http.HandlerFunc) http.HandlerFunc {
	h2s := &http2.Server{}
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := h2cUpgrade(w, r, h2s, h)
		if err != nil {
			log.Println("HTTP/2 Upgrade Failure", err)
			http.Error(w, "HTTP/2 Upgrade Failure", http.StatusInternalServerError)
			return
		}

		fmt.Println(FormatRequest(r))

		go h2s.ServeConn(conn, &http2.ServeConnOpts{
			BaseConfig: &http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprintln(w, "已升级到 HTTP/2")
					fmt.Println(FormatRequest(r))
				}),
			},
		})
		// defer conn.Close()
	}
}

func h2cUpgrade(w http.ResponseWriter, r *http.Request, h2s *http2.Server, h http.HandlerFunc) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("h2c upgrade server unsupport hijacker")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("h2c upgrade conn hijack failure: %v", err)
	}
	defer rw.Flush()

	response := "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: h2c\r\n\r\n"
	if _, er := rw.WriteString(response); er != nil {
		return nil, fmt.Errorf("send h2c upgrade response failure: %v", er)
	}
	if er := rw.Flush(); er != nil {
		return nil, fmt.Errorf("h2c upgrade flush buffer failure：%v", er)
	}

	//
	//h2s.ServeConn(conn, &http2.ServeConnOpts{
	//	BaseConfig: &http.Server{
	//		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	//			h(w, r)
	//		}),
	//	},
	//})
	return conn, nil
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
