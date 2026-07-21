package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Event struct {
	Name string          `json:"event"`
	Data json.RawMessage `json:"data"`
}

type client struct {
	sendChan chan Event
}

// Broadcaster fans out events to every connected SSE client. Access to the
// client set is guarded by a mutex since ServeHTTP (register/unregister) and
// Broadcast (read) run on different goroutines per request.
type Broadcaster struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[*client]struct{}),
	}
}

func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// SSE connections stay open for as long as the browser tab is open —
	// the server's global WriteTimeout (set for regular request/response
	// endpoints) would otherwise kill this connection after a fixed
	// duration. Disabling the write deadline here, for this connection
	// only, is scoped entirely to the SSE stream.
	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{})
	}

	setSSEHeaders(w)

	c := &client{sendChan: make(chan Event, 50)}
	b.register(c)
	defer b.unregister(c)

	b.streamLoop(r, w, flusher, c)
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
}

func (b *Broadcaster) register(c *client) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[c] = struct{}{}
}

func (b *Broadcaster) unregister(c *client) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, c)
	close(c.sendChan)
}

func (b *Broadcaster) streamLoop(r *http.Request, w http.ResponseWriter, flusher http.Flusher, c *client) {
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if !writeHeartbeat(w, flusher) {
				return
			}
		case event, open := <-c.sendChan:
			if !open {
				return
			}
			if !writeEvent(w, flusher, event) {
				return
			}
		}
	}
}

func writeHeartbeat(w http.ResponseWriter, flusher http.Flusher) bool {
	if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

func writeEvent(w http.ResponseWriter, flusher http.Flusher, event Event) bool {
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Name, event.Data); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

// Broadcast delivers an event to every connected client. If a client's
// buffer is full (a slow consumer), the oldest queued event is dropped to
// make room — this keeps a stalled browser tab from ever blocking the
// collection loop.
func (b *Broadcaster) Broadcast(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for c := range b.clients {
		select {
		case c.sendChan <- event:
		default:
			select {
			case <-c.sendChan:
			default:
			}
			select {
			case c.sendChan <- event:
			default:
			}
		}
	}
}
