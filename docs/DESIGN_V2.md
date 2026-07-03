—— ClipSync v2 详细设计文档 ——

1. 目标与背景
- 目标：将 ClipSync 重构为单二进制的中心化 Pub/Sub 架构，使用 WebSocket 作为实时传输通道。单个二进制可以以不同角色（server / publisher / subscriber）运行。
- 理念：保持协议小且稳定（envelope + body），保证低延迟、可替换的存储后端（Memory/Bolt/Redis），并在未来支持跨实例 pub/sub。
- 设计折中：默认 at-most-once 投递（非阻塞发送、有限缓冲、丢弃溢出），为简单缩短延迟与实现复杂度。将来可选“可靠投递”层（ack+retries、message queues）。

2. 总体架构（组件）
- Client (browser/desktop process)
  - 建立 WS 连接，先发 auth，再做 subscribe/publish/direct/ping。
  - 以 JSON Envelope 为消息载体。
- Hub (in-process broker)
  - 管理连接（clientID -> Client），管理 channel 索引（channel -> set(clientID)）。
  - 提供 Publish/DirectSend/Subscribe/Unsubscribe，支持可插拔 Store 后端。
- Store (接口)
  - 抽象接口：用于跨实例广播与历史持久化。实现示例：MemoryStore（MVP）、BoltStore（local persistence）、RedisStore（跨实例 pub/sub）。
- WS Server (HTTP handler)
  - 升级连接并驱动 client 生命周期（reader loop + write pump）。
- Admin HTTP API
  - /api/channels, /api/clients, /api/health, POST /api/channels 等（轻量、后续加 auth）。
- CLI / main
  - 命令行 flags（--listen, --roles, --ws-path, --tls-cert, --tls-key, --config 等）。
- Docs / tests / scripts
  - scripts/wscat_test.sh、unit/integration tests、ARCHITECTURE/USAGE docs。

3. 协议定义（JSON Envelope）
- 通用 Envelope JSON（所有方向统一）：
  - type: string — 消息类型（auth, subscribe, unsubscribe, publish, direct, ping, pong, ack, error）
  - id: string (可选) — 客户端生成的请求 ID（用于 ack/error 关联）
  - body: object (可选) — 类型依赖的 payload

- 关键 body 类型（pkg/proto）
  - AuthBody
    - token: string
    - client_id: string
    示例:
    {"type":"auth","id":"1","body":{"token":"<jwt>","client_id":"laptop-1"}}
  - SubscribeBody / UnsubscribeBody
    - channel: string
    示例:
    {"type":"subscribe","id":"2","body":{"channel":"office"}}
  - PublishBody
    - channel: string
    - message: ClipMessage
  - ClipMessage
    - message_id: string
    - timestamp: string (RFC3339 or ISO8601)
    - origin: string (client_id 或 source)
    - text: string
    - meta?: map[string]string
    示例:
    {"type":"publish","id":"3","body":{"channel":"office","message":{"message_id":"m1","timestamp":"2026-07-01T00:00:00Z","origin":"client-B","text":"hello"}}}
  - DirectBody
    - target: string (target client id)
    - message: ClipMessage
  - Ack/Error
    - ack: {"ref_id": "<id>", "status":"ok"}
    - error: {"ref_id": "<id>", "error":"unauthorized"}

4. Client lifecycle / message flows
- 建连并认证
  1. Client 打开 WS。
  2. Client 发送 auth envelope（token + client_id）。
  3. Server 校验 token（JWT） -> 若 OK，调用 hub.RegisterClient(c)；返回 ack。
- 订阅 / 退订
  - Client 发 subscribe envelope，Server 调用 hub.Subscribe(clientID, channel)，返回 ack。
  - Cancel：unsubscribe -> hub.Unsubscribe。
- 发布（Publish）
  - Publisher 发 publish envelope（channel + message）。
  - Server 做消息校验（channel 非空、message_id 非空、text 非空且 <= MaxMessageTextBytes），将原始 publish envelope JSON 转为 rawMsg，然后：
    - h.store.Publish(channel, rawMsg)
    - h.store.SaveHistory(channel, rawMsg)
    - h.Publish(channel, rawMsg, excludeClientID=publisherID)
  - Server 返回 ack 给 publisher（代表已接收/处理，不代表已成功投递给所有订阅者）。
