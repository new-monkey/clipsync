package proto

// AuthBody sent by a client immediately after connecting.
type AuthBody struct {
	Token    string `json:"token"`
	ClientID string `json:"client_id"`
}

// SubscribeBody requests subscription to a channel.
type SubscribeBody struct {
	Channel string `json:"channel"`
}

// ClipMessage is the payload transported inside publish/direct messages.
type ClipMessage struct {
	MessageID string            `json:"message_id"`
	Timestamp string            `json:"timestamp"`
	Origin    string            `json:"origin"`
	Text      string            `json:"text"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// PublishBody is the body for a publish envelope.
type PublishBody struct {
	Channel string      `json:"channel"`
	Message ClipMessage `json:"message"`
}

// DirectBody is for one-to-one delivery.
type DirectBody struct {
	Target  string      `json:"target"`
	Message ClipMessage `json:"message"`
}
