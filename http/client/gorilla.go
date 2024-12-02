package client

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pysugar/wheels/http/extensions"
)

func (f *fetcher) doGorilla(ctx context.Context, req *http.Request) error {
	var serverAddr string
	if strings.EqualFold(req.URL.Scheme, "https") {
		serverAddr = fmt.Sprintf("wss://%s%s", req.URL.Host, req.URL.RequestURI())
	} else {
		serverAddr = fmt.Sprintf("ws://%s%s", req.URL.Host, req.URL.RequestURI())
	}

	logger := newVerboseLogger(ctx)
	conn, resp, err := websocket.DefaultDialer.Dial(serverAddr, req.Header)
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

	messageChannel := make(chan string, 10)
	go f.wsGorillaLoop(ctx, conn, messageChannel)
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Please input message: ")
		msg, _ := reader.ReadString('\n')
		if er := conn.WriteMessage(websocket.TextMessage, []byte(msg)); er != nil {
			log.Printf("send message failure: %v", er)
			return er
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
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

func (f *fetcher) wsGorillaLoop(ctx context.Context, conn *websocket.Conn, ch chan<- string) {
	defer close(ch)
	for {
		select {
		case <-ctx.Done():
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
