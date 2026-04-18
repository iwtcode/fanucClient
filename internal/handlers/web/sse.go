package web

import (
	"fmt"
	"net/http"
	"sync"
)

// SSEBroker управляет подключениями браузеров для Server-Sent Events
type SSEBroker struct {
	clients map[int64]map[chan []byte]bool // userID -> set of channels
	mu      sync.RWMutex
}

func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients: make(map[int64]map[chan []byte]bool),
	}
}

func (b *SSEBroker) AddClient(userID int64, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.clients[userID] == nil {
		b.clients[userID] = make(map[chan []byte]bool)
	}
	b.clients[userID][ch] = true
}

func (b *SSEBroker) RemoveClient(userID int64, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.clients[userID] != nil {
		delete(b.clients[userID], ch)
		close(ch)
		if len(b.clients[userID]) == 0 {
			delete(b.clients, userID)
		}
	}
}

func (b *SSEBroker) BroadcastToUser(userID int64, eventType string, payload []byte) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	channels, ok := b.clients[userID]
	if !ok {
		return
	}

	// Формируем событие SSE
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, payload))

	for ch := range channels {
		// Неблокирующая отправка
		select {
		case ch <- msg:
		default:
			// Канал переполнен, пропускаем
		}
	}
}

func (b *SSEBroker) HandleStream(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := make(chan []byte, 10)
	b.AddClient(userID, ch)
	defer b.RemoveClient(userID, ch)

	// Отправляем начальное событие для подтверждения соединения
	fmt.Fprintf(w, "event: connected\ndata: {\"status\": \"ok\"}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			w.Write(msg)
			flusher.Flush()
		}
	}
}
