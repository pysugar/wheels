package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("WebSocket server is listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer conn.Close()
	log.Println("Client connected")

	conn.SetPongHandler(func(appData string) error {
		log.Printf("Received Pong from client: %s\n", appData)
		return nil
	})
	go startHeartbeat(conn)

	for {
		messageType, message, er := conn.ReadMessage()

		if er != nil {
			log.Printf("Read error: %v", er)
			break
		}
		log.Printf("Received: %s, type: %d", message, messageType)

		if er := conn.WriteMessage(messageType, message); er != nil {
			log.Printf("Write error: %v", er)
			break
		}
	}
}

func startHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			if err != nil {
				log.Printf("Failed to send Ping: %v", err)
				return
			}
			log.Println("Ping sent to client")
		}
	}
}
