# Contributing

Thanks for your interest in contributing to ClipSync.

## Development Setup

1. Install Go (matching version in go.mod or newer).
2. Clone repository.
3. Run:

```bash
go build ./...
```

4. For Windows binaries:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/clipsync-server.exe ./cmd/server
GOOS=windows GOARCH=amd64 go build -o dist/clipsync-client.exe ./cmd/client
```

## Pull Request Guidelines

1. Keep changes focused and small.
2. Preserve current behavior unless change is intentional and documented.
3. Update docs when CLI/config behavior changes.
4. Ensure `go build ./...` passes before opening PR.
5. Add or adjust tests when introducing logic that can regress.

## Commit Message Style

Recommended format:

- `feat: ...`
- `fix: ...`
- `docs: ...`
- `refactor: ...`
- `chore: ...`

## Reporting Bugs

Please include:

1. OS version (Windows 10/11 build info).
2. Command used to run client/server.
3. Relevant logs.
4. Config files (with secrets/token removed).
5. Repro steps and expected behavior.
