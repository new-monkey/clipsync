# ClipSync 使用手册

## 目录

- [参数说明](#参数说明)
- [环境变量](#环境变量)
- [部署场景](#部署场景)
  - [场景一：单机测试（全角色）](#场景一单机测试全角色)
  - [场景二：三机中转（A → Server → B）](#场景二三机中转a--server--b)
  - [场景三：直接直连（双机一对一）](#场景三直接直连双机一对一)
  - [场景四：一对多广播](#场景四一对多广播)
- [管理 API](#管理-api)
- [协议说明](#协议说明)
- [常见问题](#常见问题)

---

## 参数说明

### 全局参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--roles` | string | `"server"` | 逗号分隔的角色列表，可选值：`server`,`publisher`,`subscriber` |
| `--config` | string | `""` | 配置文件路径（预留，当前未实现） |

### Server 角色参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--listen` | string | `":8080"` | HTTP/WS 监听地址 |
| `--tls-cert` | string | `""` | TLS 证书文件路径（可选） |
| `--tls-key` | string | `""` | TLS 私钥文件路径（可选） |
| `--ws-path` | string | `"/v2/ws"` | WebSocket 挂载路径 |

### Publisher / Subscriber 公共参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--server-url` | string | `ws://<listen>/<ws-path>` | 服务器 WebSocket 地址 |
| `--channel` | string | `"default"` | 订阅/发布频道名称 |
| `--client-id` | string | `hostname` | 客户端标识 |
| `--auth-token` | string | `""` | 认证令牌（覆盖环境变量） |

### Publisher 独有参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--poll-interval` | duration | `"300ms"` | 剪贴板轮询间隔 |
| `--max-bytes` | int | `1048576` | 最大消息字节数（1MB） |

---

## 环境变量

| 变量 | 说明 | 适用角色 |
|------|------|---------|
| `CLIPSYNC_JWT_SECRET` | JWT HMAC 密钥（生产环境推荐） | server |
| `CLIPSYNC_AUTH_TOKEN` | 静态令牌认证（开发/测试用） | server / publisher / subscriber |

**认证规则**：Server 角色必须设置 `CLIPSYNC_JWT_SECRET`（JWT 验证）或 `CLIPSYNC_AUTH_TOKEN`（简单令牌）。两者都未设置时，server 拒绝启动。Publisher/Subscriber 通过 `--auth-token` 或环境变量传递令牌。

---

## 部署场景

### 场景一：单机测试（全角色）

在一台机器上同时运行所有角色，验证基本功能：

```bash
# 终端 1：启动服务端
set CLIPSYNC_AUTH_TOKEN=dev-token
clipsync.exe --roles server --listen :8080

# 终端 2：启动订阅端
clipsync.exe --roles subscriber --server-url ws://127.0.0.1:8080/v2/ws --channel test --auth-token dev-token

# 终端 3：启动发布端（复制任意文本即可触发推送）
clipsync.exe --roles publisher --server-url ws://127.0.0.1:8080/v2/ws --channel test --auth-token dev-token
```

> 发布端每次检测到剪贴板内容变化（SHA256 不同），会将文本推送到 `test` 频道，订阅端接收后写入本地剪贴板。

---

### 场景二：三机中转（A → Server → B）

你的目标场景：主机 A 在内网发布剪贴板，经过公网 Server 中转，主机 B 在内网接收。

```
┌──────────────┐    WebSocket     ┌──────────────┐    WebSocket     ┌──────────────┐
│  主机 A      │ ────────────────→│  公网 Server  │ ────────────────→│  主机 B      │
│ (Publisher)  │                  │  (Server)     │                  │ (Subscriber) │
│  内网        │                  │  公网 IP      │                  │  内网        │
└──────────────┘                  └──────────────┘                  └──────────────┘
```

```bash
# 公网 Server（中转）
set CLIPSYNC_AUTH_TOKEN=my-secret-token
clipsync.exe --roles server --listen :8080

# 主机 A（发布端，内网能访问公网即可）
clipsync.exe --roles publisher ^
  --server-url ws://<公网IP>:8080/v2/ws ^
  --channel office ^
  --auth-token my-secret-token ^
  --poll-interval 300ms ^
  --client-id "office-pc"

# 主机 B（订阅端，内网能访问公网即可）
clipsync.exe --roles subscriber ^
  --server-url ws://<公网IP>:8080/v2/ws ^
  --channel office ^
  --auth-token my-secret-token ^
  --client-id "home-laptop"
```

**关键条件**：A 和 B 都需要能**主动访问**公网 Server。Server 不需要反向连入内网。

**行为**：
1. A 轮询本地剪贴板（每 300ms），变化时 SHA256 去重后发布
2. Server 接收并广播给 `office` 频道的所有订阅者
3. B 收到消息，SHA256 去重后写入本地剪贴板
4. 如果 Server 重启，A/B 会自动重连（指数退避 1~30s）

---

### 场景三：直接直连（双机一对一）

两台机器直接 WebSocket 连接，无需公网中转。适用于 A/B 在同一内网。

```bash
# 主机 A（同时作为 Server + Publisher）
set CLIPSYNC_AUTH_TOKEN=dev-token
clipsync.exe --roles server,publisher --listen :8080 --channel direct

# 主机 B（Subscriber）
clipsync.exe --roles subscriber ^
  --server-url ws://<A的内网IP>:8080/v2/ws ^
  --channel direct ^
  --auth-token dev-token
```

> A 的 `server` 角色监听 WS 连接，`publisher` 角色轮询剪贴板并发布。B 连接后订阅 `direct` 频道，接收 A 的剪贴板内容。

---

### 场景四：一对多广播

一个发布端，多个订阅端同时接收。

```bash
# Server（公网或内网网关）
set CLIPSYNC_AUTH_TOKEN=broadcast-token
clipsync.exe --roles server --listen :8080

# 发布端（如会议室电脑）
clipsync.exe --roles publisher --server-url ws://<server>:8080/v2/ws --channel meeting-room --auth-token broadcast-token

# 订阅端（多个参会者）
clipsync.exe --roles subscriber --server-url ws://<server>:8080/v2/ws --channel meeting-room --auth-token broadcast-token
clipsync.exe --roles subscriber --server-url ws://<server>:8080/v2/ws --channel meeting-room --auth-token broadcast-token
```

---

## 管理 API

Server 角色内置 HTTP 管理接口，默认挂载在根路径：

| 端点 | 方法 | 认证 | 说明 |
|------|------|------|------|
| `/api/health` | GET | 否 | 健康检查，返回 `"ok"` |
| `/api/channels` | GET | Bearer Token | 返回活跃频道列表 |
| `/api/channels` | POST | Bearer Token | 创建频道，`{"channel":"name"}` |
| `/api/clients` | GET | Bearer Token | 返回已连接客户端列表 |

> 管理 API 的认证使用与 WebSocket 相同的令牌，通过 `Authorization: Bearer <token>` 头传递。

---

## 协议说明

所有通信使用 JSON 信封（Envelope）格式：

```json
{
  "type": "auth | subscribe | unsubscribe | publish | direct | ping | pong | ack | error",
  "id": "请求ID（可选）",
  "body": { "取决于 type 的 payload" }
}
```

### 消息类型

| Type | 方向 | Body | 说明 |
|------|------|------|------|
| `auth` | C→S | `{"token":"...","client_id":"..."}` | 客户端认证 |
| `subscribe` | C→S | `{"channel":"office"}` | 订阅频道 |
| `unsubscribe` | C→S | `{"channel":"office"}` | 退订频道 |
| `publish` | C→S | `{"channel":"office","message":{...}}` | 发布消息 |
| `direct` | C→S | `{"target":"client-A","message":{...}}` | 直接发送 |
| `ack` | S→C | `{"ref_id":"...","status":"ok"}` | 操作确认 |
| `error` | S→C | `{"ref_id":"...","error":"..."}` | 错误响应 |

### ClipMessage 结构

```json
{
  "message_id": "uuid",
  "timestamp": "2026-07-01T00:00:00Z",
  "origin": "hostname",
  "text": "剪贴板内容",
  "meta": { "hash": "sha256" }
}
```

---

## 常见问题

### Q: 发布端需要管理员权限吗？
不需要。Publisher 仅读取剪贴板，普通用户权限即可。

### Q: 订阅端写入剪贴板需要管理员权限吗？
不需要。Windows 剪贴板写入是用户级操作。

### Q: 剪贴板内容有大小限制？
默认 1MB，可通过 `--max-bytes` 调整。超出大小的文本会被跳过。

### Q: 重启后需要重新连接吗？
AgentWS 内置自动重连，断开后指数退避重试（1s→2s→4s→...→30s max），重连后自动重新认证和订阅。

### Q: Server 支持 TLS 吗？
支持。提供 `--tls-cert` 和 `--tls-key` 参数即可启用 WSS 和 HTTPS。

### Q: 如何查看连接状态？
Server 的 `/api/clients` 和 `/api/channels` 端点可以查看当前连接和频道状态。
