package extensions

import (
	"crypto/tls"
	"log"
	"net/http/httptrace"
	"net/textproto"
)

func NewDebugClientTrace(prefix string) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			log.Printf("[%s] GetConn hostPort: %s\n", prefix, hostPort)
		},
		GotConn: func(info httptrace.GotConnInfo) {
			log.Printf("[%s] GotConn info: %+v\n", prefix, info)
		},
		PutIdleConn: func(err error) {
			log.Printf("[%s] PutIdleConn err: %v\n", prefix, err)
		},
		GotFirstResponseByte: func() {
			log.Printf("[%s] GotFirstResponseByte\n", prefix)
		},
		Got100Continue: func() {
			log.Printf("[%s] Got100Continue\n", prefix)
		},
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			log.Printf("[%s] Got1xxResponse: %d, header: %+v\n", prefix, code, header)
			return nil
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			log.Printf("[%s] DNSStart: %+v\n", prefix, info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			log.Printf("[%s] DNSDone: %+v\n", prefix, info)
		},
		ConnectStart: func(network, addr string) {
			log.Printf("[%s] ConnectStart: %s:%s\n", prefix, network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			log.Printf("[%s] ConnectDone: %s:%s err: %+v\n", prefix, network, addr, err)
		},
		TLSHandshakeStart: func() {
			log.Printf("[%s] TLSHandshakeStart\n", prefix)
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			log.Printf("[%s] TLSHandshakeDone: %+v, err: %v\n", prefix, state, err)
		},
		WroteHeaderField: func(key string, value []string) {
			log.Printf("[%s] WroteHeaderField: %s, %v\n", prefix, key, value)
		},
		WroteHeaders: func() {
			log.Printf("[%s] WroteHeaders\n", prefix)
		},
		Wait100Continue: func() {
			log.Printf("[%s] Wait100Continue\n", prefix)
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			log.Printf("[%s] WroteRequest: %+v\n", prefix, info)
		},
	}
}
