<div align="center">

# Aurora Signal

**Lightweight WebRTC signaling server built with Go**

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/LessUp/aurora-signal/actions/workflows/ci.yml/badge.svg)](https://github.com/LessUp/aurora-signal/actions/workflows/ci.yml)
[![Docs](https://img.shields.io/badge/Docs-GitHub%20Pages-blue?logo=github)](https://lessup.github.io/aurora-signal/)
[![Go Report Card](https://goreportcard.com/badge/github.com/LessUp/aurora-signal)](https://goreportcard.com/report/github.com/LessUp/aurora-signal)

[简体中文](README.zh-CN.md) | English

</div>

---

A production-ready WebRTC signaling server providing room management, session negotiation (SDP/ICE), and basic media controls (chat/mute). Supports single-instance deployment and horizontal scaling via Redis Pub/Sub, with built-in Prometheus metrics and health checks. Ships with a Web Demo for local testing.

## ✨ Features

- **Room Management** — REST API to create/query rooms with `maxParticipants` cap and automatic empty-room cleanup
- **WebSocket Signaling** — offer / answer / trickle / chat / mute / leave; messages auto-stamped with `id` + `ts` + `version` for traceability
- **Role-Based Access** — Three-tier roles: `viewer` (no media negotiation) / `speaker` / `moderator` (can remote-mute others)
- **Security** — JWT authentication, constant-time Admin Key comparison, per-connection rate limiting, secure response headers
- **Observability** — Prometheus metrics (`signal_*` namespace with `participants` gauge & `message_latency` histogram), structured JSON logging, request logger middleware, Request-ID tracing
- **High Availability** — Redis Pub/Sub multi-node scaling, graceful shutdown (including WebSocket drain), panic recovery middleware
- **Web Demo** — Exponential-backoff reconnect, color-coded connection status, Enter-to-send chat
- **Build** — `ldflags` version injection, OCI labels, Distroless runtime image

## 🚀 Quick Start

```bash
# 1. Set JWT secret
export SIGNAL_JWT_SECRET="dev-secret-change"

# 2. Run the server
make run
# or: go run ./cmd/server

# 3. Open the demo
# Visit http://localhost:8080/demo in your browser
# Enter a room ID and display name, click Join
# Open a second window and repeat to test 1-on-1 calling
```

## 📡 REST API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/rooms` | Create a room |
| GET | `/api/v1/rooms/{id}` | Get room details |
| POST | `/api/v1/rooms/{id}/join-token` | Issue a join token |
| GET | `/api/v1/ice-servers` | ICE server configuration |
| GET | `/healthz` | Liveness probe |
| GET | `/readyz` | Readiness probe (checks Redis) |
| GET | `/metrics` | Prometheus metrics |

**WebSocket** — `GET /ws/v1?token=<JWT>`, first message: `{"type":"join","payload":{"roomId":"..."}}`

Full API documentation → [`docs/API.md`](docs/API.md)

## ⚙️ Configuration

All settings are configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SIGNAL_LOG_LEVEL` | `info` | Log level (`debug` / `info` / `warn` / `error`) |
| `SIGNAL_ADDR` | `:8080` | Listen address |
| `SIGNAL_JWT_SECRET` | — | JWT signing key (**required**) |
| `SIGNAL_ADMIN_KEY` | — | Admin API key (optional) |
| `SIGNAL_ALLOWED_ORIGINS` | — | Allowed origins (comma-separated) |
| `SIGNAL_MAX_MSG_BYTES` | `65536` | Max WebSocket message size (bytes) |
| `SIGNAL_WS_PING_INTERVAL` | `10` | Ping interval (seconds) |
| `SIGNAL_WS_PONG_WAIT` | `25` | Pong timeout (seconds) |
| `SIGNAL_WS_RPS` / `SIGNAL_WS_BURST` | `20` / `40` | Per-connection rate limit |
| `SIGNAL_REDIS_ENABLED` | `false` | Enable Redis scaling |
| `SIGNAL_REDIS_ADDR` | `localhost:6379` | Redis address |

Full list → [`env.example`](env.example)

## 🐳 Docker

```bash
# Build image (with version injection)
docker build --build-arg VERSION=v0.2.0 -t lessup/signaling:v0.2.0 .

# Local orchestration (Redis + coturn included)
cd docker && docker compose up --build
```

## 🧪 Development

```bash
make test          # Unit tests
make test-race     # Race detector
make test-cover    # Coverage report
make vet           # go vet
make lint          # golangci-lint
make build         # Build (with version injection)
```

Load testing: `k6 run k6/ws-smoke.js`

## 📖 Documentation

📘 **Online docs** → [lessup.github.io/aurora-signal](https://lessup.github.io/aurora-signal/)

- [API Reference](https://lessup.github.io/aurora-signal/API.html)
- [Design Document](https://lessup.github.io/aurora-signal/design.html)
- [Changelog](https://lessup.github.io/aurora-signal/changelog.html)
- [Contributing](https://lessup.github.io/aurora-signal/contributing.html)

## 📄 License

[MIT](LICENSE) © LessUp
