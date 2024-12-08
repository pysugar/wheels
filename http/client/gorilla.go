package client

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pysugar/wheels/http/extensions"
)

const (
	writeWait = time.Second
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

	f.setHandlers(ctx, conn)

	gCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	messageChannel := make(chan string, 10)
	go f.wsGorillaLoop(gCtx, conn, messageChannel, &wg)
	wg.Add(1)
	inputChannel := make(chan string)
	go f.wsInputLoop(gCtx, inputChannel, &wg)

	for {
		select {
		case <-gCtx.Done():
			wg.Wait()
			log.Printf("websocket exit success")
			return nil
		case m, ok := <-messageChannel:
			if !ok {
				return nil
			}
			fmt.Printf("receive message: %s\nPlease input message: ", m)
		case inputMsg, ok := <-inputChannel:
			if !ok {
				return nil
			}
			if inputMsg == ":quit" || inputMsg == ":exit" {
				conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, inputMsg), time.Now().Add(writeWait))
				cancel()
			} else {
				if er := conn.WriteMessage(websocket.TextMessage, []byte(inputMsg)); er != nil {
					log.Printf("send message failure: %v", er)
					return er
				}
			}
		case <-time.After(60 * time.Second):
			fmt.Println("receive message timeout")
		}
	}
}

func (f *fetcher) setHandlers(ctx context.Context, conn *websocket.Conn) {
	logger := newVerboseLogger(ctx)
	conn.SetPingHandler(func(message string) error {
		logger.Printf("Received Ping from server: %s, send pong\n", message)
		err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(writeWait))
		if errors.Is(err, websocket.ErrCloseSent) {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}
		return err
	})
	conn.SetPongHandler(func(appData string) error {
		logger.Printf("Received Pong from server: %s\n", appData)
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		logger.Printf("Received Close from server: %s\n", text)
		message := websocket.FormatCloseMessage(code, "")
		conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(writeWait))
		return nil
	})
}

func (f *fetcher) wsInputLoop(ctx context.Context, ch chan<- string, wg *sync.WaitGroup) {
	defer close(ch)
	defer wg.Done()
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			log.Printf("read loop context done")
			return
		default:
			msg, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("stdin read error: %v", err)
				return
			}
			msg = strings.TrimSpace(msg)
			ch <- msg
			if msg == ":quit" || msg == ":exit" {
				log.Printf("exit loop")
				return
			}
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
				var closeErr *websocket.CloseError
				if errors.As(err, &closeErr) {
					log.Printf("Receive server close: %d (%s)", closeErr.Code, closeErr.Text)
				} else if errors.Is(err, io.EOF) {
					log.Printf("Server close connection, err: %v", err)
				} else {
					log.Printf("Receive message failure：%v", err)
				}
				return
			}
			fmt.Printf("receive message: %d %s", mt, msg)
			ch <- string(msg)
		}
	}
}
