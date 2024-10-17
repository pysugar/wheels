package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Upgrade error: %v", err)
			return
		}
		defer conn.Close()
		log.Println("Client connected")

		for {
			messageType, message, er := conn.ReadMessage()
			if er != nil {
				log.Printf("Read error: %v", er)
				break
			}
			log.Printf("Received: %s", message)

			if er := conn.WriteMessage(messageType, message); er != nil {
				log.Printf("Write error: %v", er)
				break
			}
		}
	})

	log.Println("Server is listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}
