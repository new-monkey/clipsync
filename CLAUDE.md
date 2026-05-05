# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ClipSync is a lightweight Windows clipboard synchronization tool with two sync modes:
- **push (default)**: Client polls clipboard and HTTP POSTs to server
- **reverse-push**: Client runs a WebSocket server, other clients connect and receive broadcasts (for restricted network topologies)

Target environment: Windows 10/11. Text-only, 1MB limit.

## Build Commands

```bash
# Cross-compile for Windows (from Linux/macOS)
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/clipsync-server.exe ./cmd/server
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/clipsync-client.exe ./cmd/client

# Or use the provided scripts
chmod +x scripts/build-windows.sh && ./scripts/build-windows.sh
```

No test suite exists (`go test` will report no test files).

## Architecture

```
cmd/
  server/main.go   # HTTP server + WebSocket client + Web panel
  client/main.go   # Clipboard poller + HTTP pusher + WebSocket server

internal/
  client/           # Platform-specific clipboard read (Windows API)
  config/           # JSON config loading for both binaries
  protocol/         # ClipPayload JSON schema
  servernotify/     # Windows toast notifications
  serverpanel/      # Embedded web panel (HTML+JS) + history REST API
  winclip/          # Platform-specific clipboard write
  ws/               # WebSocket hub (reverse-push mode: A-side broadcasts)
  wsclient/         # WebSocket client (reverse-push mode: B-side receives)
```

### Sync Mode Flow

**push mode**: Client polls clipboard → HTTP POST to server `/clip` → server logs + stores in panel + optional toast

**reverse-push mode**: A-side runs WebSocket server on `:8081`, polls clipboard, broadcasts to all connected B-sides. B-side connects via `ws://<A-ip>:8081/ws` and receives broadcasts.

### Key Data Structures

`protocol.ClipPayload`: `machine_id`, `timestamp`, `text`, `sha256`

`serverpanel.Panel`: thread-safe history buffer with `Add`, `copyLatest`, `copyByID`

`ws.Hub`: concurrent WebSocket connection registry with `Broadcast`

### Platform-Specific Code

Uses Go build tags (`//go:build windows`):
- `internal/client/clipboard_windows.go` — reads Windows clipboard via `user32.dll`/`kernel32.dll`
- `internal/winclip/set_windows.go` — writes Windows clipboard
- `internal/serverpanel/browser_windows.go` — opens browser via `cmd.exe`
- `internal/servernotify/windows.go` — Windows toast notifications via PowerShell

## Configuration

Both server and client load JSON config files (via `-config` flag or default path resolution). CLI flags override config file values.

Server config (`configs/server.json`): `listen_addr`, `token`, `max_clip_bytes`, `panel_max_history`, `auto_open_panel`, `notify`, `mode`, `client_ws_addr`

Client config (`configs/client.json`): `server_url`, `ws_listen_addr`, `token`, `interval`, `machine_id`, `max_clip_bytes`, `timeout`, `mode`
