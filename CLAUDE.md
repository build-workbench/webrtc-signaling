# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **aurora-signal**, a WebRTC signaling server written in Go. It provides room management, session negotiation (SDP/ICE), and basic controls (chat/mute) for real-time audio/video communication. The project supports single-instance and Redis Pub/Sub-based multi-instance horizontal scaling, with Prometheus metrics and health checks integrated. A web demo is included for local testing.

**Main technologies:**
- Go 1.21
- WebSocket + JSON for signaling
- Redis (optional) for multi-node communication
- Prometheus for metrics
- Gorilla WebSocket for WebSocket handling
- Chi for HTTP routing

## Common Commands

### Development
```bash
# Install dependencies
go mod tidy

# Build the server
make build
# or
go build -o bin/signal-server ./cmd/server

# Run the server (with default JWT secret)
make run
# or
SIGNAL_JWT_SECRET=dev-secret-change go run ./cmd/server

# Run tests
make test
# or
go test ./...

# Run a specific test
go test -v ./internal/room/... -run TestManager

# Run linting (requires golangci-lint)
make lint
# or
golangci-lint run
```

### Docker & Deployment
```bash
# Build Docker image
make docker-build
# or
docker build -t lessup/signaling:dev .

# Start full stack (server + redis + coturn)
make compose-up
# or
cd docker && docker compose up --build -d

# Stop the stack
make compose-down
# or
cd docker && docker compose down
```

### Load Testing
```bash
# Run k6 WebSocket smoke test
k6 run k6/ws-smoke.js

# With custom base URL
BASE_URL=http://localhost:8080 k6 run k6/ws-smoke.js

# With custom room ID
ROOM_ID=my-room k6 run k6/ws-smoke.js
```

## Environment Configuration

Required environment variables:
- `SIGNAL_JWT_SECRET`: JWT signing secret (required)

Common optional variables (see `env.example` for full list):
- `SIGNAL_ADDR`: Server listen address (default: `:8080`)
- `SIGNAL_ALLOWED_ORIGINS`: Comma-separated CORS whitelist
- `SIGNAL_REDIS_ENABLED`: Enable Redis for multi-node (default: `false`)
- `SIGNAL_REDIS_ADDR`: Redis address (default: `redis:6379`)
- `SIGNAL_STUN`: STUN servers (default: `stun:stun.l.google.com:19302`)
- `SIGNAL_TURN_URLS`, `SIGNAL_TURN_USERNAME`, `SIGNAL_TURN_CREDENTIAL`: TURN configuration
- `SIGNAL_PROM_ENABLED`: Enable Prometheus metrics (default: `true`)

## High-Level Architecture

### Core Components

```
┌─────────────────────────────────────────┐
│           cmd/server/main.go            │
│              (Entry Point)             │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│       internal/httpapi/Server          │
│  ┌─────────────────────────────────┐   │
│  │ HTTP Server (Chi Router)        │   │
│  │  - REST API (/api/v1/*)         │   │
│  │  - WebSocket (/ws/v1)           │   │
│  │  - Web Demo (/demo/*)           │   │
│  └─────────────────────────────────┘   │
└──────────────┬──────────────────────────┘
               │
               ▼
┌──────────────┬──────────────┬─────────────┐
│              │              │             │
▼              ▼              ▼             ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐
│  room/   │ │ signaling│ │   auth/  │ │ observability│
│ Manager  │ │Messages  │ │   JWT    │ │   Metrics    │
└──────────┘ └──────────┘ └──────────┘ └──────────────┘
               │
               ▼
       ┌──────────────┐
       │ store/redis/ │
       │     Bus     │
       │ (Pub/Sub)   │
       └──────────────┘
```

### Key Components

1. **cmd/server/main.go:17-44**: Application entry point. Initializes config, logger, room manager, JWT auth, and HTTP server with graceful shutdown.

2. **internal/room/Manager:32-152**: Core state management. Maintains in-memory room and participant data with mutex protection. Handles:
   - `CreateRoom()`: Creates or returns existing room
   - `Join()`: Adds participant to room, returns peer list
   - `Leave()`: Removes participant, cleans up empty rooms
   - `SendTo()`: Direct message to specific peer
   - `Broadcast()`: Message to all room participants

