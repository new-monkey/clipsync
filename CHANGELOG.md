# Changelog

All notable changes to this project will be documented in this file.

## [2.0.0] - 2026-07-08

### Changed
- Complete rewrite: single-binary Pub/Sub architecture over WebSocket.
- One binary (`cmd/clipsync`) replaces former `cmd/server` + `cmd/client`.
- Roles: `server`, `publisher`, `subscriber` — any combination via `--roles`.
- Protocol: JSON envelope over WebSocket (auth/subscribe/publish/direct).
- New internal packages: `hub/`, `agentws/`, `net/`, `store/`, `clipboard/`, `auth/`, `publisher/`, `subscriber/`, `pkg/proto/`.
- Clipboard module unified under `internal/clipboard/` with Win32 and Linux support.

### Removed
- `cmd/server/`, `cmd/client/` — replaced by `cmd/clipsync/`.
- `internal/client/`, `config/`, `protocol/`, `servernotify/`, `serverpanel/`, `ws/`, `wsclient/`, `winclip/`.
- HTTP POST `/clip` push mode, reverse-push mode, Web panel, Windows toast notifications.
- JSON config files (`configs/server.json`, `configs/client.json`), `configs/default.yaml`.
- All build/packaging/autostart scripts under `scripts/`.

## [1.2.0] - 2026-03-13

### Added
- Added built-in Web panel as the primary server UI.
- Added panel copy feedback enhancements: busy state, success/error status, and toast hints.
- Added auto-open browser option for panel startup.

### Changed
- Improved panel expand/collapse behavior:
  - latest message auto-expands,
  - auto-expanded message collapses when it becomes historical,
  - manual expand/collapse state persists across refresh.
- Refined server documentation and launch defaults.

### Removed
- Removed desktop GUI runtime path and related dependencies.

## [1.1.0] - 2026-03-13

### Added
- Added server-side Windows notification flow with optional diagnostics.
- Added config-file loading with CLI override behavior for both server and client.
- Added Windows autostart scripts for client/server.

### Changed
- Improved reliability and diagnostics around notification handling.

## [1.0.0] - 2026-03-13

### Added
- Initial release.
- One-way clipboard text sync (client -> server).
- Client clipboard polling and dedup push logic.
- Server HTTP receiver with payload validation and console logging.
- 1MB text size guard on client and server.
- Windows build scripts and basic run documentation.
