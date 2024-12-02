package client

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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

	go f.wsReadLoop(ctx, conn)
	reader := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Print("Please input message: ")
			msg, _ := reader.ReadString('\n')
			if er := websocket.Message.Send(conn, msg); er != nil {
				log.Printf("send message failure: %v", er)
				return er
			}
		}
	}
}

func (f *fetcher) wsReadLoop(ctx context.Context, ws *websocket.Conn) {
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
			fmt.Printf("receive message: %s\n", msg)
		}
	}
}
