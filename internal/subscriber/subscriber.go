package subscriber

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"

	"clipsync/internal/agentws"
	"clipsync/internal/clipboard"
	"clipsync/pkg/proto"
)

type Subscriber struct {
	ws       *agentws.AgentWS
	channel  string
	lastHash string
	stopChan chan struct{}
}

func NewSubscriber(ws *agentws.AgentWS, channel string) *Subscriber {
	return &Subscriber{
		ws:       ws,
		channel:  channel,
		stopChan: make(chan struct{}),
	}
}

func (s *Subscriber) Start() {
	log.Printf("subscriber started, channel=%s", s.channel)

	for {
		select {
		case <-s.stopChan:
			log.Printf("subscriber stopping")
			return
		case env := <-s.ws.MessageChan():
			s.handleMessage(env)
		}
	}
}

func (s *Subscriber) Stop() {
	close(s.stopChan)
}

func (s *Subscriber) handleMessage(env *proto.Envelope) {
	if env.Type != "publish" {
		return
	}

	var pb proto.PublishBody
	if err := json.Unmarshal(env.Body, &pb); err != nil {
		log.Printf("subscriber: invalid publish body: %v", err)
		return
	}

	if pb.Message.MessageID == "" {
		return
	}

	if pb.Message.Text == "" {
		return
	}

	h := sha256.Sum256([]byte(pb.Message.Text))
	hash := hex.EncodeToString(h[:])
	if hash == s.lastHash {
		return
	}

	if err := clipboard.WriteText(pb.Message.Text); err != nil {
		log.Printf("subscriber: write clipboard failed: %v", err)
		return
	}

	s.lastHash = hash
	log.Printf("subscriber: wrote clipboard: %d bytes, hash=%s, origin=%s", len(pb.Message.Text), hash, pb.Message.Origin)
}
