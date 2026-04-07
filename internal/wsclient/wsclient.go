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

	log.Printf("Connecting to ws://%s ...", addr)
	c, _, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws", nil)
	if err != nil {
		return err
	}
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				return
			}
			var payload protocol.ClipPayload
			if err := json.Unmarshal(message, &payload); err != nil {
				log.Println("invalid payload:", err)
				continue
			}
			if onClip != nil {
				onClip(payload)
			}
		}
	}()

	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			log.Println("interrupt")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			time.Sleep(time.Second)
			return nil
		}
	}
}
