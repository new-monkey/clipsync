package main

import (
	"flag"
	"log"
	"time"

	"clipsync/internal/client"
	"clipsync/internal/config"
)

func main() {
	var configPath string
	var mode string
	// push模式参数
	var serverURL string
	var token string
	var interval time.Duration
	var machineID string
	var maxClipBytes int
	var timeout time.Duration
	// reverse-push模式参数
	var wsListenAddr string

	flag.StringVar(&configPath, "config", "", "path to JSON config file")
	flag.StringVar(&mode, "mode", "", "work mode: push or reverse-push")
	// push模式参数
	flag.StringVar(&serverURL, "server", "http://127.0.0.1:8080/clip", "server endpoint URL (push mode)")
	flag.StringVar(&token, "token", "", "optional shared token (push mode)")
	flag.DurationVar(&interval, "interval", 300*time.Millisecond, "clipboard polling interval (push mode)")
	flag.StringVar(&machineID, "machine", "", "machine id shown in server logs (push mode)")
	flag.IntVar(&maxClipBytes, "max-bytes", 1024*1024, "max clipboard text size in bytes (push mode)")
	flag.DurationVar(&timeout, "timeout", 8*time.Second, "http request timeout (push mode)")
	// reverse-push模式参数
	flag.StringVar(&wsListenAddr, "ws-listen", ":8081", "WebSocket listen addr (reverse-push mode)")

	flag.Parse()

	// 先加载配置文件
	var cfg client.Config
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
		cfg = fileCfg
	}

	// 命令行参数覆盖
	setFlags := map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})
	if setFlags["mode"] {
		cfg.Mode = mode
	}

	switch cfg.Mode {
	case "reverse-push":
		if setFlags["ws-listen"] {
			cfg.WSListenAddr = wsListenAddr
		}
		// 只允许reverse-push相关参数生效
		if cfg.WSListenAddr == "" {
			cfg.WSListenAddr = ":8081"
		}
		log.Printf("[reverse-push] listen on %s", cfg.WSListenAddr)
	case "push":
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
		if cfg.ServerURL == "" {
			cfg.ServerURL = "http://127.0.0.1:8080/clip"
		}
		log.Printf("[push] server: %s", cfg.ServerURL)
	default:
		log.Fatalf("unknown mode: %s (must be push or reverse-push)", cfg.Mode)
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
