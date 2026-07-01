package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"clipsync/internal/hub"
	"clipsync/internal/net"
	"clipsync/internal/store"
)

func main() {
	var config string
	var roles string
	var listen string
	var tlsCert string
	var tlsKey string
	var wsPath string

	flag.StringVar(&config, "config", "", "path to config file")
	flag.StringVar(&roles, "roles", "server", "comma-separated roles: server,publisher,subscriber")
	flag.StringVar(&listen, "listen", ":8080", "server listen address")
	flag.StringVar(&tlsCert, "tls-cert", "", "path to TLS cert file (optional)")
	flag.StringVar(&tlsKey, "tls-key", "", "path to TLS key file (optional)")
	flag.StringVar(&wsPath, "ws-path", "/v2/ws", "websocket path to mount")
	flag.Parse()

	fmt.Printf("clipsync v2 (feature/v2-pubsub)\n")
	fmt.Printf("config=%s roles=%s listen=%s tls=%s\n", config, roles, listen, (tlsCert != ""))

	// instantiate MemoryStore (MVP default)
	st := store.NewMemoryStore()
	h := hub.NewHub(st)

	// HTTP mux and ws handler
	mux := http.NewServeMux()
	mux.Handle(wsPath, net.NewWSHandler(h))
	// placeholder admin endpoints
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	r := strings.Split(roles, ",")
	for _, role := range r {
		switch strings.TrimSpace(role) {
		case "server":
			log.Printf("starting server on %s (TLS cert provided=%v)", listen, tlsCert != "")
		case "publisher":
			log.Printf("starting publisher (clipboard watcher)")
		case "subscriber":
			log.Printf("starting subscriber (clipboard writer)")
		default:
			log.Printf("unknown role: %s", role)
		}
	}

	srv := &http.Server{Addr: listen, Handler: mux}
	if tlsCert != "" && tlsKey != "" {
		log.Printf("listening (TLS) on %s", listen)
		if err := srv.ListenAndServeTLS(tlsCert, tlsKey); err != nil {
			log.Fatalf("server exited: %v", err)
		}
	} else {
		log.Printf("listening (plain) on %s", listen)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("server exited: %v", err)
		}
	}
}
