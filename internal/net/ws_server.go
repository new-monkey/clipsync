package netw

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"clipsync/internal/hub"
	"clipsync/pkg/proto"

	"github.com/gorilla/websocket"
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
			conn.SetPongHandler(func(appData string) error {
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				return nil
			})
			for {
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				_, msg, err := conn.ReadMessage()
				if err != nil {
					log.Printf("ws read error: %v", err)
					return
				}
				var env proto.Envelope
				if err := json.Unmarshal(msg, &env); err != nil {
					client.SendError("", "invalid envelope: "+err.Error())
					continue
				}
				if env.Type == "" {
					client.SendError("", "empty message type")
					continue
				}
				client.HandleMessage(&env)
			}
		}()
	}
}

// helper to get a free TCP listener port for tests
func ListenOnFreePort() (net.Listener, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	return ln, ln.Addr().String(), nil
}
