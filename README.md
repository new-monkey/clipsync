# ClipSync


轻量级 Windows 单向剪贴板同步工具，支持多种同步模式：

- push：客户端主动推送到服务端（默认/原有模式）
- reverse-push：A端监听剪贴板，B端主动连接A端，A端通过WebSocket推送（适合目标网络仅允许B->A单向连接场景）
- pull（预留）：B端主动拉取A端剪贴板

- 客户端监听本机剪贴板文本变化并推送。
- 服务端接收后输出控制台日志，并提供本地 Web 面板用于查看历史、展开详情和复制。

目标环境：Windows 10 / Windows 11。

## ClipSync v2 (Pub/Sub)

ClipSync v2 is a redesigned single-binary Pub/Sub architecture that replaces
point-to-point push/pull modes with a central WebSocket-based broker. The v2
protocol is small and centered on channels. Clients authenticate with a JWT
(or development token) and then subscribe and publish JSON envelope messages.

Key points:
- Single binary can act as server, publisher, subscriber, or any combination via --roles flag.
- Transport: WebSocket (wss:// supported; TLS can be configured with --tls-cert/--tls-key).
- Protocol: JSON envelope with types: auth, subscribe, unsubscribe, publish, direct, ping, pong.
- Default delivery semantics: at-most-once (non-blocking sends; messages may be dropped under load).
- Storage: default in-memory (MemoryStore). Bolt/Redis backends are planned.

See docs/ARCHITECTURE.md and docs/USAGE_V2.md for full details and examples.

