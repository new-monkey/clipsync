package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"clipsync/internal/auth"
	"clipsync/pkg/proto"

	"github.com/gorilla/websocket"
)

const MaxMessageTextBytes = 1048576 // 1MB max for message text

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
		conn:     conn,
		hub:      hub,
		send:     make(chan []byte, 64),
		subs:     make(map[string]struct{}),
		authed:   false,
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

// writePump sends messages from c.send to the websocket connection and sends
// periodic control pings to keep the connection alive.
func (c *Client) writePump() {
	pingTicker := time.NewTicker(25 * time.Second)
	defer func() {
		pingTicker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case b, ok := <-c.send:
			if !ok {
				// channel closed
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
				log.Printf("client %s write error: %v", c.ID, err)
				return
			}
		case <-pingTicker.C:
			// send control ping frame
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				log.Printf("client %s ping error: %v", c.ID, err)
				return
			}
		}
	}
}

// HandleMessage handles an incoming envelope. Called from reader loop.
func (c *Client) HandleMessage(env *proto.Envelope) {
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
		var sb proto.SubscribeBody
		if err := json.Unmarshal(env.Body, &sb); err != nil {
			c.sendError(env.ID, "invalid subscribe body")
			return
		}
		if !c.ensureAuthed(env.ID) {
			return
		}
		c.hub.Subscribe(c.ID, sb.Channel)
		c.sendAck(env.ID)
	case "unsubscribe":
		var sb proto.UnsubscribeBody
		if err := json.Unmarshal(env.Body, &sb); err != nil {
			c.sendError(env.ID, "invalid unsubscribe body")
			return
		}
		if !c.ensureAuthed(env.ID) {
			return
		}
		c.hub.Unsubscribe(c.ID, sb.Channel)
		c.sendAck(env.ID)
	case "publish":
		if !c.ensureAuthed(env.ID) {
			return
		}
		if !c.hub.CheckRateLimit(c.ID) {
			c.sendError(env.ID, "rate limited")
			return
		}
		// validate publish body
		var pb proto.PublishBody
		if err := json.Unmarshal(env.Body, &pb); err != nil {
			c.sendError(env.ID, "invalid publish body")
			return
		}
		if pb.Channel == "" {
			c.sendError(env.ID, "publish missing channel")
			return
		}
		if pb.Message.MessageID == "" {
			c.sendError(env.ID, "message_id required")
			return
		}
		if pb.Message.Text == "" {
			c.sendError(env.ID, "empty message text")
			return
		}
		if len(pb.Message.Text) > MaxMessageTextBytes {
			c.sendError(env.ID, "message too large")
			return
		}
		if pb.Message.Timestamp == "" {
			c.sendError(env.ID, "timestamp required")
			return
		}
		if pb.Message.Origin == "" {
			c.sendError(env.ID, "origin required")
			return
		}
		// forward the original envelope JSON to subscribers; extract raw
		raw, _ := json.Marshal(env)
		c.hub.Publish(pb.Channel, raw, c.ID)
		c.sendAck(env.ID)
	case "direct":
		if !c.ensureAuthed(env.ID) {
			return
		}
		if !c.hub.CheckRateLimit(c.ID) {
			c.sendError(env.ID, "rate limited")
			return
		}
		var db proto.DirectBody
		if err := json.Unmarshal(env.Body, &db); err != nil {
			c.sendError(env.ID, "invalid direct body")
			return
		}
		if db.Target == "" {
			c.sendError(env.ID, "direct target required")
			return
		}
		if db.Message.MessageID == "" {
			c.sendError(env.ID, "message_id required")
			return
		}
		if db.Message.Text == "" {
			c.sendError(env.ID, "empty message text")
			return
		}
		if len(db.Message.Text) > MaxMessageTextBytes {
			c.sendError(env.ID, "message too large")
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
	b := struct {
		RefID  string `json:"ref_id"`
		Status string `json:"status"`
	}{RefID: refID, Status: "ok"}
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
	b := struct {
		RefID string `json:"ref_id"`
		Error string `json:"error"`
	}{RefID: refID, Error: reason}
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
