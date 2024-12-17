package ws

//import (
//	"context"
//	"crypto/tls"
//	"fmt"
//	"github.com/gorilla/websocket"
//	//"github.com/pysugar/wheels/http/extensions"
//	"log"
//	"net/http"
//	"strings"
//	"sync"
//)

//type wsClient struct {
//	domain string
//	useSSL bool
//	conn   *websocket.Conn
//	done   chan struct{}
//	rw     sync.RWMutex
//}
//
//func (wc *wsClient) connect() {
//
//}

//
//func dialContext(ctx context.Context, req *http.Request) (*websocket.Conn, error) {
//	var serverAddr string
//	var dialer *websocket.Dialer
//	if strings.EqualFold(req.URL.Scheme, "https") {
//		serverAddr = fmt.Sprintf("wss://%s%s", req.URL.Host, req.URL.RequestURI())
//		if InsecureFromContext(ctx) {
//			dialer = &websocket.Dialer{
//				TLSClientConfig: &tls.Config{
//					InsecureSkipVerify: true,
//				},
//			}
//			log.Printf("[%s] insecure skip verify\n", serverAddr)
//		} else {
//			dialer = websocket.DefaultDialer
//		}
//	} else {
//		serverAddr = fmt.Sprintf("ws://%s%s", req.URL.Host, req.URL.RequestURI())
//		dialer = websocket.DefaultDialer
//	}
//
//	conn, resp, err := dialer.DialContext(ctx, serverAddr, req.Header)
//	if err != nil {
//		log.Printf("[%s] websocket dial error: %v\n", serverAddr, err)
//		return err
//	}
//	defer conn.Close()
//
//	log.Println("Success to connect to WebSocket Server")
//	if resp != nil {
//		defer resp.Body.Close()
//		log.Printf("Response: %s", extensions.FormatResponse(resp))
//	}
//}
