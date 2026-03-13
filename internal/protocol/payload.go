package protocol

// ClipPayload is the JSON schema used between client and server.
type ClipPayload struct {
	MachineID string `json:"machine_id"`
	Timestamp string `json:"timestamp"`
	Text      string `json:"text"`
	SHA256    string `json:"sha256"`
}
