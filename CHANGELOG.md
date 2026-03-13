# Changelog

All notable changes to this project will be documented in this file.

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
