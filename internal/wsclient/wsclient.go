package wsclient

import (
	"clipsync/internal/protocol"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// OnClipReceivedFunc 剪贴板数据回调
// 可用于保存、打印或写入本地剪贴板
// func OnClipReceived(payload protocol.ClipPayload)

type OnClipReceivedFunc func(payload protocol.ClipPayload)

func RunWSClient(addr string, token string, onClip OnClipReceivedFunc) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	wsURL := "ws://" + addr + "/ws"
	log.Printf("[mode] server running in reverse-push mode, connecting to wsURL: %s", wsURL)
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Printf("[reverse-push] ws connect error: %v", err)
		return err
	}
	log.Printf("[reverse-push] ws connected to %s", wsURL)
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("[reverse-push] ws read error: %v", err)
				return
			}
			var payload protocol.ClipPayload
			if err := json.Unmarshal(message, &payload); err != nil {
				log.Printf("[reverse-push] invalid payload: %v", err)
				continue
			}
			log.Printf("[reverse-push] received clipboard: %d bytes, hash=%s", len(payload.Text), payload.SHA256)
			if onClip != nil {
				onClip(payload)
			}
		}
	}()

	for {
		select {
		case <-done:
			log.Printf("[reverse-push] ws receive done")
			return nil
		case <-interrupt:
			log.Printf("[reverse-push] ws interrupt, closing connection")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			time.Sleep(time.Second)
			return nil
		}
	}
}
