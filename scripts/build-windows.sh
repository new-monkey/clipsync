#!/usr/bin/env bash
set -euo pipefail

export GOOS=windows
export GOARCH=amd64

mkdir -p dist

go build -trimpath -ldflags "-s -w" -o dist/clipsync-server.exe ./cmd/server
go build -trimpath -ldflags "-s -w" -o dist/clipsync-client.exe ./cmd/client

echo "Build complete: dist/clipsync-server.exe, dist/clipsync-client.exe"
