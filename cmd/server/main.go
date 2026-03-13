package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"clipsync/internal/config"
	"clipsync/internal/protocol"
)

func main() {
	cfg := config.ServerConfig{
		ListenAddr:   ":8080",
		MaxClipBytes: 1024 * 1024,
	}

	var configPath string
	var listenAddr string
	var token string
	var maxClipBytes int

	flag.StringVar(&configPath, "config", "", "path to JSON config file")
	flag.StringVar(&listenAddr, "listen", ":8080", "HTTP listen address")
	flag.StringVar(&token, "token", "", "optional shared token")
	flag.IntVar(&maxClipBytes, "max-bytes", 1024*1024, "max clipboard text size in bytes")
	flag.Parse()

	if configPath != "" {
		fileCfg, err := config.LoadServer(configPath)
		if err != nil {
			log.Fatalf("load config failed: %v", err)
		}
		if fileCfg.ListenAddr != "" {
			cfg.ListenAddr = fileCfg.ListenAddr
		}
		if fileCfg.Token != "" {
			cfg.Token = fileCfg.Token
		}
		if fileCfg.MaxClipBytes > 0 {
			cfg.MaxClipBytes = fileCfg.MaxClipBytes
		}
	}

	setFlags := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if setFlags["listen"] {
		cfg.ListenAddr = listenAddr
	}
	if setFlags["token"] {
		cfg.Token = token
	}
	if setFlags["max-bytes"] {
		cfg.MaxClipBytes = maxClipBytes
	}

	if cfg.MaxClipBytes <= 0 {
		log.Fatal("max-bytes must be positive")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/clip", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if cfg.Token != "" && r.Header.Get("X-ClipSync-Token") != cfg.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		maxBody := int64(cfg.MaxClipBytes + 64*1024)
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		defer r.Body.Close()

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		var payload protocol.ClipPayload
		if err := decoder.Decode(&payload); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}

		if err := ensureSingleJSONValue(decoder); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}

		textBytes := len([]byte(payload.Text))
		if textBytes > cfg.MaxClipBytes {
			http.Error(w, "clipboard text too large", http.StatusRequestEntityTooLarge)
			return
		}

		machine := payload.MachineID
		if strings.TrimSpace(machine) == "" {
			machine = "unknown"
		}

		log.Printf("\n=== Clip Received ===\nTime: %s\nMachine: %s\nBytes: %d\nSHA256: %s\n--- BEGIN TEXT ---\n%s\n--- END TEXT ---\n",
			time.Now().Format(time.RFC3339),
			machine,
			textBytes,
			payload.SHA256,
			payload.Text,
		)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("ClipSync server listening on %s", cfg.ListenAddr)
	log.Printf("Receive endpoint: http://<host>%s/clip", cfg.ListenAddr)
	if cfg.Token != "" {
		log.Printf("Token auth enabled")
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	if decoder.More() {
		return fmt.Errorf("multiple json values")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple json values")
		}
		return err
	}
	return nil
}
