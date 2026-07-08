# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ClipSync is a lightweight Windows clipboard synchronization tool based on a Pub/Sub architecture over WebSocket. A single binary can act as server, publisher, subscriber, or any combination via `--roles` flag.

Target environment: Windows 10/11. Text-only, 1MB limit.

## Build Commands

```bash
# Cross-compile for Windows (from Linux/macOS)
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/clipsync.exe ./cmd/clipsync

# Build for current platform
go build -o dist/clipsync ./cmd/clipsync
```

```bash
go test ./internal/... ./pkg/...
go vet ./...
```

## Architecture

```
cmd/clipsync/       # Single binary entrypoint

internal/
  agentws/          # WebSocket client (auth, subscribe, publish, reconnect)
  auth/             # JWT / token validation
  clipboard/        # Platform-specific clipboard read/write (Win32 API)
  hub/              # Pub/sub broker (client registry, channel management)
  net/              # WebSocket server + admin HTTP API
  publisher/        # Clipboard poller + publisher role
  store/            # Storage interface + MemoryStore
  subscriber/       # Message receiver + clipboard writer role

pkg/proto/          # Envelope and message type definitions
```

### Message Flow

```
Publisher polls clipboard → create ClipMessage → publish envelope → WebSocket → Hub
  → Hub validates + Store.SaveHistory + broadcast to subscribers
Subscriber receives publish envelope → extract ClipMessage.Text → write to clipboard
```

### Protocol

JSON envelope format:

```json
{"type":"auth","id":"1","body":{"token":"<token>","client_id":"laptop-1"}}
{"type":"subscribe","id":"2","body":{"channel":"office"}}
{"type":"publish","id":"3","body":{"channel":"office","message":{"message_id":"uuid","timestamp":"RFC3339","origin":"hostname","text":"content"}}}
{"type":"direct","id":"4","body":{"target":"client-A","message":{...}}}
```

### Platform-Specific Code

- `internal/clipboard/clipboard_windows.go` — reads/writes clipboard via `user32.dll`/`kernel32.dll`
- `internal/clipboard/clipboard_linux.go` — uses `xclip`/`xsel`/`wl-paste`

## Configuration

Uses CLI flags and environment variables exclusively. No config file required.

| Env var | Purpose |
|---------|---------|
| `CLIPSYNC_JWT_SECRET` | JWT secret (production) |
| `CLIPSYNC_AUTH_TOKEN` | Static token auth (dev/test) |

See `docs/USAGE.md` for full CLI reference.
