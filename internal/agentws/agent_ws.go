package agentws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"clipsync/pkg/proto"

	"github.com/gorilla/websocket"
)

type AgentWS struct {
	conn              *websocket.Conn
	clientID          string
	token             string
	wsURL             string
	reconnect         bool
	mu                sync.Mutex
	messageChan       chan *proto.Envelope
	stopChan          chan struct{}
	subscribed        map[string]bool
	writeMu           sync.Mutex
	hbStopChan        chan struct{}
	heartbeatInterval time.Duration
}

func NewAgentWS(wsURL, clientID, token string, reconnect bool) *AgentWS {
	return &AgentWS{
		clientID:    clientID,
		token:       token,
		wsURL:       wsURL,
		reconnect:   reconnect,
		messageChan: make(chan *proto.Envelope, 64),
		stopChan:    make(chan struct{}),
		subscribed:  make(map[string]bool),
	}
}

func (a *AgentWS) Connect() error {
	var err error
	a.conn, _, err = websocket.DefaultDialer.Dial(a.wsURL, nil)
	if err != nil {
		return err
	}
	a.conn.SetReadLimit(1024 * 1024)
	a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	a.conn.SetPingHandler(func(appData string) error {
		a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		a.writeMu.Lock()
		err := a.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
		a.writeMu.Unlock()
		return err
	})
	a.conn.SetPongHandler(func(appData string) error {
		a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	return nil
}

func (a *AgentWS) Authenticate() error {
	env := proto.Envelope{
		Type: "auth",
		ID:   "auth-1",
	}
	body := proto.AuthBody{
		Token:    a.token,
		ClientID: a.clientID,
	}
	bb, _ := json.Marshal(body)
	env.Body = bb
	raw, _ := json.Marshal(env)

	a.writeMu.Lock()
	a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := a.conn.WriteMessage(websocket.TextMessage, raw)
	a.writeMu.Unlock()
	if err != nil {
		return err
	}

	a.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := a.conn.ReadMessage()
	if err != nil {
		return err
	}

	var ack proto.Envelope
	if err := json.Unmarshal(msg, &ack); err != nil {
		return err
	}
	if ack.Type == "error" {
		return &AuthError{msg: string(msg)}
	}
	return nil
}

func (a *AgentWS) Subscribe(channel string) error {
	env := proto.Envelope{
		Type: "subscribe",
		ID:   "sub-" + channel,
	}
	body := proto.SubscribeBody{Channel: channel}
	bb, _ := json.Marshal(body)
	env.Body = bb
	raw, _ := json.Marshal(env)

	a.writeMu.Lock()
	a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := a.conn.WriteMessage(websocket.TextMessage, raw)
	a.writeMu.Unlock()
	if err != nil {
		return err
	}

	a.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := a.conn.ReadMessage()
	if err != nil {
		return err
	}

	var ack proto.Envelope
	if err := json.Unmarshal(msg, &ack); err != nil {
		return err
	}
	if ack.Type == "error" {
		return &SubscribeError{channel: channel, msg: string(msg)}
	}

	a.mu.Lock()
	a.subscribed[channel] = true
	a.mu.Unlock()
	return nil
}

func (a *AgentWS) Unsubscribe(channel string) error {
	env := proto.Envelope{
		Type: "unsubscribe",
		ID:   "unsub-" + channel,
	}
	body := proto.UnsubscribeBody{Channel: channel}
	bb, _ := json.Marshal(body)
	env.Body = bb
	raw, _ := json.Marshal(env)

	a.writeMu.Lock()
	a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := a.conn.WriteMessage(websocket.TextMessage, raw)
	a.writeMu.Unlock()
	if err != nil {
		return err
	}

	a.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, msg, err := a.conn.ReadMessage()
	if err != nil {
		return err
	}

	var ack proto.Envelope
	if err := json.Unmarshal(msg, &ack); err != nil {
		return err
	}
	if ack.Type == "error" {
		return &UnsubscribeError{channel: channel, msg: string(msg)}
	}

	a.mu.Lock()
	delete(a.subscribed, channel)
	a.mu.Unlock()
	return nil
}

func (a *AgentWS) Publish(channel string, msg *proto.ClipMessage) error {
	env := proto.Envelope{
		Type: "publish",
		ID:   "pub-" + msg.MessageID,
	}
	body := proto.PublishBody{
		Channel: channel,
		Message: *msg,
	}
	bb, _ := json.Marshal(body)
	env.Body = bb
	raw, _ := json.Marshal(env)

	a.writeMu.Lock()
	a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := a.conn.WriteMessage(websocket.TextMessage, raw)
	a.writeMu.Unlock()
	return err
}

func (a *AgentWS) DirectSend(targetID string, msg *proto.ClipMessage) error {
	env := proto.Envelope{
		Type: "direct",
		ID:   "dir-" + msg.MessageID,
	}
	body := proto.DirectBody{
		Target:  targetID,
		Message: *msg,
	}
	bb, _ := json.Marshal(body)
	env.Body = bb
	raw, _ := json.Marshal(env)

	a.writeMu.Lock()
	a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := a.conn.WriteMessage(websocket.TextMessage, raw)
	a.writeMu.Unlock()
	return err
}

func (a *AgentWS) StartReader() {
	go func() {
		for {
			select {
			case <-a.stopChan:
				return
			default:
			}

			a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, msg, err := a.conn.ReadMessage()
			if err != nil {
				log.Printf("agentws read error: %v", err)
				if a.reconnect {
					a.reconnectLoop()
					continue
				}
				return
			}

			var env proto.Envelope
			if err := json.Unmarshal(msg, &env); err != nil {
				log.Printf("agentws invalid envelope: %v", err)
				continue
			}

			// heartbeat pong response, not for application
			if env.Type == "pong" {
				continue
			}

			select {
			case a.messageChan <- &env:
			default:
				log.Printf("agentws message buffer full, dropping message")
			}
		}
	}()
}

func (a *AgentWS) reconnectLoop() {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-a.stopChan:
			return
		default:
		}

		log.Printf("agentws reconnecting in %v...", backoff)
		time.Sleep(backoff)

		if err := a.Connect(); err != nil {
			log.Printf("agentws reconnect failed: %v", err)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		log.Printf("agentws reconnected to %s", a.wsURL)

		if err := a.Authenticate(); err != nil {
			log.Printf("agentws reauth failed: %v", err)
			a.conn.Close()
			continue
		}

		log.Printf("agentws reauthenticated")

		a.mu.Lock()
		channels := make([]string, 0, len(a.subscribed))
		for ch := range a.subscribed {
			channels = append(channels, ch)
		}
		a.mu.Unlock()
		for _, ch := range channels {
			if err := a.Subscribe(ch); err != nil {
				log.Printf("agentws resubscribe %s failed: %v", ch, err)
			}
		}

		log.Printf("agentws all channels resubscribed")

		a.stopHeartbeat()
		a.startHeartbeatGoroutine()
		return
	}
}

func (a *AgentWS) Close() error {
	a.stopHeartbeat()
	close(a.stopChan)
	if a.conn != nil {
		return a.conn.Close()
	}
	return nil
}

func (a *AgentWS) MessageChan() <-chan *proto.Envelope {
	return a.messageChan
}

func (a *AgentWS) ClientID() string {
	return a.clientID
}

func (a *AgentWS) StartHeartbeat(interval time.Duration) {
	if interval <= 0 {
		return
	}
	a.mu.Lock()
	a.heartbeatInterval = interval
	if a.hbStopChan != nil {
		close(a.hbStopChan)
		a.hbStopChan = nil
	}
	a.mu.Unlock()
	a.startHeartbeatGoroutine()
}

func (a *AgentWS) startHeartbeatGoroutine() {
	a.mu.Lock()
	interval := a.heartbeatInterval
	a.mu.Unlock()
	if interval <= 0 {
		return
	}

	ch := make(chan struct{})
	a.mu.Lock()
	a.hbStopChan = ch
	a.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ch:
				return
			case <-ticker.C:
				if err := a.sendPing(); err != nil {
					log.Printf("agentws heartbeat ping error: %v", err)
					return
				}
			}
		}
	}()
}

func (a *AgentWS) stopHeartbeat() {
	a.mu.Lock()
	if a.hbStopChan != nil {
		close(a.hbStopChan)
		a.hbStopChan = nil
	}
	a.mu.Unlock()
}

func (a *AgentWS) sendPing() error {
	env := proto.Envelope{Type: "ping", ID: "hb"}
	raw, _ := json.Marshal(env)
	a.writeMu.Lock()
	a.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := a.conn.WriteMessage(websocket.TextMessage, raw)
	a.writeMu.Unlock()
	return err
}

type AuthError struct {
	msg string
}

func (e *AuthError) Error() string {
	return "auth failed: " + e.msg
}

type SubscribeError struct {
	channel string
	msg     string
}

func (e *SubscribeError) Error() string {
	return "subscribe " + e.channel + " failed: " + e.msg
}

type UnsubscribeError struct {
	channel string
	msg     string
}

func (e *UnsubscribeError) Error() string {
	return "unsubscribe " + e.channel + " failed: " + e.msg
}
