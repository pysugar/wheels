package main

import (
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()
	log.Println("Connected to server")

	for {
		if er := conn.WriteMessage(websocket.TextMessage, []byte("Hello, WebSocket!")); er != nil {
			log.Printf("Write error: %v", er)
			break
		}

		_, message, er := conn.ReadMessage()
		if er != nil {
			log.Printf("Read error: %v", er)
			break
		}
		log.Printf("Received: %s", message)

		time.Sleep(5 * time.Second)
	}
}
