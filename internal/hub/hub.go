package hub

import (
	"clipsync/internal/store"
)

// Hub is the in-process pub/sub broker. It relies on a Store to provide
// cross-instance pub/sub or persistence when configured.
type Hub struct {
	store store.Store
	// TODO: add connection maps and channel indices here
}

// NewHub creates a Hub with the provided Store backend.
func NewHub(s store.Store) *Hub {
	return &Hub{store: s}
}

// Register/Unregister/Subscribe/Publish will be implemented in follow-up PRs.

