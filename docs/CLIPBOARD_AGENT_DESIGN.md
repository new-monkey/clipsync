# ClipSync v2 Publisher/Subscriber 剪贴板集成设计文档

## 1. 目标

实现 v2 架构中的 `publisher` 和 `subscriber` 角色，使单二进制能够：
- **Publisher**: 监控本地剪贴板变化，通过 WebSocket 发送到服务器
- **Subscriber**: 接收服务器推送的剪贴板消息，写入本地剪贴板

## 2. 架构设计

### 2.1 组件关系

```
┌─────────────────────────────────────────────────────────────────┐
│                    clipsync binary                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│  │   Server    │    │  Publisher  │    │ Subscriber  │        │
│  │ (WS Server) │    │ (Clipboard  │    │ (Clipboard  │        │
│  │   + Hub     │    │   Watcher)  │    │    Writer)  │        │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘        │
│         │                  │                  │                 │
│         │ WebSocket        │ WebSocket        │ WebSocket       │
│         └──────────────────┼──────────────────┘                 │
│                            ▼                                    │
│                    ┌─────────────┐                              │
│                    │   AgentWS   │                              │
│                    │ (统一 WS    │                              │
│                    │  客户端)    │                              │
│                    └─────────────┘                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐    ┌─────────────┐                            │
│  │ Clipboard   │    │ Clipboard   │                            │
│  │   Reader    │    │   Writer    │                            │
│  │ (Windows)   │    │ (Windows)   │                            │
│  └─────────────┘    └─────────────┘                            │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 核心模块

| 模块 | 职责 | 位置 |
|------|------|------|
| **AgentWS** | 统一的 WebSocket 客户端，处理认证、订阅、消息收发 | `internal/agentws/` |
| **ClipboardReader** | 读取本地剪贴板内容 | `internal/client/clipboard_*.go` |
| **ClipboardWriter** | 写入内容到本地剪贴板 | `internal/winclip/set_*.go` |
| **Publisher** | 监控剪贴板变化并发布到服务器 | `internal/publisher/` |
| **Subscriber** | 接收消息并写入剪贴板 | `internal/subscriber/` |

## 3. 关键接口与类型

### 3.1 AgentWS 接口

```go
type AgentWS struct {
    conn        *websocket.Conn
    clientID    string
    token       string
    wsURL       string
    reconnect   bool
    mu          sync.Mutex
    messageChan chan *proto.Envelope
}

func NewAgentWS(wsURL, clientID, token string) *AgentWS

// 连接到服务器并认证
func (a *AgentWS) Connect() error

// 订阅通道
func (a *AgentWS) Subscribe(channel string) error

// 取消订阅通道
func (a *AgentWS) Unsubscribe(channel string) error

// 发布消息到通道
func (a *AgentWS) Publish(channel string, msg *proto.ClipMessage) error

// 发送直接消息给目标客户端
func (a *AgentWS) DirectSend(targetID string, msg *proto.ClipMessage) error

// 启动消息读取循环，将收到的消息写入 messageChan
func (a *AgentWS) StartReader()

// 关闭连接
func (a *AgentWS) Close() error
```

### 3.2 Publisher 接口

```go
type Publisher struct {
    ws          *AgentWS
    channel     string
    interval    time.Duration
    maxBytes    int
    lastHash    string
    ticker      *time.Ticker
    stopChan    chan struct{}
}

func NewPublisher(ws *AgentWS, channel string, interval time.Duration, maxBytes int) *Publisher

// 启动剪贴板监控
func (p *Publisher) Start()

// 停止监控
func (p *Publisher) Stop()
```

### 3.3 Subscriber 接口

```go
type Subscriber struct {
    ws          *AgentWS
    channel     string
    lastHash    string
    stopChan    chan struct{}
}

func NewSubscriber(ws *AgentWS, channel string) *Subscriber

// 启动消息接收循环
func (s *Subscriber) Start()

// 停止接收
func (s *Subscriber) Stop()
```

## 4. 消息流程

### 4.1 Publisher 流程

```
1. 启动 Publisher
   │
   ▼
2. 创建 AgentWS 并 Connect()
   │
   ▼
3. 发送 auth envelope → 接收 ack
   │
   ▼
4. 发送 subscribe envelope → 接收 ack
   │
   ▼
5. 启动定时器 (interval)，定期轮询剪贴板
   │
   ▼
6. 读取剪贴板内容
   │
   ├── 空内容 → 跳过
   ├── 内容过大 → 跳过并记录日志
   ├── 与上次相同(hash) → 跳过
   └── 新内容 → 继续
       │
       ▼
7. 构建 ClipMessage
   {
     message_id: uuid,
     timestamp: RFC3339,
     origin: clientID,
     text: clipboard_text,
     meta: { "hash": sha256 }
   }
   │
   ▼
8. 发送 publish envelope → 接收 ack
   │
   ▼
9. 更新 lastHash，继续轮询
```

### 4.2 Subscriber 流程

```
1. 启动 Subscriber
   │
   ▼
