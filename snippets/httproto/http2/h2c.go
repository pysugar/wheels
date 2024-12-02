package http2

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

func SimpleH2cHandler(h http.HandlerFunc) http.HandlerFunc {
	h2s := &http2.Server{}
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := h2cUpgrade(w, r)
		if err != nil {
			log.Println("HTTP/2 Upgrade Failure", err)
			http.Error(w, "HTTP/2 Upgrade Failure", http.StatusInternalServerError)
			return
		}

		go h2s.ServeConn(conn, &http2.ServeConnOpts{
			BaseConfig: &http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					h(w, r)
				}),
			},
		})
	}
}

func h2cUpgrade(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("h2c upgrade server unsupport hijacker")
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("h2c upgrade conn hijack failure: %v", err)
	}
	defer rw.Flush()

	//reader := bufio.NewReader(conn)
	//preface := make([]byte, 24)
	//_, err = reader.Read(preface)
	//if err != nil {
	//	return nil, fmt.Errorf("read client preface: %v", err)
	//}
	//if string(preface) != "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n" {
	//	return nil, fmt.Errorf("client preface invalid: %v", preface)
	//}

	response := "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: h2c\r\n\r\n"
	if _, er := rw.WriteString(response); er != nil {
		return nil, fmt.Errorf("send h2c upgrade response failure: %v", er)
	}
	if er := rw.Flush(); er != nil {
		return nil, fmt.Errorf("h2c upgrade flush buffer failure：%v", er)
	}

	return conn, nil
}
