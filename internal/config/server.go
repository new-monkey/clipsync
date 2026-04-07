package config

import (
	"encoding/json"
	"os"
	"strings"
)

type ServerConfig struct {
	ListenAddr         string
	Token              string
	MaxClipBytes       int
	PanelMaxHistory    int
	PanelMaxHistorySet bool
	AutoOpenPanel      bool
	AutoOpenPanelSet   bool
	Notify             bool
	NotifySet          bool
	ToastAppID         string
	NotifySelfTest     bool
	NotifySelfTestSet  bool
	NotifyDebug        bool
	NotifyDebugSet     bool
	Mode               string // push, reverse-push, pull
	ClientWSAddr       string // reverse-push模式下A端ws地址
}

type ServerFile struct {
	ListenAddr          *string `json:"listen_addr"`
	Token               *string `json:"token"`
	MaxClipBytes        *int    `json:"max_clip_bytes"`
	PanelMaxHistory     *int    `json:"panel_max_history"`
	LegacyGUIMaxHistory *int    `json:"gui_max_history"`
	AutoOpenPanel       *bool   `json:"auto_open_panel"`
	Notify              *bool   `json:"notify"`
	ToastAppID          *string `json:"toast_app_id"`
	NotifySelfTest      *bool   `json:"notify_self_test"`
	NotifyDebug         *bool   `json:"notify_debug"`
	Mode                *string `json:"mode"`
	ClientWSAddr        *string `json:"client_ws_addr"`
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
	if fileCfg.PanelMaxHistory != nil {
		cfg.PanelMaxHistory = *fileCfg.PanelMaxHistory
		cfg.PanelMaxHistorySet = true
	} else if fileCfg.LegacyGUIMaxHistory != nil {
		cfg.PanelMaxHistory = *fileCfg.LegacyGUIMaxHistory
		cfg.PanelMaxHistorySet = true
	}
	if fileCfg.AutoOpenPanel != nil {
		cfg.AutoOpenPanel = *fileCfg.AutoOpenPanel
		cfg.AutoOpenPanelSet = true
	}
	if fileCfg.Notify != nil {
		cfg.Notify = *fileCfg.Notify
		cfg.NotifySet = true
	}
	if fileCfg.ToastAppID != nil {
		cfg.ToastAppID = strings.TrimSpace(*fileCfg.ToastAppID)
	}
	if fileCfg.NotifySelfTest != nil {
		cfg.NotifySelfTest = *fileCfg.NotifySelfTest
		cfg.NotifySelfTestSet = true
	}
	if fileCfg.NotifyDebug != nil {
		cfg.NotifyDebug = *fileCfg.NotifyDebug
		cfg.NotifyDebugSet = true
	}

	if fileCfg.Mode != nil {
		cfg.Mode = strings.TrimSpace(*fileCfg.Mode)
	}
	if fileCfg.ClientWSAddr != nil {
		cfg.ClientWSAddr = strings.TrimSpace(*fileCfg.ClientWSAddr)
	}
	return cfg, nil
}
