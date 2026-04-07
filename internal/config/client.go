package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"clipsync/internal/client"
)

type ClientFile struct {
	ServerURL    *string `json:"server_url"`
	WSListenAddr *string `json:"ws_listen_addr"`
	Token        *string `json:"token"`
	Interval     *string `json:"interval"`
	MachineID    *string `json:"machine_id"`
	MaxClipBytes *int    `json:"max_clip_bytes"`
	Timeout      *string `json:"timeout"`
	Mode         *string `json:"mode"`
}

func LoadClient(path string) (client.Config, error) {
	cfg := client.Config{}
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()

	var fileCfg ClientFile
	if err := dec.Decode(&fileCfg); err != nil {
		return cfg, err
	}

	if fileCfg.ServerURL != nil {
		cfg.ServerURL = strings.TrimSpace(*fileCfg.ServerURL)
	}
	if fileCfg.WSListenAddr != nil {
		cfg.WSListenAddr = strings.TrimSpace(*fileCfg.WSListenAddr)
	}
	if fileCfg.Token != nil {
		cfg.Token = *fileCfg.Token
	}
	if fileCfg.Interval != nil {
		d, err := time.ParseDuration(strings.TrimSpace(*fileCfg.Interval))
		if err != nil {
			return cfg, fmt.Errorf("invalid interval: %w", err)
		}
		cfg.Interval = d
	}
	if fileCfg.MachineID != nil {
		cfg.MachineID = strings.TrimSpace(*fileCfg.MachineID)
	}
	if fileCfg.MaxClipBytes != nil {
		cfg.MaxClipBytes = *fileCfg.MaxClipBytes
	}
	if fileCfg.Timeout != nil {
		d, err := time.ParseDuration(strings.TrimSpace(*fileCfg.Timeout))
		if err != nil {
			return cfg, fmt.Errorf("invalid timeout: %w", err)
		}
		cfg.Timeout = d
	}
	if fileCfg.Mode != nil {
		cfg.Mode = strings.TrimSpace(*fileCfg.Mode)
	}

	return cfg, nil
}
