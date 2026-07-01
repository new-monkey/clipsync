package store

// Store is a minimal abstraction for pub/sub and optional history persistence.
// Implementations: MemoryStore (MVP), BoltStore, RedisStore.
type Store interface {
	// Publish publishes a raw message payload to the named channel.
	Publish(channel string, msg []byte) error
	// SubscribeBack registers a callback which is invoked for messages published
	// to the channel. It returns an unsubscribe function.
	SubscribeBack(channel string, handler func(msg []byte)) (unsubscribe func())
	// SaveHistory optionally persists a message for later retrieval/offline delivery.
	SaveHistory(channel string, msg []byte) error
	// ListHistory returns the most recent messages up to limit.
	ListHistory(channel string, limit int) ([][]byte, error)
}
