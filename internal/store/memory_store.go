package store

import (
	"sync"
)

// MemoryStore is a simple in-process store used for MVP and tests.
// It supports Publish -> notify subscribers and an in-memory history buffer.
type MemoryStore struct {
	mu      sync.RWMutex
	subs    map[string]map[int]func(msg []byte)
	history map[string][][]byte
	nextID  int
}

// NewMemoryStore constructs a ready MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		subs:    make(map[string]map[int]func(msg []byte)),
		history: make(map[string][][]byte),
		nextID:  1,
	}
}

func (m *MemoryStore) Publish(channel string, msg []byte) error {
	m.mu.RLock()
	subs := m.subs[channel]
	m.mu.RUnlock()
	for _, h := range subs {
		h := h
		go h(msg)
	}
	return nil
}

func (m *MemoryStore) SubscribeBack(channel string, handler func(msg []byte)) (func()) {
	m.mu.Lock()
	id := m.nextID
	m.nextID++
	if _, ok := m.subs[channel]; !ok {
		m.subs[channel] = make(map[int]func(msg []byte))
	}
	m.subs[channel][id] = handler
	m.mu.Unlock()

	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if s, ok := m.subs[channel]; ok {
			delete(s, id)
			if len(s) == 0 {
				delete(m.subs, channel)
			}
		}
	}
}

func (m *MemoryStore) SaveHistory(channel string, msg []byte) error {
	m.mu.Lock()
	m.history[channel] = append(m.history[channel], append([]byte(nil), msg...))
	m.mu.Unlock()
	return nil
}

func (m *MemoryStore) ListHistory(channel string, limit int) ([][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	arr := m.history[channel]
	if limit <= 0 || limit > len(arr) {
		limit = len(arr)
	}
	start := len(arr) - limit
	if start < 0 {
		start = 0
	}
	res := make([][]byte, 0, limit)
	for i := start; i < len(arr); i++ {
		res = append(res, append([]byte(nil), arr[i]...))
	}
	return res, nil
}
