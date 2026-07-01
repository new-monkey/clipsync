package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"clipsync/internal/hub"
	"clipsync/internal/store"
)

func main() {
	var config string
	var roles string
	var listen string
	var tlsCert string
	var tlsKey string

	flag.StringVar(&config, "config", "", "path to config file")
	flag.StringVar(&roles, "roles", "server", "comma-separated roles: server,publisher,subscriber")
	flag.StringVar(&listen, "listen", ":8080", "server listen address")
	flag.StringVar(&tlsCert, "tls-cert", "", "path to TLS cert file (optional)")
	flag.StringVar(&tlsKey, "tls-key", "", "path to TLS key file (optional)")
	flag.Parse()

	fmt.Printf("clipsync v2 (feature/v2-pubsub)\n")
	fmt.Printf("config=%s roles=%s listen=%s tls=%s\n", config, roles, listen, (tlsCert != ""))

	// instantiate MemoryStore (MVP default)
	st := store.NewMemoryStore()
	h := hub.NewHub(st)
	_ = h // hub will be used in later PRs

	// simple role handling for the skeleton: print what would run
	r := strings.Split(roles, ",")
	for _, role := range r {
		switch strings.TrimSpace(role) {
		case "server":
			log.Printf("[skeleton] would start server on %s (TLS cert provided=%v)", listen, tlsCert != "")
		case "publisher":
			log.Printf("[skeleton] would start publisher (clipboard watcher)")
		case "subscriber":
			log.Printf("[skeleton] would start subscriber (clipboard writer)")
		default:
			log.Printf("unknown role: %s", role)
		}
	}

	// block here in the skeleton so the binary stays alive during manual tests
	select {}
}
