# AGENTS.md

This file is for coding agents working in `aurora-signal`.

## Purpose

- This repository is a Go WebRTC signaling server with REST, WebSocket, Redis Pub/Sub, metrics, and a demo web UI.
- Prefer small, local changes that match the current structure.
- Keep behavior explicit and operationally safe; this server handles auth, room state, and live network traffic.

## Rule Files Present

- `CLAUDE.md` exists at the repo root and contains repository guidance.
- No `.cursor/rules/` files were found.
- No `.cursorrules` file was found.
- No `.github/copilot-instructions.md` file was found.
- If new Cursor or Copilot rules are added later, merge them into this guidance instead of contradicting them.

## Toolchain

- `go.mod` requires Go `1.23`.
- In the current workspace, `go version` reported `go1.22.0`, and `go test`, `go build`, and `golangci-lint run` failed because the Go 1.23 toolchain was unavailable.
- Before relying on build or test output, ensure Go 1.23 is installed or otherwise available to the Go toolchain.

## Repository Layout

- `cmd/server/main.go`: process entrypoint and graceful shutdown.
- `internal/httpapi/`: HTTP handlers, middleware, WebSocket session handling.
- `internal/room/`: in-memory room and participant management.
- `internal/auth/`: JWT join token signing and parsing.
- `internal/config/`: environment-driven configuration and validation.
- `internal/store/redis/`: Redis Pub/Sub bridge for multi-node fanout.
- `internal/observability/`: Prometheus metrics.
- `internal/signaling/`: message envelope and payload types.
- `web/`: demo frontend.

## Build Commands

- Install and normalize dependencies: `go mod tidy`
- Make target for dependency cleanup: `make deps`
- Fast local compile of the entrypoint: `go build ./cmd/server`
- Release-style build used by the repo: `make build`
- Direct equivalent of the Make target: `go build -trimpath -ldflags "$(...)" -o bin/signal-server ./cmd/server`
- Run locally with env set: `SIGNAL_JWT_SECRET=dev-secret-change go run ./cmd/server`
- Make target for local run: `SIGNAL_JWT_SECRET=dev-secret-change make run`

## Format, Vet, and Lint

- Format all packages: `make fmt`
- Direct formatter path used by the Makefile: `go fmt ./...`
- Vet all packages: `make vet`
- Direct vet command: `go vet ./...`
- Run configured linters: `make lint`
- Direct linter command: `golangci-lint run`
- The repo enables `govet`, `gosimple`, `staticcheck`, `ineffassign`, `typecheck`, `gocritic`, `errcheck`, `unused`, `misspell`, `bodyclose`, `nilerr`, `prealloc`, `gosec`, and `gofmt`.

## Test Commands

- Run all tests: `make test`
- Direct all-tests command: `go test ./...`
- Run with race detector: `make test-race`
- Direct race command: `go test -race ./...`
- Run with coverage artifact: `make test-cover`
- Direct coverage command: `go test -coverprofile=coverage.out ./...`
- Render HTML coverage report: `go tool cover -html=coverage.out -o coverage.html`

## Single-Package and Single-Test Commands

- Run one package: `go test ./internal/room`
- Run one package verbosely: `go test -v ./internal/httpapi`
- Run one exact test: `go test -v ./internal/room -run '^TestRoomLifecycle$'`
- Run one HTTP/WebSocket integration test: `go test -v ./internal/httpapi -run '^TestWSJoinAndLeave$'`
- Run a group of tests by prefix: `go test -v ./internal/httpapi -run '^TestWS'`
- Run one test with race detection: `go test -race ./internal/httpapi -run '^TestWSTwoPeersSignaling$'`
- Prefer exact `^...$` regexes for single-test runs so agents do not accidentally run unrelated tests.

## Formatting Conventions

- Keep all Go files `gofmt`-clean.
- Satisfy `golangci-lint` rather than treating lint as advisory.
- Use tabs and standard Go formatting; do not hand-align code.
- Prefer one blank line between standard-library imports and non-stdlib imports.

## Import Conventions

- Order imports as standard library, third-party, then internal packages.
- Use aliases only when they remove ambiguity or improve clarity.
- Existing example: `redispubsub` is an acceptable alias for `internal/store/redis`.
- Avoid unused imports; the lint config treats them as errors.

## Naming Conventions

- Exported names use PascalCase.
- Unexported names use lowerCamelCase.
- Package names are short, lowercase, and concrete: `room`, `auth`, `config`, `httpapi`.
- Preserve established acronym style: `JWT`, `WS`, `TTL`, `ID`.
- Prefer descriptive receiver names already used in the package, such as `s` for `Server`, `m` for `Manager`, `b` for `Bus`.
- Keep message and config field names aligned with existing JSON names such as `roomId`, `displayName`, and `ttlSeconds`.

## Type Guidelines

- Prefer concrete structs for protocol and config data.
- Use strong named types when the domain benefits, such as `signaling.MessageType`.
- Use `any` only at JSON boundaries, for flexible payload maps, or in small helpers where strong typing would add noise.
- Preserve existing JSON struct tags and omitempty behavior.

## Error Handling

- Prefer early returns for validation failures and exceptional cases.
- Return `error`; do not panic in normal request or message paths.
- Panics are only acceptable for unrecoverable startup failures already following repo patterns.
- When ignoring an error, do it deliberately with `_ = ...` and only for best-effort cleanup or response writes.
- Use `errors.Is` when comparing returned sentinel errors from library calls.

## HTTP and WebSocket Style

- HTTP handlers decode into small local request structs.
- REST handlers use `writeJSON` and `writeError` for consistent responses.
- WebSocket traffic flows through `signaling.Envelope`.
- The first client WebSocket message must remain `join`.
- Validate user input close to the boundary before mutating room state.
- Preserve rate limiting, auth checks, and origin checks when changing connection flow.

## Concurrency and State

- Shared maps are protected with `sync.Mutex` or `sync.RWMutex`.
- Hold locks only for state mutation or snapshotting.
- Do not perform network writes while holding the room manager lock.
- Follow the existing pattern of copying targets under lock and sending after unlock.
- Ensure goroutines have a clear shutdown path via channels, contexts, or connection closure.

## Logging and Metrics

- Use structured `zap` logging with fields, not formatted strings.
- Keep log messages concise and operational.
- Update Prometheus counters and gauges consistently when adding new message or error paths.

## Security and Config

- Configuration is environment-driven under the `SIGNAL_` prefix.
- Call `Config.Validate()` semantics equivalent to the existing startup path when adding new settings.
- Preserve constant-time admin-key comparisons for sensitive checks.
- Do not log JWT secrets, Redis passwords, tokens, or TURN credentials.
- Respect role normalization and TTL clamping helpers in `internal/config`.

## Testing Style

- Use standard `testing` with focused helpers.
- Mark helper functions with `t.Helper()`.
- Prefer `httptest` for HTTP and WebSocket integration coverage.
- Keep tests deterministic with explicit deadlines and bounded timeouts.
- Add tests close to the package being changed.

## When Editing

- Match the existing package layout before creating new files.
- Keep JSON response shapes and signaling message shapes backward compatible unless the task explicitly changes the API.
- If you touch startup, auth, room state, or WebSocket flow, review adjacent tests and add coverage when behavior changes.
- If Go 1.23 is unavailable locally, still update code carefully, but note that build and test verification is blocked by the missing toolchain.
