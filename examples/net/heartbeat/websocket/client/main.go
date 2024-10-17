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

	conn.SetPongHandler(func(appData string) error {
		log.Printf("Received Pong from server: %s\n", appData)
		return nil
	})

	conn.SetPingHandler(func(appData string) error {
		log.Printf("Received Ping from server: %s\n", appData)
		if er := conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second)); er != nil {
			log.Printf("Failed to send Pong: %v", er)
			return er
		}
		log.Println("Pong sent to server")
		return nil
	})

	go startHeartbeat(conn)

	for {
		messageType, message, er := conn.ReadMessage()
		if er != nil {
			log.Printf("Read error: %v", er)
			break
		}
		log.Printf("Received message: %s:%d\n", message, messageType)
	}
}

func startHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			if err != nil {
				log.Printf("Failed to send Ping: %v", err)
				return
			}
			log.Println("Ping sent to server")
		}
	}
}
