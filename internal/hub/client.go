package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"clipsync/pkg/proto"
	"clipsync/internal/auth"
)

// Client represents a connected websocket client.
type Client struct {
	ID     string
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	subs   map[string]struct{}
	mu     sync.Mutex // protects auth state
	authed bool
	// lastSeen can be used for TTL/idle disconnects
	lastSeen time.Time
}

// NewClient constructs a client and starts its write pump.
func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	c := &Client{
		conn:   conn,
		hub:    hub,
		send:   make(chan []byte, 64),
		subs:   make(map[string]struct{}),
		authed: false,
		lastSeen: time.Now(),
	}
	go c.writePump()
	return c
}

func (c *Client) Close() {
	c.conn.Close()
	// ensure send channel closed to stop writer
	c.mu.Lock()
	if c.send != nil {
		close(c.send)
		c.send = nil
	}
	c.mu.Unlock()
}

// writePump sends messages from c.send to the websocket connection.
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for b := range c.send {
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
			log.Printf("client %s write error: %v", c.ID, err)
			return
		}
	}
}

// handleMessage handles an incoming envelope. Called from reader loop.
func (c *Client) handleMessage(env *proto.Envelope) {
	switch env.Type {
	case "auth":
		var b struct {
			Token    string `json:"token"`
			ClientID string `json:"client_id"`
		}
		if err := json.Unmarshal(env.Body, &b); err != nil {
			c.sendError(env.ID, "invalid auth body")
			c.Close()
			return
		}
		// token check using JWT/legacy fallback
		if !auth.ValidateToken(b.Token) {
			c.sendError(env.ID, "unauthorized")
			c.Close()
			return
		}
		c.mu.Lock()
		c.ID = b.ClientID
		c.authed = true
		c.mu.Unlock()
		// register with hub
		c.hub.RegisterClient(c)
		c.sendAck(env.ID)
	case "subscribe":
		var sb struct{ Channel string `json:"channel"` }
		if err := json.Unmarshal(env.Body, &sb); err != nil {
			c.sendError(env.ID, "invalid subscribe body")
			return
		}
		if !c.ensureAuthed(env.ID) { return }
		c.hub.Subscribe(c.ID, sb.Channel)
		c.sendAck(env.ID)
	case "unsubscribe":
		var sb struct{ Channel string `json:"channel"` }
		if err := json.Unmarshal(env.Body, &sb); err != nil {
			c.sendError(env.ID, "invalid unsubscribe body")
			return
		}
		if !c.ensureAuthed(env.ID) { return }
		c.hub.Unsubscribe(c.ID, sb.Channel)
		c.sendAck(env.ID)
	case "publish":
		if !c.ensureAuthed(env.ID) { return }
		// forward raw body to hub.Publish. The body includes channel+message
		// We build a delivery envelope for subscribers which is the original
		// publish envelope (so clients can parse consistently).
		raw, _ := json.Marshal(env)
		// extract channel to pass exclude
		var pb struct{ Channel string `json:"channel"` }
		json.Unmarshal(env.Body, &pb)
		// exclude sender to avoid echo; default behavior
		c.hub.Publish(pb.Channel, raw, c.ID)
		c.sendAck(env.ID)
	case "direct":
		if !c.ensureAuthed(env.ID) { return }
		var db struct{
			Target string `json:"target"`
		}
		if err := json.Unmarshal(env.Body, &db); err != nil {
			c.sendError(env.ID, "invalid direct body")
			return
		}
		raw, _ := json.Marshal(env)
		ok := c.hub.DirectSend(db.Target, raw)
		if ok {
			c.sendAck(env.ID)
		} else {
			c.sendError(env.ID, "target not connected")
		}
	case "ping":
		c.sendPong(env.ID)
	default:
		c.sendError(env.ID, "unknown message type")
	}
}

func (c *Client) sendAck(refID string) {
	env := proto.Envelope{Type: "ack", ID: "", Body: nil}
	b := struct{ RefID string `json:"ref_id"`; Status string `json:"status"` }{RefID: refID, Status: "ok"}
	bb, _ := json.Marshal(b)
	env.Body = bb
	raw, _ := json.Marshal(env)
	select {
	case c.send <- raw:
	default:
		// drop
	}
}

func (c *Client) sendError(refID, reason string) {
	env := proto.Envelope{Type: "error", ID: "", Body: nil}
	b := struct{ RefID string `json:"ref_id"`; Error string `json:"error"` }{RefID: refID, Error: reason}
	bb, _ := json.Marshal(b)
	env.Body = bb
	raw, _ := json.Marshal(env)
	select {
	case c.send <- raw:
	default:
	}
}

func (c *Client) sendPong(refID string) {
	env := proto.Envelope{Type: "pong", ID: refID}
	raw, _ := json.Marshal(env)
	select {
	case c.send <- raw:
	default:
	}
}

func (c *Client) ensureAuthed(refID string) bool {
	c.mu.Lock()
	authed := c.authed
	c.mu.Unlock()
	if !authed {
		c.sendError(refID, "unauthorized: must auth first")
		c.Close()
		return false
	}
	return true
}
