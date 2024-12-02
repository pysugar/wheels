package ws

import (
	"context"
	"io"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

func SimpleEchoHandler(w http.ResponseWriter, r *http.Request) {
	websocket.Handler(func(ws *websocket.Conn) {
		simpleEchoHandler(r.Context(), ws)
	}).ServeHTTP(w, r)
}

func simpleEchoHandler(ctx context.Context, ws *websocket.Conn) {
	defer ws.Close()

loop:
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg string
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				if err == io.EOF {
					log.Println("Client close connection")
				} else {
					log.Printf("receive message failure：%v", err)
				}
				break loop
			}

			log.Printf("received：%s", msg)

			if err := websocket.Message.Send(ws, msg); err != nil {
				log.Printf("send message failure：%v", err)
				break loop
			}
		}
	}
}
