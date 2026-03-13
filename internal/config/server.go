package config

import (
	"encoding/json"
	"os"
	"strings"
)

type ServerConfig struct {
	ListenAddr   string
	Token        string
	MaxClipBytes int
}

type ServerFile struct {
	ListenAddr   *string `json:"listen_addr"`
	Token        *string `json:"token"`
	MaxClipBytes *int    `json:"max_clip_bytes"`
}

func LoadServer(path string) (ServerConfig, error) {
	cfg := ServerConfig{}
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()

	var fileCfg ServerFile
	if err := dec.Decode(&fileCfg); err != nil {
		return cfg, err
	}

	if fileCfg.ListenAddr != nil {
		cfg.ListenAddr = strings.TrimSpace(*fileCfg.ListenAddr)
	}
	if fileCfg.Token != nil {
		cfg.Token = *fileCfg.Token
	}
	if fileCfg.MaxClipBytes != nil {
		cfg.MaxClipBytes = *fileCfg.MaxClipBytes
	}

	return cfg, nil
}
