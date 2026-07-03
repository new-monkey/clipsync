package publisher

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"clipsync/internal/agentws"
	"clipsync/pkg/proto"

	"github.com/google/uuid"
)

type Publisher struct {
	ws       *agentws.AgentWS
	channel  string
	interval time.Duration
	maxBytes int
	lastHash string
	ticker   *time.Ticker
	stopChan chan struct{}
}

func NewPublisher(ws *agentws.AgentWS, channel string, interval time.Duration, maxBytes int) *Publisher {
	return &Publisher{
		ws:       ws,
		channel:  channel,
		interval: interval,
		maxBytes: maxBytes,
		stopChan: make(chan struct{}),
	}
}

func (p *Publisher) Start() {
	p.ticker = time.NewTicker(p.interval)
	defer p.ticker.Stop()

	log.Printf("publisher started, channel=%s, interval=%s, maxBytes=%d", p.channel, p.interval, p.maxBytes)

	for {
		select {
		case <-p.stopChan:
			log.Printf("publisher stopping")
			return
		case <-p.ticker.C:
			p.poll()
		}
	}
}

func (p *Publisher) Stop() {
	close(p.stopChan)
}

func (p *Publisher) poll() {
	text, err := readClipboardText()
	if err != nil {
		return
	}

	if strings.TrimSpace(text) == "" {
		return
	}

	textBytes := len([]byte(text))
	if textBytes > p.maxBytes {
		log.Printf("publisher: clipboard too large: %d bytes (max=%d)", textBytes, p.maxBytes)
		return
	}

	h := sha256.Sum256([]byte(text))
	hash := hex.EncodeToString(h[:])
	if hash == p.lastHash {
		return
	}

	msg := &proto.ClipMessage{
		MessageID: uuid.New().String(),
		Timestamp: time.Now().Format(time.RFC3339),
		Origin:    p.ws.ClientID(),
		Text:      text,
		Meta: map[string]string{
			"hash": hash,
		},
	}

	if err := p.ws.Publish(p.channel, msg); err != nil {
		log.Printf("publisher publish failed: %v", err)
		return
	}

	p.lastHash = hash
	log.Printf("publisher: pushed clipboard: %d bytes, hash=%s", textBytes, hash)
}