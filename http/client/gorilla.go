package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pysugar/wheels/http/extensions"
)

func (f *fetcher) doGorilla(ctx context.Context, req *http.Request) error {
	logger := newVerboseLogger(ctx)
	var serverAddr string
	var dialer *websocket.Dialer
	if strings.EqualFold(req.URL.Scheme, "https") {
		serverAddr = fmt.Sprintf("wss://%s%s", req.URL.Host, req.URL.RequestURI())
		if InsecureFromContext(ctx) {
			dialer = &websocket.Dialer{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
			logger.Printf("[%s] insecure skip verify", serverAddr)
		} else {
			dialer = websocket.DefaultDialer
		}
	} else {
		serverAddr = fmt.Sprintf("ws://%s%s", req.URL.Host, req.URL.RequestURI())
		dialer = websocket.DefaultDialer
	}

	conn, resp, err := dialer.DialContext(ctx, serverAddr, req.Header)
	if err != nil {
		logger.Printf("[%s] websocket dial error: %v", serverAddr, err)
		return err
	}
	defer conn.Close()

	logger.Println("Success to connect to WebSocket Server")
	if resp != nil {
		defer resp.Body.Close()
		logger.Printf("Response: %s", extensions.FormatResponse(resp))
	}

	gCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	messageChannel := make(chan string, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go f.wsGorillaLoop(gCtx, conn, messageChannel, &wg)
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Please input message: ")
		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)
		if msg == ":quit" || msg == ":exit" {
			cancel()
		}
		if er := conn.WriteMessage(websocket.TextMessage, []byte(msg)); er != nil {
			log.Printf("send message failure: %v", er)
			return er
		}

		select {
		case <-gCtx.Done():
			wg.Wait()
			log.Printf("websocket exit success")
			return nil
		case m, ok := <-messageChannel:
			if !ok {
				return nil
			}
			fmt.Printf("receive message: %s\n", m)
		case <-time.After(1 * time.Second):
			fmt.Println("receive message timeout")
		}
	}
}

func (f *fetcher) wsGorillaLoop(ctx context.Context, conn *websocket.Conn, ch chan<- string, wg *sync.WaitGroup) {
	defer close(ch)
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Printf("read loop context done")
			return
		default:
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("read message(%d) failure: %v", mt, err)
				return
			}
			ch <- string(msg)
		}
	}
}
