package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"clipsync/internal/config"
	"clipsync/internal/protocol"
	"clipsync/internal/servernotify"
	"clipsync/internal/serverpanel"
)

func main() {
	if len(os.Args) > 1 {
		handled, err := servernotify.HandleProtocolArg(servernotify.NormalizePathFromArgs(os.Args[1]))
		if handled {
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Clipboard updated from notification action")
			return
		}
	}

	cfg := config.ServerConfig{
		ListenAddr:      ":8080",
		MaxClipBytes:    1024 * 1024,
		PanelMaxHistory: 200,
		AutoOpenPanel:   true,
		Notify:          false,
		ToastAppID:      "PowerShell",
		NotifySelfTest:  false,
		NotifyDebug:     false,
	}

	var configPath string
	var listenAddr string
	var token string
	var maxClipBytes int
	var panelMaxHistory int
	var autoOpenPanel bool
	var enableNotify bool
	var toastAppID string
	var notifySelfTest bool
	var notifyDebug bool

	flag.StringVar(&configPath, "config", "", "path to JSON config file")
	flag.StringVar(&listenAddr, "listen", ":8080", "HTTP listen address")
	flag.StringVar(&token, "token", "", "optional shared token")
	flag.IntVar(&maxClipBytes, "max-bytes", 1024*1024, "max clipboard text size in bytes")
	flag.IntVar(&panelMaxHistory, "panel-max-history", 200, "max history records shown in web panel")
	flag.BoolVar(&autoOpenPanel, "auto-open-panel", true, "auto open web panel in browser on startup")
	flag.BoolVar(&enableNotify, "notify", false, "show Windows toast on received clipboard")
	flag.StringVar(&toastAppID, "toast-app-id", "PowerShell", "Windows toast AppUserModelID")
	flag.BoolVar(&notifySelfTest, "notify-self-test", false, "show a startup self-test toast")
	flag.BoolVar(&notifyDebug, "notify-debug", false, "enable verbose notify diagnostics")
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
		if fileCfg.PanelMaxHistorySet {
			cfg.PanelMaxHistory = fileCfg.PanelMaxHistory
		}
		if fileCfg.AutoOpenPanelSet {
			cfg.AutoOpenPanel = fileCfg.AutoOpenPanel
		}
		if fileCfg.NotifySet {
			cfg.Notify = fileCfg.Notify
		}
		if fileCfg.ToastAppID != "" {
			cfg.ToastAppID = fileCfg.ToastAppID
		}
		if fileCfg.NotifySelfTestSet {
			cfg.NotifySelfTest = fileCfg.NotifySelfTest
		}
		if fileCfg.NotifyDebugSet {
			cfg.NotifyDebug = fileCfg.NotifyDebug
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
	if setFlags["panel-max-history"] {
		cfg.PanelMaxHistory = panelMaxHistory
	}
	if setFlags["auto-open-panel"] {
		cfg.AutoOpenPanel = autoOpenPanel
	}
	if setFlags["notify"] {
		cfg.Notify = enableNotify
	}
	if setFlags["toast-app-id"] {
		cfg.ToastAppID = toastAppID
	}
	if setFlags["notify-self-test"] {
		cfg.NotifySelfTest = notifySelfTest
	}
	if setFlags["notify-debug"] {
		cfg.NotifyDebug = notifyDebug
	}

	if cfg.MaxClipBytes <= 0 {
		log.Fatal("max-bytes must be positive")
	}
	if cfg.PanelMaxHistory <= 0 {
		log.Fatal("panel-max-history must be positive")
	}

	if runtime.GOOS != "windows" {
		cfg.Notify = false
		cfg.AutoOpenPanel = false
	}

	if cfg.Notify {
		log.Printf("Toast AppID: %s", cfg.ToastAppID)
		if cfg.NotifyDebug {
			log.Printf("Notify debug mode enabled")
		}
		if err := servernotify.DiagnoseEnvironment(cfg.NotifyDebug); err != nil {
			log.Printf("Notify environment diagnostic failed: %v", err)
		}
		exePath, err := servernotify.ExecutablePath()
		if err != nil {
			log.Printf("Notification disabled: unable to resolve executable path: %v", err)
			cfg.Notify = false
		} else {
			if servernotify.IsLikelyWSLPath(exePath) {
				log.Printf("Warning: executable is running from WSL path (%s)", exePath)
				log.Printf("Warning: Windows toast visibility and protocol callback may be unreliable from WSL path. Prefer running from local Windows path such as C:\\ClipSync\\dist")
			}
			if err := servernotify.EnsureProtocolRegistered(exePath, cfg.NotifyDebug); err != nil {
				log.Printf("Notification enabled but protocol registration failed: %v", err)
			}
		}
		if cfg.NotifySelfTest {
			testText := "ClipSync server startup self-test"
			path, err := servernotify.SaveClipText(testText)
			if err != nil {
				log.Printf("Self-test notification skipped: %v", err)
			} else {
				if err := servernotify.NotifyClipReceived(cfg.ToastAppID, "ClipSync 通知自检", "如果你看到这条通知，说明服务端通知链路可用。", path, cfg.NotifyDebug); err != nil {
					log.Printf("Self-test notification failed: %v", err)
				} else {
					log.Printf("Self-test notification sent")
				}
			}
		}
	}

	mux := http.NewServeMux()
	panel := serverpanel.New(cfg.PanelMaxHistory)
	serverpanel.RegisterHandlers(mux, panel)
	panelURL := fmt.Sprintf("http://127.0.0.1%s/panel", cfg.ListenAddr)
	log.Printf("Web panel: %s", panelURL)
	if cfg.AutoOpenPanel {
		if err := serverpanel.OpenBrowser(panelURL); err != nil {
			log.Printf("Auto open panel failed: %v", err)
		} else {
			log.Printf("Web panel opened in browser")
		}
	}
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

		if cfg.Notify {
			path, err := servernotify.SaveClipText(payload.Text)
			if err != nil {
				log.Printf("Save received clip for copy action failed: %v", err)
			} else {
				title := fmt.Sprintf("收到来自 %s 的剪贴板", machine)
				if err := servernotify.NotifyClipReceived(cfg.ToastAppID, title, payload.Text, path, cfg.NotifyDebug); err != nil {
					log.Printf("Windows toast failed: %v", err)
				}
			}
		}

		panel.Add(machine, textBytes, payload.SHA256, payload.Text)

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
	if cfg.Notify {
		log.Printf("Windows notification enabled")
	}
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
