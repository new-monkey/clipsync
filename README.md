# ClipSync (Windows one-way clipboard sync)

ClipSync is a tiny two-process tool for one-way clipboard sync:

- Client: watches local clipboard text changes and pushes updates.
- Server: receives pushed text and prints it to console for manual viewing/copying.

It is designed for quick delivery and portable deployment on Windows 10/11.

## Features

- One-way sync: client -> server
- Text clipboard only
- Supports long text up to 1MB
- Single-file executables (no runtime dependency installation)
- Optional shared token authentication
- Server includes a built-in web panel for copy, collapse/expand, and history

## Build

### Build on Windows

```powershell
scripts\build-windows.bat
```

### Build on Linux/macOS for Windows

```bash
chmod +x scripts/build-windows.sh
./scripts/build-windows.sh
```

Output files:

- `dist/clipsync-server.exe`
- `dist/clipsync-client.exe`

## Run

### 1) Start server (on destination machine)

```powershell
clipsync-server.exe -listen :8080 -max-bytes 1048576
```

By default, web panel is enabled (no desktop GUI dependency):

- Open in browser: `http://127.0.0.1:8080/panel`
- Button: `复制最新内容`
- History list: latest messages with timestamp/machine/size
- Detail area: newest message is expanded by default; others can `展开` / `收起`

Optional token:

```powershell
clipsync-server.exe -listen :8080 -token your-secret-token
```

### 2) Start client (on source machine)

```powershell
clipsync-client.exe -server http://SERVER_IP:8080/clip -interval 300ms -max-bytes 1048576
```

With token:

```powershell
clipsync-client.exe -server http://SERVER_IP:8080/clip -token your-secret-token
```

## Config File Mode

Both executables support `-config` to load JSON config.

### Server config example

File: `configs/server.json`

```json
{
	"listen_addr": ":8080",
	"token": "",
	"max_clip_bytes": 1048576,
	"panel_max_history": 200,
	"auto_open_panel": true,
	"notify": false,
	"toast_app_id": "PowerShell",
	"notify_self_test": false,
	"notify_debug": false
}
```

Run with config:

```powershell
clipsync-server.exe -config .\configs\server.json
```

Disable notification:

```powershell
clipsync-server.exe -config .\configs\server.json -notify=false
```

Web panel URL (same server port):

```text
http://127.0.0.1:8080/panel
```

Disable auto-open browser:

```powershell
clipsync-server.exe -config .\configs\server.json -auto-open-panel=false
```

Send one startup self-test notification:

```powershell
clipsync-server.exe -config .\configs\server.json -notify-self-test=true
```

Enable verbose notification diagnostics:

```powershell
clipsync-server.exe -config .\configs\server.json -notify-debug=true -notify-self-test=true
```

### Client config example

File: `configs/client.json`

```json
{
	"server_url": "http://127.0.0.1:8080/clip",
	"token": "",
	"interval": "300ms",
	"machine_id": "",
	"max_clip_bytes": 1048576,
	"timeout": "8s"
}
```

Run with config:

```powershell
clipsync-client.exe -config .\configs\client.json
```

You can still override config values using CLI flags. Example:

```powershell
clipsync-client.exe -config .\configs\client.json -server http://SERVER_IP:8080/clip
```

## Windows Auto Start (Task Scheduler)

Build first, then register scheduled tasks by PowerShell scripts.

### Register client at user logon

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-autostart-client.ps1
```

### Register server at user logon

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-autostart-server.ps1
```

### Register server at system startup (usually needs admin)

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-autostart-server.ps1 -AtStartup
```

### Remove scheduled task

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\uninstall-autostart.ps1 -TaskName ClipSyncClient
powershell -ExecutionPolicy Bypass -File .\scripts\uninstall-autostart.ps1 -TaskName ClipSyncServer
```

## Parameters

### Server

- `-config` path to JSON config file
- `-listen` HTTP listening address (default `:8080`)
- `-token` optional shared token
- `-max-bytes` max accepted clipboard text bytes (default `1048576`)
- `-panel-max-history` max history records in web panel (default `200`)
- `-auto-open-panel` auto open web panel in browser on startup (default `true`)
- `-notify` show Windows toast on receive (default `false`)
- `-toast-app-id` toast AppUserModelID (default `PowerShell`)
- `-notify-self-test` show one startup self-test toast (default `false`)
- `-notify-debug` print verbose notification diagnostics (default `false`)

### Client

- `-config` path to JSON config file
- `-server` server endpoint URL (default `http://127.0.0.1:8080/clip`)
- `-token` optional shared token
- `-interval` polling interval (default `300ms`)
- `-machine` optional machine id shown in logs
- `-max-bytes` max clipboard text bytes (default `1048576`)
- `-timeout` request timeout (default `8s`)

## Notes

- The client only supports Windows clipboard reading.
- Non-text clipboard content is ignored.
- Empty text is ignored.
- If text is larger than `-max-bytes`, client skips it and logs a warning.
- Built-in web panel shows incoming messages and supports one-click copy.
- Toast notification path remains optional (`-notify=true`) for troubleshooting.
