package sse

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func SSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	i := 1
	for {
		select {
		case <-r.Context().Done():
			log.Println("Client disconnected")
			return
		case t := <-ticker.C:
			fmt.Fprintf(w, "event: message\nid: %d\ndata: Time: %s\n\n", i, t.Format(time.RFC3339))
			flusher.Flush()
			i++
		}
	}
}
