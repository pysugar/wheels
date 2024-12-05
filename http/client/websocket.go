package client

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
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

	wsCfg, err := websocket.NewConfig(serverAddr, origin)
	if err != nil {
		return err
	}
	conn, err := wsCfg.DialContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	gCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messageChannel := make(chan string, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go f.wsReadLoop(gCtx, conn, messageChannel, &wg)
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Please input message: ")
		msg, _ := reader.ReadString('\n')
		msg = strings.TrimSpace(msg)

		if msg == ":quit" || msg == ":exit" {
			cancel()
		}
		if er := websocket.Message.Send(conn, msg); er != nil {
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
			log.Println("receive message timeout")
		}
	}
}

func (f *fetcher) wsReadLoop(ctx context.Context, ws *websocket.Conn, ch chan<- string, wg *sync.WaitGroup) {
	defer close(ch)
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Printf("read loop context done")
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
