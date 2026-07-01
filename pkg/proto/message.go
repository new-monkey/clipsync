package proto

import "encoding/json"

// Package proto defines the v2 JSON envelope and message shapes used by the new
// pub/sub protocol. Keep these types minimal and stable.

// Envelope is the outer wrapper for every WS message.
// JSON: {"type":"publish","id":"...","body":{...}}
type Envelope struct {
	Type string          `json:"type"`
	ID   string          `json:"id,omitempty"`
	Body json.RawMessage `json:"body,omitempty"`
}