2. 创建 AgentWS 并 Connect()
   │
   ▼
3. 发送 auth envelope → 接收 ack
   │
   ▼
4. 发送 subscribe envelope → 接收 ack
   │
   ▼
5. 启动消息读取循环
   │
   ▼
6. 接收 publish envelope
   │
   ├── 类型不是 publish → 忽略
   ├── message_id 为空 → 忽略
   ├── text 为空 → 忽略
   ├── 与上次相同(hash) → 忽略
   └── 新消息 → 继续
       │
       ▼
7. 写入剪贴板
   │
   ├── 成功 → 更新 lastHash
   └── 失败 → 记录日志，跳过
```

## 5. 配置参数

### 5.1 CLI 参数扩展

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `--server-url` | string | `ws://localhost:8080` | 服务器 WebSocket 地址 |
| `--channel` | string | `default` | 订阅/发布的通道名 |
| `--client-id` | string | hostname | 客户端标识 |
| `--auth-token` | string | 空 | 认证 token |
| `--poll-interval` | duration | `300ms` | 剪贴板轮询间隔 |
| `--max-bytes` | int | `1048576` | 最大消息字节数 |
| `--reconnect` | bool | true | 断开自动重连 |

### 5.2 环境变量

| 变量 | 说明 |
|------|------|
| `CLIPSYNC_SERVER_URL` | 服务器地址 |
| `CLIPSYNC_CHANNEL` | 通道名 |
| `CLIPSYNC_CLIENT_ID` | 客户端标识 |
| `CLIPSYNC_AUTH_TOKEN` | 认证 token |

## 6. 实现细节

### 6.1 剪贴板读取 (Publisher)

- 使用现有的 `internal/client/clipboard_*.go` 代码
- Windows 使用 user32.dll 直接读取
- 非 Windows 返回错误（后续可扩展）

### 6.2 剪贴板写入 (Subscriber)

- 使用现有的 `internal/winclip/set_*.go` 代码
- Windows 使用 user32.dll 直接写入
- 非 Windows 返回错误（后续可扩展）

### 6.3 消息去重

- 使用 SHA256 哈希比较剪贴板内容
- Publisher: 发送前比较，避免重复发送
- Subscriber: 接收后比较，避免重复写入

### 6.4 自动重连

- AgentWS 在连接断开时自动重连
- 重连间隔指数退避 (1s, 2s, 4s, max 30s)
- 重连后自动重新认证和订阅

### 6.5 错误处理

| 场景 | 处理方式 |
|------|----------|
| 剪贴板读取失败 | 记录日志，继续下一次轮询 |
| 剪贴板写入失败 | 记录日志，跳过当前消息 |
| WebSocket 连接断开 | 自动重连 |
| 认证失败 | 记录错误，退出或重试 |
| 消息发送失败 | 记录日志，继续 |

## 7. 文件结构

```
internal/
├── agentws/
│   ├── agent_ws.go       # AgentWS 实现
│   └── reconnect.go      # 重连逻辑
├── publisher/
│   └── publisher.go      # Publisher 实现
├── subscriber/
│   └── subscriber.go     # Subscriber 实现
├── client/
│   ├── clipboard_windows.go  # Windows 剪贴板读取
│   └── clipboard_other.go    # 非 Windows 占位
└── winclip/
    ├── set_windows.go   # Windows 剪贴板写入
    └── set_other.go     # 非 Windows 占位
```

## 8. 主程序集成

修改 `cmd/clipsync/main.go`：

```go
func main() {
    // ... 解析 flags ...
    
    var wg sync.WaitGroup
    
    for _, role := range roles {
        switch role {
        case "server":
            // 启动 HTTP server (现有逻辑)
        case "publisher":
            wg.Add(1)
            go func() {
                defer wg.Done()
                runPublisher(cfg)
            }()
        case "subscriber":
            wg.Add(1)
            go func() {
                defer wg.Done()
                runSubscriber(cfg)
            }()
        }
    }
    
    wg.Wait()
}
```

## 9. 测试策略

### 9.1 单元测试

| 模块 | 测试用例 |
|------|----------|
| AgentWS | 连接、认证、订阅、发布、消息接收 |
| Publisher | 剪贴板读取、哈希去重、消息构建 |
| Subscriber | 消息接收、哈希去重、剪贴板写入 |

### 9.2 集成测试

- 启动服务器 + Publisher + Subscriber
- 在 Publisher 端设置剪贴板内容
- 验证 Subscriber 端收到并写入剪贴板

## 10. 平台支持

| 平台 | Publisher (读取) | Subscriber (写入) |
|------|------------------|-------------------|
| Windows | ✅ | ✅ |
| macOS | ❌ (待实现) | ❌ (待实现) |
| Linux | ❌ (待实现) | ❌ (待实现) |

## 11. 安全性考虑

- 认证 token 通过环境变量传递，避免命令行暴露
- 支持 TLS 连接到服务器
- 消息内容敏感，建议生产环境启用 TLS
