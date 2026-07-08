package agentws

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"clipsync/internal/hub"
	netw "clipsync/internal/net"
	"clipsync/internal/store"

	"github.com/gorilla/websocket"
)

// testServer starts a WS server on a free port and returns its URL + close func.
func testServer(t *testing.T) (wsURL string, close func()) {
	t.Helper()
	os.Setenv("CLIPSYNC_AUTH_TOKEN", "test-token")
	st := store.NewMemoryStore()
	h := hub.NewHub(st)
	ln, addr, err := netw.ListenOnFreePort()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/v2/ws", netw.NewWSHandler(h))
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	return "ws://" + addr + "/v2/ws", func() {
		srv.Close()
		ln.Close()
	}
}

func TestAgentWSReconnect(t *testing.T) {
	serverURL, closeServer := testServer(t)
	defer closeServer()

	a := NewAgentWS(serverURL, "test-client", "test-token", true)

	if err := a.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer a.Close()

	if err := a.Authenticate(); err != nil {
		t.Fatalf("auth: %v", err)
	}

	if err := a.Subscribe("ch1"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := a.Subscribe("ch2"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// verify we have 2 subscribed channels
	a.mu.Lock()
	if len(a.subscribed) != 2 {
		a.mu.Unlock()
		t.Fatalf("expected 2 subscribed channels, got %d", len(a.subscribed))
	}
	a.mu.Unlock()

	a.StartReader()

	// wait briefly to ensure reader is running
	time.Sleep(50 * time.Millisecond)

	// force-close the websocket to trigger reconnect
	a.conn.Close()

	// wait for reconnect to happen + resubscribe
	time.Sleep(3 * time.Second)

	// verify no deadlock: the reader goroutine is alive and connection is new
	a.mu.Lock()
	if len(a.subscribed) != 2 {
		a.mu.Unlock()
		t.Fatalf("expected 2 subscribed channels after reconnect, got %d", len(a.subscribed))
	}
	a.mu.Unlock()

	// verify we can still receive messages: connect with raw ws and publish
	dialer := websocket.Dialer{}
	pubConn, _, err := dialer.Dial(serverURL, nil)
	if err != nil {
		t.Fatalf("dial publisher: %v", err)
	}
	defer pubConn.Close()

	authMsg, _ := json.Marshal(map[string]interface{}{
		"type": "auth", "id": "1",
		"body": map[string]string{"token": "test-token", "client_id": "publisher"},
	})
	pubConn.WriteMessage(websocket.TextMessage, authMsg)

	pubBody, _ := json.Marshal(map[string]interface{}{
		"type": "publish", "id": "2",
		"body": map[string]interface{}{
			"channel": "ch1",
			"message": map[string]string{
				"message_id": "m1", "timestamp": "2026-01-01T00:00:00Z",
				"origin": "publisher", "text": "hello after reconnect",
			},
		},
	})
	pubConn.WriteMessage(websocket.TextMessage, pubBody)

	// read from messageChan — should get the publish message
	select {
	case env := <-a.MessageChan():
		if env.Type != "publish" {
			t.Fatalf("expected publish, got %s", env.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message after reconnect")
	}
}

func TestAgentWSReconnectLoopNoDeadlock(t *testing.T) {
	// Tests that reconnectLoop doesn't self-deadlock on mu
	serverURL, closeServer := testServer(t)
	defer closeServer()

	a := NewAgentWS(serverURL, "test-client", "test-token", true)

	if err := a.Connect(); err != nil {
		t.Fatalf("connect: %v", err)
	}

	if err := a.Authenticate(); err != nil {
		t.Fatalf("auth: %v", err)
	}

	// subscribe to multiple channels
	if err := a.Subscribe("office"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := a.Subscribe("team"); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// close connection manually to simulate disconnect
	a.conn.Close()

	// ReconnectLoop should NOT deadlock. Run with timeout.
	done := make(chan struct{})
	go func() {
		a.reconnectLoop()
		close(done)
	}()

	select {
	case <-done:
		// success: reconnectLoop completed without deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("reconnectLoop deadlocked (timed out)")
	}

	// verify still subscribed
	if len(a.subscribed) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(a.subscribed))
	}
}
