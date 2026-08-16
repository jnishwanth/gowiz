package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// EventType represents the category of server event being broadcast.
type EventType string

const (
	EventDeviceUpdated   EventType = "device_updated"
	EventCommandExecuted EventType = "command_executed"
	EventPresetApplied   EventType = "preset_applied"
	EventCircadianTick   EventType = "circadian_tick"
	EventWebhookTest     EventType = "webhook_test"
)

// Event defines the structured payload broadcast to SSE subscribers and webhooks.
type Event struct {
	Type      EventType `json:"type"`
	Timestamp string    `json:"timestamp"`
	IP        string    `json:"ip,omitempty"`
	Room      string    `json:"room,omitempty"`
	Command   string    `json:"command,omitempty"`
	Status    string    `json:"status,omitempty"`
	Payload   any       `json:"payload,omitempty"`
}

// Broadcaster manages active SSE client subscribers and webhook event dispatches.
type Broadcaster struct {
	mu                 sync.RWMutex
	subscribers        map[chan Event]struct{}
	webhookURL         string
	httpClient         *http.Client
	webhookSentCount   atomic.Uint64
	webhookFailedCount atomic.Uint64
}

// NewBroadcaster initializes a Broadcaster with an optional Webhook URL target.
func NewBroadcaster(webhookURL string) *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan Event]struct{}),
		webhookURL:  webhookURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// SetWebhookURL dynamically updates or clears the target webhook destination URL.
func (b *Broadcaster) SetWebhookURL(url string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.webhookURL = url
}

// WebhookURL returns the current configured target webhook destination URL.
func (b *Broadcaster) WebhookURL() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.webhookURL
}

// Stats returns counters for active subscribers and webhook delivery metrics.
func (b *Broadcaster) Stats() (subscribers int, webhookSent uint64, webhookFailed uint64) {
	b.mu.RLock()
	subscribers = len(b.subscribers)
	b.mu.RUnlock()
	return subscribers, b.webhookSentCount.Load(), b.webhookFailedCount.Load()
}

// Subscribe registers a new event channel and returns an unsubscribe function.
func (b *Broadcaster) Subscribe() (chan Event, func()) {
	ch := make(chan Event, 32)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		if _, exists := b.subscribers[ch]; exists {
			delete(b.subscribers, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
	return ch, unsubscribe
}

// Publish broadcasts an Event to all connected SSE subscriber channels and webhooks.
func (b *Broadcaster) Publish(evt Event) {
	if evt.Timestamp == "" {
		evt.Timestamp = time.Now().Format(time.RFC3339)
	}

	b.mu.RLock()
	webhookTarget := b.webhookURL
	// Dispatch event to SSE subscribers with non-blocking sends
	for ch := range b.subscribers {
		select {
		case ch <- evt:
		default:
			// Subscriber buffer full; skip to prevent blocking execution
		}
	}
	b.mu.RUnlock()

	// Dispatch event asynchronously to Webhook URL target if configured
	if webhookTarget != "" {
		go b.dispatchWebhook(webhookTarget, evt)
	}
}

func (b *Broadcaster) dispatchWebhook(url string, evt Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		b.webhookFailedCount.Add(1)
		return
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		b.webhookFailedCount.Add(1)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gowiz-server/1.0.0")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		b.webhookFailedCount.Add(1)
		return
	}
	_ = resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		b.webhookSentCount.Add(1)
	} else {
		b.webhookFailedCount.Add(1)
	}
}

// ServeHTTP handles Server-Sent Events (SSE) requests on GET /api/v1/events.
func (b *Broadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = fmt.Fprintln(w, `{"status":"error","error":"method not allowed"}`)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, `{"status":"error","error":"streaming unsupported"}`)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, unsubscribe := b.Subscribe()
	defer unsubscribe()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, string(data))
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
