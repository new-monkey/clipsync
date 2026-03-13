package main

import (
	"flag"
	"log"
	"time"

	"clipsync/internal/client"
	"clipsync/internal/config"
)

func main() {
	cfg := client.Config{
		ServerURL:    "http://127.0.0.1:8080/clip",
		Interval:     300 * time.Millisecond,
		MaxClipBytes: 1024 * 1024,
		Timeout:      8 * time.Second,
	}

	var configPath string
	var serverURL string
	var token string
	var interval time.Duration
	var machineID string
	var maxClipBytes int
	var timeout time.Duration

	flag.StringVar(&configPath, "config", "", "path to JSON config file")
	flag.StringVar(&serverURL, "server", "http://127.0.0.1:8080/clip", "server endpoint URL")
	flag.StringVar(&token, "token", "", "optional shared token")
	flag.DurationVar(&interval, "interval", 300*time.Millisecond, "clipboard polling interval")
	flag.StringVar(&machineID, "machine", "", "machine id shown in server logs")
	flag.IntVar(&maxClipBytes, "max-bytes", 1024*1024, "max clipboard text size in bytes")
	flag.DurationVar(&timeout, "timeout", 8*time.Second, "http request timeout")
	flag.Parse()

	if configPath == "" {
		configPath = config.ResolveDefaultConfigPath("client.json")
		if configPath != "" {
			log.Printf("Using default client config: %s", configPath)
		}
	}

	if configPath != "" {
		fileCfg, err := config.LoadClient(configPath)
		if err != nil {
			log.Fatalf("load config failed: %v", err)
		}
		mergeClientConfig(&cfg, fileCfg)
	}

	setFlags := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if setFlags["server"] {
		cfg.ServerURL = serverURL
	}
	if setFlags["token"] {
		cfg.Token = token
	}
	if setFlags["interval"] {
		cfg.Interval = interval
	}
	if setFlags["machine"] {
		cfg.MachineID = machineID
	}
	if setFlags["max-bytes"] {
		cfg.MaxClipBytes = maxClipBytes
	}
	if setFlags["timeout"] {
		cfg.Timeout = timeout
	}

	if err := client.Run(cfg); err != nil {
		log.Fatal(err)
	}
}

func mergeClientConfig(dst *client.Config, src client.Config) {
	if src.ServerURL != "" {
		dst.ServerURL = src.ServerURL
	}
	if src.Token != "" {
		dst.Token = src.Token
	}
	if src.Interval > 0 {
		dst.Interval = src.Interval
	}
	if src.MachineID != "" {
		dst.MachineID = src.MachineID
	}
	if src.MaxClipBytes > 0 {
		dst.MaxClipBytes = src.MaxClipBytes
	}
	if src.Timeout > 0 {
		dst.Timeout = src.Timeout
	}
}
