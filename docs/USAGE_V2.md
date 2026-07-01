# ClipSync v2 usage

This document describes how to run the v2 Pub/Sub server and test basic
interactions using a WebSocket client (wscat). The v2 protocol is WebSocket
based and expects JSON "envelope" messages.

Quick run (development)

1. Build the binary:

   go build ./cmd/clipsync -o dist/clipsync

2. Start the server (no TLS):

   ./dist/clipsync --roles server --listen :8080 --ws-path /v2/ws

3. Use wscat to test (install with npm i -g wscat):

   wscat -c ws://127.0.0.1:8080/v2/ws

   Then authenticate and subscribe:

   {"type":"auth","id":"1","body":{"token":"dev-token","client_id":"client-A"}}
   {"type":"subscribe","id":"2","body":{"channel":"office"}}

   In another terminal:

   wscat -c ws://127.0.0.1:8080/v2/ws
   {"type":"auth","id":"1","body":{"token":"dev-token","client_id":"client-B"}}
   {"type":"publish","id":"3","body":{"channel":"office","message":{"message_id":"m1","timestamp":"2026-07-01T00:00:00Z","origin":"client-B","text":"hello"}}}

   Client-A should receive the publish envelope. By default, the server does
   not echo published messages back to the publisher.

Admin HTTP endpoints

The server exposes simple admin endpoints under /api/*:

- GET /api/channels  — list active channels
- POST /api/channels — create a named channel
- GET /api/clients   — list connected client IDs
- GET /api/health    — health check

Security note: these admin endpoints are intentionally minimal. In production
place the service behind a firewall or reverse proxy and enable proper auth.
