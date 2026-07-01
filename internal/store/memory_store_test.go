package store

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryStorePublishSubscribe(t *testing.T) {
	m := NewMemoryStore()
	var seen int32
	unsub := m.SubscribeBack("test-channel", func(msg []byte) {
		if len(msg) == 0 {
			t.Fatalf("expected non-empty msg")
		}
		atomic.AddInt32(&seen, 1)
	})
	defer unsub()

	if err := m.Publish("test-channel", []byte("hello")); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// give goroutine a moment
	for i := 0; i < 100 && atomic.LoadInt32(&seen) == 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&seen) == 0 {
		t.Fatalf("subscriber did not receive message")
	}
}
