package main

import (
	"clipsync/internal/protocol"
	"clipsync/internal/wsclient"
	"flag"
	"log"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", "127.0.0.1:8081", "A端WebSocket服务地址 host:port")
	flag.Parse()

	err := wsclient.RunWSClient(addr, "", func(payload protocol.ClipPayload) {
		log.Printf("收到剪贴板: %s (from %s)", payload.Text, payload.MachineID)
		// TODO: 可在此处写入本地剪贴板
	})
	if err != nil {
		log.Fatalf("wsclient error: %v", err)
	}
}
