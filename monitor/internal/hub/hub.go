// Package hub menyiarkan event ke semua browser lewat Server-Sent Events (SSE).
package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func New() *Hub {
	return &Hub{clients: make(map[chan []byte]struct{})}
}

// Broadcast tidak pernah memblokir: client yang lambat cukup kehilangan event.
func (h *Hub) Broadcast(event string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("hub marshal: %v", err)
		return
	}
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data))
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []byte, 32)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
	}()

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-ch:
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
