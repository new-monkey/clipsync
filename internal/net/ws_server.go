package netw

import (
	"encoding/json"
	"log"
	"net"
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
					// send a structured error back to the peer
					errEnv := proto.Envelope{Type: "error"}
					b := struct{ Error string `json:"error"` }{Error: "invalid envelope: " + err.Error()}
					bb, _ := json.Marshal(b)
					errEnv.Body = bb
					if rb, rerr := json.Marshal(errEnv); rerr == nil {
						conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
						conn.WriteMessage(websocket.TextMessage, rb)
					}
					continue
				}
				// enforce a max message size in bytes (already SetReadLimit), and additional
				// reject empty envelopes
				if env.Type == "" {
					// send error
					errEnv := proto.Envelope{Type: "error"}
					b := struct{ Error string `json:"error"` }{Error: "empty message type"}
					bb, _ := json.Marshal(b)
					errEnv.Body = bb
					if rb, rerr := json.Marshal(errEnv); rerr == nil {
						conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
						conn.WriteMessage(websocket.TextMessage, rb)
					}
					continue
				}
				client.handleMessage(&env)
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