- 直接发送（Direct）
  - direct -> server 通过 hub.DirectSend(targetClientID, rawMsg) 直接推送给目标客户端（返回 success/fail）。
- 心跳
  - 客户端可发 ping，服务端 send pong (type: "pong" id: refID)。
  - write pump 也会定期发 websocket ping control frames；读 loop 设置 pong handler 来重置 read deadline。
- 断连关闭
  - Reader loop 发生错误或连接关闭，defer 中 UnregisterClient + client.Close()，client.Close 会关闭 send channel 以终止 write pump。

5. Hub 设计细节（并发/数据结构/语义）
- 数据结构（Hub）
  - mu sync.RWMutex
  - conns map[string]*Client           // clientID -> Client
  - channels map[string]map[string]*Client // channel -> clientID -> Client
  - store store.Store (可为 nil)
- 核心行为
  - RegisterClient: mu.Lock() 更新 conns
  - UnregisterClient: mu.Lock() 删除 conns，并从 channels 中移除
  - Subscribe: mu.Lock() 将 client 添加到 channels[channel]
  - Unsubscribe: mu.Lock() 删除订阅
  - CreateChannel / ListChannels / ListClients: 读/写锁保护
- Publish 执行策略（高并发安全）
  - Write to store (best-effort): store.Publish + store.SaveHistory；Hub 忽略这些方法返回的错误（可改为记录日志）。
  - Deliver to local subscribers:
    - h.mu.RLock() 获取 subs map 的引用，h.mu.RUnlock()
    - 遍历 subs：对于每个 subscriber：
      - if id == excludeClientID -> continue
      - non-blocking send: select { case c.send <- append([]byte(nil), rawMsg...): default: // drop & log }
    - 这样确保 Publish 不会因某个慢客户端阻塞整个 Publish 路径 -> at-most-once 投递。
  - DirectSend: 随机读取 conns[target]（RLock），尝试非阻塞写入 c.send。
- Backpressure & Send buffer
  - Client.send chan []byte buffer size: 64 (current). 当缓冲满时，新消息被丢弃并记录日志（log.Printf("... send buffer full; dropping message")）。
  - 可选增强：记录丢弃计数（metrics），或在达到阈值时主动 unsubscribe 或降速。
- 竞态与安全
  - Close(c.send) 由 Client.Close 执行；写方要在写入前保证 send 未被置 nil（当前实现用写 select 匿名操作并 rely on send closed -> 写会 panic）。为避免写入已 close 的 channel，需要确保对 close/send 的并发访问受到保护（当前实现在写方没有同时执行 close，但在高并发下仍需注意）。
  - 建议在 write path 捕获可能的 panic（写 closed channel）并做健壮处理，或把 close 改为发送一个 shutdown signal 而不直接 close(c.send)。

6. Client 实现要点
- 结构体
  - ID string
  - conn *websocket.Conn
  - send chan []byte
  - hub *Hub
  - subs map[string]struct{}
  - mu sync.Mutex
  - authed bool
  - lastSeen time.Time
- Reader loop
  - SetReadLimit(1MB)
  - SetReadDeadline(60s)
  - SetPongHandler to extend deadline
  - for loop: ReadMessage -> json.Unmarshal -> validate -> client.handleMessage
- Write pump
  - select on c.send and pingTicker
  - WriteControl(websocket.PingMessage) every 25s
  - SetWriteDeadline when writing
- Message handlers
  - auth -> ValidateToken -> set c.ID, authed true, hub.RegisterClient
  - subscribe/unsubscribe: ensureAuthed -> hub.Subscribe/Unsubscribe
  - publish: ensureAuthed -> validate body -> hub.Publish(channel, raw, c.ID)
  - direct: ensureAuthed -> validate body -> hub.DirectSend(target, raw)
  - ping -> sendPong
  - sendAck/sendError/sendPong -> push envelope to c.send non-blocking

7. Store 接口设计（internal/store）
- interface Store {
    Publish(channel string, rawMsg []byte) error        // broadcast to backplane
    SubscribeBack(channel string, callback func([]byte)) (unsubscribe func()) // only for stores that support backplane subscriptions (MemoryStore uses goroutine callbacks)
    SaveHistory(channel string, rawMsg []byte) error   // persist message to history
    ListHistory(channel string, limit int) ([][]byte, error)
  }
- MemoryStore（MVP）
  - Publish: iterate over in-memory subscriber callbacks in goroutine (async)
  - SaveHistory: append to ring buffer per-channel (in-memory)
  - SubscribeBack: register callback
  - ListHistory: return buffered history
- BoltStore / RedisStore (future)
  - BoltStore: local disk persistence for history (good for single-instance durability)
  - RedisStore: use Redis Pub/Sub for cross-instance Publish and a Redis list for history (for horizontal scaling)
  - Important: ensure RedisStore.Publish does not block the Hub publish path -> return quickly (use go routine or pipeline)

8. 消息验证与常量
- MaxMessageTextBytes = 1 * 1024 * 1024 (1MB)
- Max envelope size: SetReadLimit to 1MB in WS handler
- Required fields:
  - publish.body.channel != ""
  - publish.body.message.message_id != ""
  - publish.body.message.text != "" and <= MaxMessageTextBytes
  - direct.body.target != ""
- Validation failures -> sendError(refID, reason) and (for auth failures) Close connection

9. 身份认证与授权
- Primary: JWT HS256
  - env: CLIPSYNC_JWT_SECRET
  - ValidateToken: jwt.Parse(..., verify HMAC)
- Dev fallback:
  - If CLIPSYNC_JWT_SECRET unset:
    - If CLIPSYNC_AUTH_TOKEN set — accept exact token string (legacy)
    - Else accept any non-empty token (DEVELOPMENT ONLY)
- Admin endpoints protection:
  - Current: none (assume internal network)
  - Recommended: require JWT Authorization header (Bearer <token>) for admin API; support roles in token claims (e.g., "admin": true)
- Transport: enforce TLS in production (ListenAndServeTLS with cert/key flags)

10. Admin HTTP API
- GET /api/channels -> { channels: ["office", ...] }
- POST /api/channels { channel: "name" } -> 201 created
- GET /api/clients -> { clients: ["client-A"] }
- GET /api/health -> 200 ok
- (Future) GET /api/channels/{name}/history?limit=100 -> returns saved messages (requires SaveHistory implementation)
- (Future) POST /api/clients/{id}/disconnect -> admin-driven disconnect

11. Error handling & logging
- sendError sends type:"error" envelope with body { ref_id, error }
- Hub store errors: currently best-effort & ignored; log store errors in future for observability
- Write failures: log.Printf with client id; close client on write error
- Validation errors -> sendError then continue / close if necessary (auth)

12. Metrics & Observability (建议)
- Counters:
  - ws_connections_total
  - ws_connections_active
  - messages_published_total
  - messages_delivered_local_total
  - messages_dropped_local_total (send buffer full)
  - store_publish_errors_total
- Gauges:
  - hub_subscribers{channel=...}
  - hub_clients_total
- Histograms:
  - publish_latency
- Logs:
  - Structured logs preferred (time, level, client, channel, message_id)
- Tracing:
  - Add trace id injection per envelope for debug (optional future)

13. Testing策略
- 单元测试
  - store (MemoryStore): Publish/SubscribeBack/SaveHistory/ListHistory
  - hub: Subscribe/Unsubscribe/DirectSend/Publish with mocked clients (channels)
  - client: message validation, sendAck/sendError behaviors (unit where possible)
- 集成测试
  - Start server on free port, two WS clients -> auth -> subscribe -> publish -> validate delivery (already implemented)
  - Tests for direct send, unsubscribe, ping/pong timeouts
- 端到端手动验证
  - scripts/wscat_test.sh
- 负载测试
  - small load: N publishers, M subscribers, measure drop rate, CPU, memory
  - high load: measure effects of buffer sizes and decide policy (unsubscribe slow clients or backpressure)
- CI
  - go test ./...
  - golangci-lint (optional)
  - static vet/lint

14. Deployment & Ops
- CLI flags (current)
  - --listen :8080
  - --roles server,publisher,subscriber
  - --ws-path /v2/ws
  - --tls-cert, --tls-key
  - --config <yaml> (future)
- Environment variables
  - CLIPSYNC_JWT_SECRET
  - CLIPSYNC_AUTH_TOKEN (legacy fallback)
- TLS cert management
  - Manual certs via flags
  - Future: integrate ACME/letsencrypt or cert hot reload
- Scaling
  - For single instance: MemoryStore ok
  - For multi-instance:
    - Use RedisStore: publish onto Redis channels; all instances subscribe to Redis pub/sub and re-broadcast to local subscribers
    - Use shared persistence (Redis lists, or central DB) for history
- Running behind reverse proxy (recommended)
  - Run clipsync behind an authenticated reverse proxy for admin endpoints (or add JWT auth to admin endpoints)
- Logging & monitoring
  - Expose Prometheus metrics endpoint (future)
  - Ensure logs include component and client ids

15. Failure modes & recovery
- Slow or stuck clients -> cause message drops
  - Detect high drop rate, log, optionally unsubscribe or mark client as congested
- Hub store errors -> log; if store is blocking, ensure Publish path is non-blocking to avoid blocking hub
- Node crash -> clients reconnect; history in MemoryStore lost -> BoltStore recommended when persistence needed
- Split-brain in multi-instance without backplane -> messages not shared; RedisStore mitigates

16. Backwards compatibility & migration
- v2 is new protocol; clients must be updated to use envelope format
- Provide adapter or compatibility layer if older clients exist (out-of-scope for this PR)
- Documentation should advertise breaking change and migration path

17. Data retention policy (future)
- MemoryStore currently keeps unlimited in-memory history — user asked to “add retention later”.
- Plan: implement retention_days and per-channel size cap:
  - retention_days default 7
  - per-channel max_history_bytes default 10MB
  - Implement in MemoryStore and BoltStore (evict oldest when exceed cap)

18. Security checklist (pre-merge / production)
- [ ] Remove dev fallback that accepts any non-empty token in production builds or gated by config flag
- [ ] Protect /api/* admin endpoints (JWT with admin claim or network ACL)
- [ ] Ensure TLS certs are configured for production
- [ ] Audit logs for sensitive data (do not log plaintext clip text in production if privacy is required)
- [ ] Rate limit publish requests (optional)

19. Review checklist (for PR reviewers)
- Protocol & structs: pkg/proto correctness, JSON field names, example messages
- Auth: internal/auth correctness, JWT parsing & algorithm restrictions (HS256 only)
- Hub concurrency: locking correctness, no data races, robust close semantics
- WS handler: read limits, pong handler, write deadlines, error responses
- Store interface: semantics and memory store tests & edge cases
- Tests: run go test ./..., run integration tests (ws tests), run manual wscat script
- Documentation: docs/USAGE_V2.md and docs/ARCHITECTURE.md clarity
- Security: confirm admin endpoints exposure & JWT behavior documented

20. Roadmap / next PRs (priority order)
- PR#3: BoltStore implementation (local persistence history + list API)
- PR#4: RedisStore + cross-instance Pub/Sub
- PR#5: Admin endpoints auth (JWT with role claims)
- PR#6: Prometheus metrics + /metrics endpoint
- PR#7: Clipboard daemon integration (publisher/subscriber agent), Windows-specific packaging
- PR#8: Reliable delivery mode (optional ack/retry semantics) for users who need stronger guarantees

附录 A — 常用配置与运行示例
- 开发（无 TLS）：
  CLIPSYNC_JWT_SECRET not set (dev fallback accepts dev-token)
  ./dist/clipsync --roles server --listen :8080 --ws-path /v2/ws
- 生产（TLS + JWT）：
  export CLIPSYNC_JWT_SECRET="super-secret"
  ./dist/clipsync --roles server --listen :8443 --ws-path /v2/ws --tls-cert /etc/certs/fullchain.pem --tls-key /etc/certs/privkey.pem

附录 B — 示例 JSON 消息
- Auth:
  {"type":"auth","id":"1","body":{"token":"<jwt>","client_id":"laptop-1"}}
- Subscribe:
  {"type":"subscribe","id":"2","body":{"channel":"office"}}
- Publish:
  {"type":"publish","id":"3","body":{"channel":"office","message":{"message_id":"m1","timestamp":"2026-07-01T00:00:00Z","origin":"client-B","text":"hello"}}}
- Direct:
  {"type":"direct","id":"4","body":{"target":"client-A","message":{"message_id":"d1","timestamp":"...","origin":"client-B","text":"secret"}}}
- Ack/Error:
  {"type":"ack","body":{"ref_id":"3","status":"ok"}}
  {"type":"error","body":{"ref_id":"1","error":"unauthorized"}}

