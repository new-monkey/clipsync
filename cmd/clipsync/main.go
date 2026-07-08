package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"clipsync/internal/agentws"
	"clipsync/internal/hub"
	netw "clipsync/internal/net"
	"clipsync/internal/publisher"
	"clipsync/internal/store"
	"clipsync/internal/subscriber"
)

type AgentConfig struct {
	ServerURL    string
	Channel      string
	ClientID     string
	AuthToken    string
	PollInterval time.Duration
	MaxBytes     int
	Reconnect    bool
}

func main() {
	var config string
	var roles string
	var listen string
	var tlsCert string
	var tlsKey string
	var wsPath string

	var serverURL string
	var channel string
	var clientID string
	var authToken string
	var pollInterval time.Duration
	var maxBytes int

	flag.StringVar(&config, "config", "", "path to config file")
	flag.StringVar(&roles, "roles", "server", "comma-separated roles: server,publisher,subscriber")
	flag.StringVar(&listen, "listen", ":8080", "server listen address")
	flag.StringVar(&tlsCert, "tls-cert", "", "path to TLS cert file (optional)")
	flag.StringVar(&tlsKey, "tls-key", "", "path to TLS key file (optional)")
	flag.StringVar(&wsPath, "ws-path", "/v2/ws", "websocket path to mount")

	flag.StringVar(&serverURL, "server-url", "", "server websocket url (for publisher/subscriber)")
	flag.StringVar(&channel, "channel", "default", "channel name for pub/sub")
	flag.StringVar(&clientID, "client-id", "", "client identifier")
	flag.StringVar(&authToken, "auth-token", "", "authentication token")
	flag.DurationVar(&pollInterval, "poll-interval", 300*time.Millisecond, "clipboard poll interval")
	flag.IntVar(&maxBytes, "max-bytes", 1048576, "max clipboard text size in bytes")

	flag.Parse()

	fmt.Printf("clipsync v2 (feature/v2-pubsub)\n")
	fmt.Printf("config=%s roles=%s listen=%s tls=%t\n", config, roles, listen, (tlsCert != ""))

	hasServerRole := false
	for _, r := range strings.Split(roles, ",") {
		if strings.TrimSpace(r) == "server" {
			hasServerRole = true
			break
		}
	}

	if hasServerRole {
		hasJWTSecret := os.Getenv("CLIPSYNC_JWT_SECRET") != ""
		hasAuthToken := os.Getenv("CLIPSYNC_AUTH_TOKEN") != ""
		if !hasJWTSecret && !hasAuthToken {
			log.Fatal("ERROR: Server requires authentication. Set CLIPSYNC_JWT_SECRET or CLIPSYNC_AUTH_TOKEN environment variable before starting.")
		}
		if !hasJWTSecret {
			log.Printf("WARNING: Using CLIPSYNC_AUTH_TOKEN (legacy mode). For production, use CLIPSYNC_JWT_SECRET with JWT tokens.")
		}
	}

	if clientID == "" {
		host, err := os.Hostname()
		if err == nil && host != "" {
			clientID = host
		} else {
			clientID = "unknown"
		}
	}

	envToken := os.Getenv("CLIPSYNC_AUTH_TOKEN")
	if authToken == "" && envToken != "" {
		authToken = envToken
	}

	if serverURL == "" {
		proto := "ws://"
		if tlsCert != "" && tlsKey != "" {
			proto = "wss://"
		}
		serverURL = proto + listen + wsPath
	}

	agentCfg := AgentConfig{
		ServerURL:    serverURL,
		Channel:      channel,
		ClientID:     clientID,
		AuthToken:    authToken,
		PollInterval: pollInterval,
		MaxBytes:     maxBytes,
		Reconnect:    true,
	}

	st := store.NewMemoryStore()
	h := hub.NewHub(st)

	mux := http.NewServeMux()
	mux.Handle(wsPath, netw.NewWSHandler(h))
	mux.Handle("/", netw.NewAdminHandler(h))

	var wg sync.WaitGroup
	roleList := strings.Split(roles, ",")

	for _, role := range roleList {
		switch strings.TrimSpace(role) {
		case "server":
			log.Printf("starting server on %s (TLS cert provided=%v)", listen, tlsCert != "")
			wg.Add(1)
			go func() {
				defer wg.Done()
				srv := &http.Server{Addr: listen, Handler: mux}
				if tlsCert != "" && tlsKey != "" {
					log.Printf("listening (TLS) on %s", listen)
					if err := srv.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
						log.Fatalf("server exited: %v", err)
					}
				} else {
					log.Printf("listening (plain) on %s", listen)
					if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
						log.Fatalf("server exited: %v", err)
					}
				}
			}()
		case "publisher":
			log.Printf("starting publisher (clipboard watcher), channel=%s, interval=%s", channel, pollInterval)
			wg.Add(1)
			go func() {
				defer wg.Done()
				runPublisher(agentCfg)
			}()
		case "subscriber":
			log.Printf("starting subscriber (clipboard writer), channel=%s", channel)
			wg.Add(1)
			go func() {
				defer wg.Done()
				runSubscriber(agentCfg)
			}()
		default:
			log.Printf("unknown role: %s", role)
		}
	}

	wg.Wait()
}

func runPublisher(cfg AgentConfig) {
	ws := agentws.NewAgentWS(cfg.ServerURL, cfg.ClientID, cfg.AuthToken, cfg.Reconnect)

	if err := ws.Connect(); err != nil {
		log.Printf("publisher connect failed: %v", err)
		return
	}
	defer ws.Close()

	if err := ws.Authenticate(); err != nil {
		log.Printf("publisher auth failed: %v", err)
		return
	}

	if err := ws.Subscribe(cfg.Channel); err != nil {
		log.Printf("publisher subscribe failed: %v", err)
		return
	}

	ws.StartReader()
	ws.StartHeartbeat(25 * time.Second)

	p := publisher.NewPublisher(ws, cfg.Channel, cfg.PollInterval, cfg.MaxBytes)
	p.Start()
}

func runSubscriber(cfg AgentConfig) {
	ws := agentws.NewAgentWS(cfg.ServerURL, cfg.ClientID, cfg.AuthToken, cfg.Reconnect)

	if err := ws.Connect(); err != nil {
		log.Printf("subscriber connect failed: %v", err)
		return
	}
	defer ws.Close()

	if err := ws.Authenticate(); err != nil {
		log.Printf("subscriber auth failed: %v", err)
		return
	}

	if err := ws.Subscribe(cfg.Channel); err != nil {
		log.Printf("subscriber subscribe failed: %v", err)
		return
	}

	ws.StartReader()
	ws.StartHeartbeat(25 * time.Second)

	s := subscriber.NewSubscriber(ws, cfg.Channel)
	s.Start()
}
