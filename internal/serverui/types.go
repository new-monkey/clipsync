package serverui

import "time"

type ClipEntry struct {
	ReceivedAt time.Time
	Machine    string
	Bytes      int
	SHA256     string
	Text       string
}

type Options struct {
	Title       string
	AlwaysOnTop bool
	MaxHistory  int
}

type Manager interface {
	Publish(entry ClipEntry)
	Close()
}
