package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"clipsync/internal/store"
	"clipsync/pkg/proto"
)

// Hub is the in-process pub/sub broker. It relies on a Store to provide
// cross-instance pub/sub or persistence when configured.
type Hub struct {
	mu       sync.RWMutex
	conns    map[string]*Client                 // clientID -> client
	channels map[string]map[string]*Client      // channel -> clientID -> client
	store    store.Store
}

// NewHub creates a Hub with the provided Store backend.
func NewHub(s store.Store) *Hub {
	return &Hub{
		conns:    make(map[string]*Client),
		channels: make(map[string]map[string]*Client),
		store:    s,
	}
}

// RegisterClient adds a client to the hub (called after successful auth).
func (h *Hub) RegisterClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.ID == "" {
		log.Printf("attempted to register client with empty ID")
		return
	}
	h.conns[c.ID] = c
}

// UnregisterClient removes client from hub and unsubscribes from channels.
func (h *Hub) UnregisterClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, c.ID)
	for ch := range c.subs {
		if subs, ok := h.channels[ch]; ok {
			delete(subs, c.ID)
			if len(subs) == 0 {
				delete(h.channels, ch)
			}
		}
	}
}

// Subscribe registers the client to a named channel.
func (h *Hub) Subscribe(clientID, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.conns[clientID]
	if !ok {
		return
	}
	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[string]*Client)
	}
	h.channels[channel][clientID] = c
	c.subs[channel] = struct{}{}
}

// Unsubscribe removes a client's subscription from a channel.
func (h *Hub) Unsubscribe(clientID, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.conns[clientID]
	if !ok {
		return
	}
	if subs, ok := h.channels[channel]; ok {
		delete(subs, clientID)
		if len(subs) == 0 {
			delete(h.channels, channel)
		}
	}
	delete(c.subs, channel)
}

// Publish delivers msg to all subscribers of channel. excludeClientID, if non-empty,
// will not receive the message (useful to avoid echo).
func (h *Hub) Publish(channel string, rawMsg []byte, excludeClientID string) {
	// first publish to store/backplane if configured
	if h.store != nil {
		// best-effort: ignore store errors here
		_ = h.store.Publish(channel, rawMsg)
		_ = h.store.SaveHistory(channel, rawMsg)
	}

	// deliver to local subscribers
	h.mu.RLock()
	subs := h.channels[channel]
	h.mu.RUnlock()
	if subs == nil {
		return
	}
	for id, c := range subs {
		if id == excludeClientID {
			continue
		}
		select {
		case c.send <- append([]byte(nil), rawMsg...):
		default:
			// subscriber congested: drop and optionally unsubscribe
			log.Printf("subscriber %s channel %s send buffer full; dropping message", id, channel)
		}
	}
}

// DirectSend attempts to deliver a raw message to a specific client by id.
// Returns true if delivered (queued), false otherwise.
func (h *Hub) DirectSend(targetClientID string, rawMsg []byte) bool {
	h.mu.RLock()
	c, ok := h.conns[targetClientID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case c.send <- append([]byte(nil), rawMsg...):
		return true
	default:
		return false
	}
}
