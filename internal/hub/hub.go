package hub

import (
	"log"
	"sync"
	"time"

	"clipsync/internal/store"
)

// Hub is the in-process pub/sub broker. It relies on a Store to provide
// cross-instance pub/sub or persistence when configured.
type Hub struct {
	mu           sync.RWMutex
	conns        map[string]*Client            // clientID -> client
	channels     map[string]map[string]*Client // channel -> clientID -> client
	store        store.Store
	unsubscribes map[string]func() // channel -> unsubscribe func
	rl           *RateLimiter
}

// NewHub creates a Hub with the provided Store backend.
func NewHub(s store.Store) *Hub {
	return &Hub{
		conns:        make(map[string]*Client),
		channels:     make(map[string]map[string]*Client),
		store:        s,
		unsubscribes: make(map[string]func()),
		rl:           NewRateLimiter(100, time.Second),
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
				if unsub, ok := h.unsubscribes[ch]; ok {
					unsub()
					delete(h.unsubscribes, ch)
				}
			}
		}
	}
}

// Subscribe registers the client to a named channel.
func (h *Hub) Subscribe(clientID, channel string) {
	h.mu.Lock()
	c, ok := h.conns[clientID]
	if !ok {
		h.mu.Unlock()
		return
	}
	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[string]*Client)
		if h.store != nil {
			if _, exists := h.unsubscribes[channel]; !exists {
				unsub := h.store.SubscribeBack(channel, func(msg []byte) {
					h.publish(channel, msg, "", false)
				})
				h.unsubscribes[channel] = unsub
			}
		}
	}
	h.channels[channel][clientID] = c
	c.subs[channel] = struct{}{}
	h.mu.Unlock()
}

// Unsubscribe removes a client's subscription from a channel.
func (h *Hub) Unsubscribe(clientID, channel string) {
	h.mu.Lock()
	c, ok := h.conns[clientID]
	if !ok {
		h.mu.Unlock()
		return
	}
	if subs, ok := h.channels[channel]; ok {
		delete(subs, clientID)
		if len(subs) == 0 {
			delete(h.channels, channel)
			if unsub, ok := h.unsubscribes[channel]; ok {
				unsub()
				delete(h.unsubscribes, channel)
			}
		}
	}
	delete(c.subs, channel)
	h.mu.Unlock()
}

// CreateChannel ensures a channel exists. Useful for admin-created channels.
func (h *Hub) CreateChannel(channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[string]*Client)
		if h.store != nil {
			if _, exists := h.unsubscribes[channel]; !exists {
				unsub := h.store.SubscribeBack(channel, func(msg []byte) {
					h.publish(channel, msg, "", false)
				})
				h.unsubscribes[channel] = unsub
			}
		}
	}
}

// ListChannels returns the currently known channel names.
func (h *Hub) ListChannels() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	res := make([]string, 0, len(h.channels))
	for ch := range h.channels {
		res = append(res, ch)
	}
	return res
}

// ListClients returns the currently connected client IDs.
func (h *Hub) ListClients() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	res := make([]string, 0, len(h.conns))
	for id := range h.conns {
		res = append(res, id)
	}
	return res
}

// Publish delivers msg to all subscribers of channel. excludeClientID, if non-empty,
// will not receive the message (useful to avoid echo).
func (h *Hub) Publish(channel string, rawMsg []byte, excludeClientID string) {
	h.publish(channel, rawMsg, excludeClientID, true)
}

func (h *Hub) publish(channel string, rawMsg []byte, excludeClientID string, writeToStore bool) {
	if writeToStore && h.store != nil {
		if err := h.store.Publish(channel, rawMsg); err != nil {
			log.Printf("store publish error: %v", err)
		}
		if err := h.store.SaveHistory(channel, rawMsg); err != nil {
			log.Printf("store save history error: %v", err)
		}
	}

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
			log.Printf("subscriber %s channel %s send buffer full; dropping message", id, channel)
		}
	}
}

// CheckRateLimit checks if the client is allowed to send a message.
// Returns true if allowed, false if rate limited.
func (h *Hub) CheckRateLimit(clientID string) bool {
	if h.rl == nil {
		return true
	}
	return h.rl.Allow(clientID)
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
