# ClipSync

轻量级 Windows 单向剪贴板同步工具（客户端 -> 服务端）。

- 客户端监听本机剪贴板文本变化并推送。
- 服务端接收后输出控制台日志，并提供本地 Web 面板用于查看历史、展开详情和复制。

目标环境：Windows 10 / Windows 11。

## 核心特性

- 单向同步：客户端 -> 服务端
- 仅处理文本剪贴板
- 文本大小上限：1MB
- 服务端内置 Web 面板（无需桌面 GUI 依赖）
- 复制反馈增强（按钮处理中、状态栏、Toast 提示）
- 可选 token 鉴权
- 可选开机自启脚本（Task Scheduler）

## 架构说明

1. 客户端轮询剪贴板文本，检测变更后发送 HTTP POST 到服务端。
2. 服务端校验请求与大小限制，写入控制台日志。
3. 服务端将消息写入内存历史，Web 面板从本地接口读取并展示。
4. 面板支持复制最新消息或指定消息到服务端机器剪贴板。

## 快速开始

### 1. 构建

Windows:

```powershell
scripts\build-windows.bat
```

Linux/macOS 交叉编译 Windows:

```bash
chmod +x scripts/build-windows.sh
./scripts/build-windows.sh
```

输出文件：

- dist/clipsync-server.exe
- dist/clipsync-client.exe

### 1.1 打包发布 ZIP

Linux/macOS:

```bash
chmod +x scripts/package-windows.sh
./scripts/package-windows.sh
```

Windows:

```powershell
scripts\package-windows.bat
```

默认会生成：

- dist/clipsync-windows-amd64.zip

ZIP 内结构示例：

```text
clipsync-windows-amd64/
  clipsync-server.exe
  clipsync-client.exe
  configs/
    server.json
    client.json
  scripts/
    install-autostart-client.ps1
    install-autostart-server.ps1
    uninstall-autostart.ps1
  README.md
  LICENSE
```

### 2. 启动服务端（目标机器）

```powershell
clipsync-server.exe -config .\configs\server.json
```

如果未显式传入 `-config`，程序会按以下顺序自动查找默认配置：

1. 可执行文件所在目录下的 `configs/server.json`
2. 可执行文件同目录下的 `server.json`
3. 当前工作目录下的 `configs/server.json`
4. 当前工作目录下的 `server.json`

默认会自动打开面板：

http://127.0.0.1:8080/panel

### 3. 启动客户端（源机器）

```powershell
clipsync-client.exe -config .\configs\client.json
```

如果未显式传入 `-config`，客户端会按相同规则自动查找 `client.json`。

## 配置文件

### 服务端配置示例

见 [configs/server.json](configs/server.json)

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

### 客户端配置示例

见 [configs/client.json](configs/client.json)

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

## 命令行参数

### 服务端

- -config：JSON 配置文件路径
- -listen：监听地址，默认 :8080
- -token：可选鉴权 token
- -max-bytes：最大接收文本字节数，默认 1048576
- -panel-max-history：Web 面板历史条数，默认 200
- -auto-open-panel：启动时自动打开面板，默认 true
- -notify：可选系统通知，默认 false
- -toast-app-id：通知 AppID，默认 PowerShell
- -notify-self-test：发送启动自检通知，默认 false
- -notify-debug：通知调试日志，默认 false

### 客户端

- -config：JSON 配置文件路径
- -server：服务端地址，默认 http://127.0.0.1:8080/clip
- -token：可选鉴权 token
- -interval：轮询间隔，默认 300ms
- -machine：机器标识（日志显示）
- -max-bytes：最大文本字节数，默认 1048576
- -timeout：请求超时，默认 8s

## Web 面板能力

- 历史列表展示（时间、来源、大小）
- 最新消息默认展开
- 手动展开/收起状态在刷新时保持
- 最新消息变历史时自动收起（若无手动覆盖）
- 复制最新消息 / 复制指定消息
- 复制成功与失败都有明确反馈

## 自动启动脚本

脚本目录：[scripts](scripts)

- [scripts/install-autostart-client.ps1](scripts/install-autostart-client.ps1)
- [scripts/install-autostart-server.ps1](scripts/install-autostart-server.ps1)
- [scripts/uninstall-autostart.ps1](scripts/uninstall-autostart.ps1)
- [scripts/package-windows.bat](scripts/package-windows.bat)
- [scripts/package-windows.sh](scripts/package-windows.sh)

## 开源文档

- [LICENSE](LICENSE)
- [CHANGELOG.md](CHANGELOG.md)
- [SECURITY.md](SECURITY.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)

## 注意事项

- 客户端仅支持 Windows 剪贴板读取。
- 仅同步文本内容，非文本格式会忽略。
- 空文本会忽略。
- 超过 max-bytes 的文本会跳过并记录日志。
- 默认不开启系统通知（notify=false）。
