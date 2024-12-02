package ws

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func GorillaEchoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket Upgrade failure: %v", err)
		return
	}
	defer conn.Close()

	gorillaEchoHandler(r.Context(), conn)
}

func gorillaEchoHandler(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()

loop:
	for {
		select {
		case <-ctx.Done():
			return
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Receive message failure: %v", err)
				break loop
			}
			log.Printf("Receive message: %s", message)
			if er := conn.WriteMessage(messageType, message); er != nil {
				log.Printf("Send message failure: %v", er)
				break loop
			}
		}
	}
}
