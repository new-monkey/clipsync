package netw

import (
	"net/http"
	"testing"
	"time"
	"encoding/json"

	"github.com/gorilla/websocket"
	"clipsync/internal/hub"
	"clipsync/internal/store"
)

func TestWSPublishSubscribeIntegration(t *testing.T) {
	st := store.NewMemoryStore()
	h := hub.NewHub(st)
	ln, addr, err := ListenOnFreePort()
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/v2/ws", NewWSHandler(h))
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	wsURL := "ws://" + addr + "/v2/ws"
	// connect subscriber A
	dialer := websocket.Dialer{}
	connA, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial A failed: %v", err)
	}
	defer connA.Close()
	// connect publisher B
	connB, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial B failed: %v", err)
	}
	defer connB.Close()

	// auth both
	authA := map[string]interface{}{"type":"auth","id":"1","body":map[string]string{"token":"dev-token","client_id":"A"}}
	authB := map[string]interface{}{"type":"auth","id":"1","body":map[string]string{"token":"dev-token","client_id":"B"}}
	ab, _ := json.Marshal(authA)
	bb, _ := json.Marshal(authB)
	connA.WriteMessage(websocket.TextMessage, ab)
	connB.WriteMessage(websocket.TextMessage, bb)

	// subscribe A to channel "office"
	sub := map[string]interface{}{"type":"subscribe","id":"2","body":map[string]string{"channel":"office"}}
	sb, _ := json.Marshal(sub)
	connA.WriteMessage(websocket.TextMessage, sb)

	// allow processing
	time.Sleep(100 * time.Millisecond)

	// publish from B
	pub := map[string]interface{}{"type":"publish","id":"3","body":map[string]interface{}{"channel":"office","message":map[string]string{"message_id":"m1","timestamp":"2026-07-01T00:00:00Z","origin":"B","text":"hello"}}}
	pb, _ := json.Marshal(pub)
	connB.WriteMessage(websocket.TextMessage, pb)

	// read from A and check that it receives publish envelope
	connA.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := connA.ReadMessage()
	if err != nil {
		t.Fatalf("A did not receive message: %v", err)
	}
	var env map[string]interface{}
	if err := json.Unmarshal(msg, &env); err != nil {
		t.Fatalf("failed to unmarshal received: %v", err)
	}
	if env["type"] != "publish" {
		t.Fatalf("expected publish type, got %v", env["type"])
	}
	// ensure publisher B did not receive the message (no echo)
	connB.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = connB.ReadMessage()
	if err == nil {
		t.Fatalf("publisher B unexpectedly received a message")
	}
}
