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

	"golang.org/x/net/websocket"
)

func (f *fetcher) doWebsocket(ctx context.Context, req *http.Request) error {
	origin := req.Header.Get("Origin")
	if origin == "" {
		origin = fmt.Sprintf("%s://%s", req.URL.Scheme, req.URL.Host)
	}

	var serverAddr string
	if strings.EqualFold(req.URL.Scheme, "https") {
		serverAddr = fmt.Sprintf("wss://%s%s", req.URL.Host, req.URL.RequestURI())
	} else {
		serverAddr = fmt.Sprintf("ws://%s%s", req.URL.Host, req.URL.RequestURI())
	}

	conn, err := websocket.Dial(serverAddr, "", origin)
	if err != nil {
		return err
	}
	defer conn.Close()

	messageChannel := make(chan string, 10)
	go f.wsReadLoop(ctx, conn, messageChannel)
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Please input message: ")
		msg, _ := reader.ReadString('\n')
		if er := websocket.Message.Send(conn, msg); er != nil {
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

func (f *fetcher) wsReadLoop(ctx context.Context, ws *websocket.Conn, ch chan<- string) {
	defer close(ch)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg string
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				log.Printf("read message failure: %v", err)
				return
			}
			ch <- strings.TrimSpace(msg)
		}
	}
}
