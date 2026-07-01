# ClipSync v2 architecture (summary)

This document describes the minimal, focused architecture for ClipSync v2
("feature/v2-pubsub"). The goal is a single binary that can act as server,
publisher, subscriber, or any combination, and a compact WebSocket-based
pub/sub protocol.

Goals
- Single executable, composable roles
- Small, well-defined WebSocket protocol (auth/subscribe/publish/direct)
- TLS managed by the binary (configurable cert/key)
- MVP storage: in-memory MemoryStore; optional Bolt/Redis backends later
- Delivery semantics: at-most-once (simple, low-latency)

Project layout in this branch
- cmd/clipsync/      single entrypoint
- pkg/proto/         protocol envelope and message types
- internal/hub/      the pub/sub broker core (skeleton)
- internal/store/    store interface + memory store implementation
- configs/default.yaml
- docs/ARCHITECTURE.md (this file)

Next steps
1. Implement hub client lifecycle, WS handlers, and the v2 protocol (auth/subscribe/publish)
2. Integrate clipboard reader/writer daemons
3. Add bolt_store for local history and admin HTTP endpoints
4. Add RedisStore for multi-instance pub/sub

