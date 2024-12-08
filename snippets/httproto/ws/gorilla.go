package ws

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = time.Second
)

type (
	message struct {
		messageType int
		data        []byte
	}
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func GorillaEchoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, http.Header{"Server-sid": {"gorilla-echo"}})
	if err != nil {
		log.Printf("WebSocket Upgrade failure: %v", err)
		return
	}
	defer conn.Close()

	setHandlers(r.Context(), conn)

	msgCh := make(chan *message)
	go readLoop(r.Context(), conn, msgCh)

	gorillaEchoHandler(r.Context(), conn, msgCh)
}

func readLoop(ctx context.Context, conn *websocket.Conn, msgCh chan<- *message) {
	defer close(msgCh)
loop:
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				var closeErr *websocket.CloseError
				if errors.As(err, &closeErr) {
					log.Printf("Receive client close: %d (%s)", closeErr.Code, closeErr.Text)
				} else if errors.Is(err, io.EOF) {
					log.Printf("Client close connection, err: %v", err)
				} else {
					log.Printf("Receive message failure：%v", err)
				}
				break loop
			}
			log.Printf("Receive message: (%d)%s", msgType, msg)
			msgCh <- &message{msgType, msg}
		}
	}
}

func gorillaEchoHandler(ctx context.Context, conn *websocket.Conn, msgCh <-chan *message) {
	defer conn.Close()
	timer := time.NewTicker(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			if er := conn.WriteMessage(msg.messageType, msg.data); er != nil {
				log.Printf("Send echo message failure: %v", er)
				return
			}
			log.Printf("Send echo message success.")
		case <-timer.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				// if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				log.Printf("Send ping message failure: %v", err)
				return
			}
			log.Printf("Send ping message success.")
		}
	}
}

func setHandlers(ctx context.Context, conn *websocket.Conn) {
	conn.SetPingHandler(func(message string) error {
		log.Printf("Received Ping from client: %s, send pong\n", message)
		err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(writeWait))
		if errors.Is(err, websocket.ErrCloseSent) {
			return nil
		} else if e, ok := err.(net.Error); ok && e.Temporary() {
			return nil
		}
		return err
	})
	conn.SetPongHandler(func(appData string) error {
		log.Printf("Received Pong from client: %s\n", appData)
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("Received Close from client: %s\n", text)
		message := websocket.FormatCloseMessage(code, "")
		conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(writeWait))
		return nil
	})
}
