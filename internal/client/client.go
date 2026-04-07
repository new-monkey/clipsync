package client

import (
	"bytes"
	"clipsync/internal/ws"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"clipsync/internal/protocol"
)

type Config struct {
	ServerURL    string
	WSListenAddr string // reverse-push模式下WebSocket监听地址
	Token        string
	Interval     time.Duration
	MachineID    string
	MaxClipBytes int
	Timeout      time.Duration
	Mode         string // push, reverse-push, pull
}

func Run(cfg Config) error {
	if cfg.Mode == "reverse-push" {
		return runReversePush(cfg)
	} else {
		if cfg.ServerURL == "" {
			return fmt.Errorf("server url is required")
		}
		if cfg.Interval <= 0 {
			cfg.Interval = 300 * time.Millisecond
		}
		if cfg.Timeout <= 0 {
			cfg.Timeout = 8 * time.Second
		}
		if cfg.MaxClipBytes <= 0 {
			cfg.MaxClipBytes = 1024 * 1024
		}
		if strings.TrimSpace(cfg.MachineID) == "" {
			host, err := os.Hostname()
			if err == nil && host != "" {
				cfg.MachineID = host
			} else {
				cfg.MachineID = "unknown"
			}
		}

		httpClient := &http.Client{Timeout: cfg.Timeout}
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		var lastHash string
		log.Printf("ClipSync client started, polling interval=%s", cfg.Interval)
		log.Printf("Server endpoint: %s", cfg.ServerURL)

		for range ticker.C {
			text, err := readClipboardText()
			if err != nil {
				continue
			}

			if strings.TrimSpace(text) == "" {
				continue
			}

			textBytes := len([]byte(text))
			if textBytes > cfg.MaxClipBytes {
				log.Printf("Skip oversized clipboard: %d bytes (max=%d)", textBytes, cfg.MaxClipBytes)
				continue
			}

			h := sha256.Sum256([]byte(text))
			hash := hex.EncodeToString(h[:])
			if hash == lastHash {
				continue
			}

			payload := protocol.ClipPayload{
				MachineID: cfg.MachineID,
				Timestamp: time.Now().Format(time.RFC3339Nano),
				Text:      text,
				SHA256:    hash,
			}

			if err := sendWithRetry(httpClient, cfg.ServerURL, cfg.Token, payload); err != nil {
				log.Printf("Push failed: %v", err)
				continue
			}

			lastHash = hash
			log.Printf("Pushed clipboard: %d bytes", textBytes)
		}

		return nil
	}
}

// reverse-push模式下，作为WebSocket服务端，B端主动连接A端，A端推送剪贴板内容
func runReversePush(cfg Config) error {
	addr := cfg.WSListenAddr
	if addr == "" {
		addr = ":8081" // 默认监听端口
	}
	hub := ws.NewHub()
	http.HandleFunc("/ws", hub.ServeWS)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen failed: %v", err)
	}
	go func() {
		log.Printf("WebSocket server listening on %s/ws", addr)
		if err := http.Serve(listener, nil); err != nil {
			log.Fatalf("WebSocket server error: %v", err)
		}
	}()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	var lastHash string
	for range ticker.C {
		text, err := readClipboardText()
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		textBytes := len([]byte(text))
		if textBytes > cfg.MaxClipBytes {
			continue
		}
		h := sha256.Sum256([]byte(text))
		hash := hex.EncodeToString(h[:])
		if hash == lastHash {
			continue
		}
		payload := protocol.ClipPayload{
			MachineID: cfg.MachineID,
			Timestamp: time.Now().Format(time.RFC3339Nano),
			Text:      text,
			SHA256:    hash,
		}
		b, _ := json.Marshal(payload)
		hub.Broadcast(b)
		lastHash = hash
		log.Printf("[reverse-push] pushed clipboard: %d bytes", textBytes)
	}
	return nil
}

func sendWithRetry(client *http.Client, url, token string, payload protocol.ClipPayload) error {
	var lastErr error
	backoff := 300 * time.Millisecond

	for attempt := 1; attempt <= 3; attempt++ {
		if err := sendOnce(client, url, token, payload); err != nil {
			lastErr = err
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		return nil
	}

	return lastErr
}

func sendOnce(client *http.Client, url, token string, payload protocol.ClipPayload) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-ClipSync-Token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}
