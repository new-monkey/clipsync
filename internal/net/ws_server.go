package netw

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"clipsync/internal/hub"
	"clipsync/pkg/proto"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWSHandler returns an http.Handler that upgrades connections to websockets
// and drives the hub client lifecycle.
func NewWSHandler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade failed: %v", err)
			return
		}
		client := hub.NewClient(conn, h)

		// reader loop
		go func() {
			defer func() {
				// ensure unregister and close
				h.UnregisterClient(client)
				client.Close()
			}()
			conn.SetReadLimit(1024 * 1024)
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			conn.SetPongHandler(func(appData string) error {
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				return nil
			})
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					log.Printf("ws read error: %v", err)
					return
				}
				var env proto.Envelope
				if err := json.Unmarshal(msg, &env); err != nil {
					log.Printf("invalid envelope: %v", err)
					// send error and continue
					continue
				}
				client.handleMessage(&env)
			}
		}()
	}
}