3. **internal/httpapi/Server:23-192**: HTTP and WebSocket server. Manages:
   - REST endpoints for room/ICE/token management
   - WebSocket connections with origin checking
   - Demo web interface
   - Redis integration for multi-node (if enabled)

4. **internal/signaling/messages.go:5-56**: Message protocol definitions. Defines:
   - Message types: join, joined, offer, answer, trickle, leave, chat, mute, unmute, error
   - Envelope structure with routing fields (to/from/roomId)
   - Payload types for different message kinds

5. **internal/store/redis/Bus:31-110**: Redis Pub/Sub for multi-node communication:
   - `PublishBroadcast()`: Room-wide messages
   - `PublishDirect()`: Peer-to-peer messages
   - `SubscribeRoom()`: Receives cross-node messages
   - Automatically ignores messages from own node

6. **internal/config/config.go:62-147**: Configuration loading from environment variables with sensible defaults (see Environment Configuration section).

7. **internal/auth/jwt.go**: JWT token signing and verification for WebSocket authentication.

8. **internal/observability/metrics.go**: Prometheus metrics collection (ws_connections, rooms, participants, messages, errors).

### Message Flow

```
Client A                    Server                     Client B
   │                        │                             │
   │───WebSocket Connect───>│                             │
   │  /ws/v1?token=<JWT>    │                             │
   │                        │                             │
   │───join(roomId)────────>│                             │
   │                        │───joined+peers─────────────>│
   │                        │                             │
   │                        │<─────join(roomId)────────────│
   │<────participant-joined─┤                             │
   │                        │───joined+peers─────────────>│
   │                        │                             │
   │───offer(to=B)─────────>│                             │
   │                        │───offer(from=A)───────────>│
   │                        │                             │
   │                        │<────answer(to=A)────────────│
   │<────answer(from=B)─────┤                             │
   │                        │                             │
   │───trickle(to=B)───────>│                             │
   │                        │───trickle(from=A)─────────>│
   │                        │                             │
```

### Deployment Modes

**Single Node (default):**
- No Redis required
- All room state in memory
- Only clients on same node can communicate

**Multi-Node (Redis enabled):**
- Set `SIGNAL_REDIS_ENABLED=true`
- Room state synced via Redis Pub/Sub
- Clients on different nodes can join same room
- Horizontal scaling supported

### API Endpoints

**REST API** (`/api/v1`):
- `POST /rooms` - Create room
- `GET /rooms/{id}` - Get room info
- `POST /rooms/{id}/join-token` - Issue JWT join token
- `GET /ice-servers` - Get STUN/TURN config

**WebSocket** (`/ws/v1`):
- Connect with `?token=<JWT>`
- First message must be `join` with room details
- Messages: join, offer, answer, trickle, chat, mute, leave

**Health & Metrics**:
- `GET /healthz` - Liveness check
- `GET /readyz` - Readiness check
- `GET /metrics` - Prometheus metrics

## Key Files

- **cmd/server/main.go**: Application bootstrap
- **internal/httpapi/server.go**: HTTP/WS server with routing
- **internal/httpapi/ws.go**: WebSocket connection handling
- **internal/room/manager.go**: Room/participant state management
- **internal/signaling/messages.go**: Protocol definitions
- **internal/store/redis/bus.go**: Redis Pub/Sub bridge
- **internal/auth/jwt.go**: JWT authentication
- **internal/config/config.go**: Configuration management
- **web/**: Built-in demo web interface (HTML/JS/CSS)
- **docker/docker-compose.yml**: Full stack deployment
- **k6/ws-smoke.js**: WebSocket load testing script

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package
go test ./internal/room/...

# Run with verbose output
go test -v ./...

# Run with race detector
go test -race ./...
```

## Development Notes

- The module name is `singal` (note: likely a typo for "signal")
- WebSocket connections are upgrade from HTTP using gorilla/websocket
- Room cleanup happens automatically when last participant leaves
- Redis is optional - the server works standalone
- Demo web interface available at `/demo` when server is running
- JWT tokens expire (default 900 seconds) and must be refreshed
- Message size limit: 64KB by default (configurable)
- Rate limiting: 20 RPS per connection, burst 40

## Documentation

- **docs/design.md**: Comprehensive system design document with architecture, protocol, security, and deployment details
- **docs/API.md**: API reference for REST and WebSocket endpoints
- **README.md**: Quick start guide and configuration reference
- **CONTRIBUTING.md**: Contribution guidelines
