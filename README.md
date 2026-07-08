# ClipSync

轻量级 Windows 剪贴板同步工具，基于 WebSocket Pub/Sub 架构。

一个二进制，三种角色：`server` / `publisher` / `subscriber`，通过 `--roles` 组合使用。

- **Server**: WebSocket 服务端 + pub/sub 消息枢纽 + 管理 API
- **Publisher**: 轮询本地剪贴板，SHA256 去重，发布到指定频道
- **Subscriber**: 订阅频道，接收消息并写入本地剪贴板

目标环境：Windows 10/11。纯文本，上限 1MB。

## 快速开始

```bash
# 编译
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/clipsync.exe ./cmd/clipsync

# 启动服务端（公网中转）
set CLIPSYNC_AUTH_TOKEN=my-token
clipsync.exe --roles server --listen :8080

# 启动发布端（主机A，内网）
clipsync.exe --roles publisher --server-url ws://<公网IP>:8080/v2/ws --channel office

# 启动订阅端（主机B，内网）
clipsync.exe --roles subscriber --server-url ws://<公网IP>:8080/v2/ws --channel office
```

详细使用说明见 [docs/USAGE.md](docs/USAGE.md)。
