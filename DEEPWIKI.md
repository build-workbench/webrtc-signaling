# Aurora Signal - 深度代码库分析 (DeepWiki)

## 📋 项目元数据

| 属性 | 值 |
|------|-----|
| 项目名称 | Aurora Signal |
| 项目类型 | WebRTC 信令服务器 |
| 编程语言 | Go 1.21 |
| 许可证 | (查看 LICENSE 文件) |
| 版本 | v1 (基于文档分析) |
| 模块名 | singal |
| 构建工具 | Make, Go Build |
| 容器化 | Docker + Docker Compose |

## 🎯 项目概述

Aurora Signal 是一个基于 Go 语言开发的 WebRTC 信令服务器，提供实时音视频通信所需的房间管理、会话协商（SDP/ICE）和基础控制功能。该项目支持单实例部署和基于 Redis Pub/Sub 的多实例水平扩展，并集成了 Prometheus 指标监控和健康检查。

### 核心功能

- ✅ **房间管理**: 创建、查询、加入/离开房间
- ✅ **会话协商**: SDP Offer/Answer 交换
- ✅ **ICE 候选**: Trickle ICE 机制
- ✅ **实时消息**: 聊天、mute/unmute 控制
- ✅ **JWT 认证**: 基于令牌的 WebSocket 认证
- ✅ **多节点扩展**: Redis Pub/Sub 支持水平扩展
- ✅ **指标监控**: Prometheus 集成
- ✅ **Web Demo**: 内置演示界面

### 非目标（当前版本不包含）

- ❌ 媒体转发/混流（SFU/MCU）
- ❌ 录制/回放功能
- ❌ 端到端加密
- ❌ 复杂聊天室/文件传输

## 📁 项目结构

```
aurora-signal/
├── .github/                    # GitHub 配置
│   ├── ISSUE_TEMPLATE/         # Issue 模板
│   │   ├── bug_report.md
│   │   └── feature_request.md
│   └── pull_request_template.md
├── .git/                       # Git 仓库
├── changelog/                  # 变更日志
├── cmd/                        # 命令入口
│   └── server/
│       └── main.go            # 应用主入口
├── docker/                     # Docker 配置
│   ├── coturn/
│   │   └── turnserver.conf
│   └── docker-compose.yml     # 完整栈部署
├── docs/                       # 文档
│   ├── API.md                 # API 参考
│   └── design.md              # 系统设计文档
├── internal/                   # 内部包
│   ├── auth/                   # 认证模块
│   │   ├── jwt.go
│   │   └── jwt_test.go
│   ├── config/                 # 配置管理
│   │   └── config.go
│   ├── httpapi/                # HTTP API
│   │   ├── server.go          # HTTP 服务器
│   │   ├── utils.go           # 工具函数
│   │   └── ws.go              # WebSocket 处理
│   ├── logger/                 # 日志
│   │   └── logger.go
│   ├── observability/          # 可观测性
│   │   └── metrics.go         # Prometheus 指标
│   ├── room/                   # 房间管理
│   │   ├── manager.go         # 房间管理器
│   │   └── manager_test.go
│   ├── signaling/               # 信令协议
│   │   └── messages.go        # 消息定义
│   └── store/                  # 存储
│       └── redis/
│           └── bus.go         # Redis Pub/Sub
├── k6/                         # 负载测试
│   └── ws-smoke.js            # WebSocket 测试脚本
├── web/                        # Web Demo
│   ├── app.js                 # 前端逻辑
│   ├── index.html             # 演示页面
│   └── style.css              # 样式
├── .editorconfig               # 编辑器配置
├── .gitignore                  # Git 忽略规则
├── .golangci.yml               # Go 代码检查配置
├── CHANGELOG.md                # 变更日志
├── CODE_OF_CONDUCT.md          # 行为准则
├── CONTRIBUTING.md             # 贡献指南
├── Dockerfile                  # Docker 构建文件
├── env.example                 # 环境变量示例
├── go.mod                      # Go 模块定义
├── go.sum                      # 依赖校验
├── LICENSE                     # 许可证
├── Makefile                    # 构建脚本
├── README.md                   # 项目说明
└── SECURITY.md                 # 安全策略
```

## 🏗️ 技术栈

### 核心依赖

```go
// 路由和 HTTP
github.com/go-chi/chi/v5 v5.0.10       // 轻量级 HTTP 路由器

// WebSocket
github.com/gorilla/websocket v1.5.1     // WebSocket 实现

// 认证
github.com/golang-jwt/jwt/v5 v5.2.0     // JWT 令牌处理
github.com/google/uuid v1.6.0           // UUID 生成

// 存储
github.com/redis/go-redis/v9 v9.5.1     // Redis 客户端

// 日志
go.uber.org/zap v1.27.0                 // 结构化日志

// 监控
github.com/prometheus/client_golang v1.19.0  // Prometheus 客户端

// 工具
golang.org/x/time v0.5.0               // 时间工具
```

### 构建和部署

- **Docker**: 多阶段构建，最终镜像使用 distroless
- **Make**: 自动化构建、测试、部署
- **k6**: WebSocket 负载测试
- **golangci-lint**: 代码质量检查

## 🔧 构建系统

### Makefile 任务

| 命令 | 说明 |
|------|------|
| `make all` | 构建项目 |
| `make deps` | 下载依赖 |
| `make build` | 编译二进制文件到 `bin/signal-server` |
| `make run` | 运行服务器（使用默认 JWT 密钥） |
| `make test` | 运行所有测试 |
| `make lint` | 运行代码检查 |
| `make docker-build` | 构建 Docker 镜像 `lessup/signaling:dev` |
| `make compose-up` | 启动完整栈（server + redis + coturn） |
| `make compose-down` | 停止完整栈 |

### Docker 构建

```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache ca-certificates && update-ca-certificates
WORKDIR /src
COPY go.mod .
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/signal-server ./cmd/server

# 运行时阶段
FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /out/signal-server /signal-server
COPY web /web
ENV SIGNAL_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/signal-server"]
```

## 🏛️ 系统架构

### 架构概览图

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Web Browser  │  │ Mobile App  │  │ Desktop App │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
└─────────┼──────────────────┼──────────────────┼─────────────────┘
          │                  │                  │
          │ WebSocket (WSS)  │                  │
          └──────────────────┴──────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Server Layer (Go)                            │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              internal/httpapi/Server                     │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │  │
│  │  │ HTTP Router  │  │ WebSocket    │  │ REST API     │ │  │
│  │  │ (Chi)        │  │ Handler      │  │ Handler      │ │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘ │  │
│  └───────────────────────┬───────────────────────────────────┘  │
│                           │                                     │
│  ┌───────────────────────┴───────────────────────────────────┐  │
│  │              Core Services                               │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │  │
│  │  │   room/  │ │ signaling│ │   auth/  │ │observability│   │  │
│  │  │ Manager  │ │Messages  │ │   JWT    │ │  Metrics  │   │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                           │                                     │
└───────────────────────────┼─────────────────────────────────────┘
                            │
              ┌─────────────┴─────────────┐
              │                           │
              ▼                           ▼
    ┌─────────────────┐         ┌──────────────────┐
    │   Redis (Optional)        │  Prometheus       │
    │   ┌───────────┐ │         │  ┌────────────┐ │
    │   │Pub/Sub   │ │         │  │  Metrics   │ │
    │   │Channel   │ │         │  │  Endpoint  │ │
    │   └───────────┘ │         │  └────────────┘ │
    └─────────────────┘         └──────────────────┘
              │
              ▼
    ┌─────────────────┐
    │  STUN/TURN      │
    │  (coturn)       │
    └─────────────────┘
```

### 组件详细说明

#### 1. HTTP API 层 (internal/httpapi/server.go)

**职责:**
- HTTP/1.1 服务器实现
- Chi 路由器配置
- WebSocket 连接升级
- 静态文件服务（Web Demo）

**路由表:**

| 路径 | 方法 | 处理器 | 说明 |
|------|------|--------|------|
| `/healthz` | GET | handleHealth | 健康检查 |
| `/readyz` | GET | handleReady | 就绪检查 |
| `/metrics` | GET | promhttp.Handler() | Prometheus 指标 |
| `/api/v1/ice-servers` | GET | handleICEServers | 获取 ICE 配置 |
| `/api/v1/rooms` | POST | handleCreateRoom | 创建房间 |
| `/api/v1/rooms/{id}` | GET | handleGetRoom | 获取房间信息 |
| `/api/v1/rooms/{id}/join-token` | POST | handleJoinToken | 签发加入令牌 |
| `/ws/v1` | GET | handleWS | WebSocket 端点 |
| `/demo/*` | GET | FileServer | Web Demo 静态文件 |

**关键字段:**

```go
type Server struct {
    cfg      *config.Config      // 配置
    log      *zap.Logger         // 日志
    rooms    *room.Manager       // 房间管理器
    auth     *auth.JWT           // JWT 认证
    upgrader websocket.Upgrader  // WebSocket 升级器
    httpSrv  *http.Server        // HTTP 服务器
    nodeID   string              // 节点唯一标识
    bus      *redispubsub.Bus    // Redis 总线（可选）
    mu       sync.Mutex          // 互斥锁
    roomSubs map[string]int      // 房间订阅计数
}
```

#### 2. 房间管理器 (internal/room/manager.go)

**职责:**
- 内存中房间状态管理
- 参与者加入/离开处理
- 消息路由（单播/广播）
- 房间生命周期管理

**核心数据结构:**

```go
// 参与者
type Participant struct {
    ID          string    // 参与者唯一ID
    UserID      string    // 业务用户ID
    Role        string    // 角色 (viewer/speaker/moderator)
    DisplayName string    // 显示名称
    Conn        SafeConn  // WebSocket 连接
    JoinedAt    time.Time // 加入时间
}

// 房间
type Room struct {
    ID           string              // 房间ID
    Participants map[string]*Participant // 参与者映射
}

// 管理器
type Manager struct {
    mu    sync.RWMutex          // 读写锁
    rooms map[string]*Room       // 房间映射
    log   *zap.Logger           // 日志
}
```

**关键方法:**

| 方法 | 复杂度 | 说明 |
|------|--------|------|
| `CreateRoom(id)` | O(1) | 创建或获取房间 |
| `Join(roomID, p)` | O(1) | 参与者加入，返回现有成员列表 |
| `Leave(roomID, peerID)` | O(1) | 参与者离开，返回被移除的参与者 |
| `ListPeers(roomID)` | O(n) | 获取房间所有成员 |
| `SendTo(roomID, toPeerID, env)` | O(1) | 发送消息给指定成员 |
| `Broadcast(roomID, excludePeerID, env)` | O(n) | 广播消息给房间成员 |

#### 3. 信令消息 (internal/signaling/messages.go)

**消息类型:**

```go
const (
    TypeJoin      MessageType = "join"              // 加入房间
    TypeJoined    MessageType = "joined"            // 加入成功
    TypeOffer     MessageType = "offer"             // SDP Offer
    TypeAnswer    MessageType = "answer"            // SDP Answer
    TypeTrickle   MessageType = "trickle"           // ICE 候选
    TypeLeave     MessageType = "leave"             // 离开房间
    TypeChat      MessageType = "chat"              // 聊天消息
    TypeMute      MessageType = "mute"              // 静音
    TypeUnmute    MessageType = "unmute"            // 取消静音
    TypeError     MessageType = "error"              // 错误
    TypePeerJoin  MessageType = "participant-joined"  // 成员加入
    TypePeerLeave MessageType = "participant-left"     // 成员离开
)
```

**消息信封:**

```go
type Envelope struct {
    ID      string          `json:"id,omitempty"`      // 消息ID（可选）
    Version string          `json:"version,omitempty"` // 协议版本
    Type    MessageType     `json:"type"`              // 消息类型
    RoomID  string          `json:"roomId,omitempty"`  // 房间ID
    From    string          `json:"from,omitempty"`    // 发送者ID
    To      string          `json:"to,omitempty"`      // 接收者ID (* 或具体ID)
    Ts      int64           `json:"ts,omitempty"`      // 时间戳
    Payload json.RawMessage `json:"payload,omitempty"` // 负载数据
}
```

#### 4. Redis 总线 (internal/store/redis/bus.go)

**职责:**
- 跨节点消息传播
- Pub/Sub 通道管理
- 消息序列化/反序列化

**消息结构:**

```go
type WireMessage struct {
    Kind        MessageKind        `json:"kind"`           // 消息类型 (broadcast/direct)
    RoomID      string             `json:"roomId"`         // 房间ID
    ToPeer      string             `json:"toPeer,omitempty"`      // 目标peer
    ExcludePeer string             `json:"excludePeer,omitempty"` // 排除peer
    Envelope    signaling.Envelope `json:"envelope"`       // 信令消息
    Origin      string             `json:"origin"`         // 源节点ID
}
```

**Pub/Sub 通道命名:**
- 格式: `chan:room:{roomID}`
- 例如: `chan:room:room-001`

#### 5. JWT 认证 (internal/auth/jwt.go)

**令牌结构 (推测):**
```json
{
  "sub": "userId",
  "rid": "roomId",
  "role": "speaker",
  "exp": 1730000000,
  "iat": 1729996400,
  "nbf": 1729996400
}
```

**主要方法:**
- `SignJoinToken()`: 签发加入令牌
- `Verify()`: 验证令牌

#### 6. 配置管理 (internal/config/config.go)

**配置项:**

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `SIGNAL_ADDR` | `:8080` | 监听地址 |
| `SIGNAL_JWT_SECRET` | `dev-secret-change` | JWT 密钥（必填） |
| `SIGNAL_MAX_MSG_BYTES` | `65536` | 最大消息大小 |
| `SIGNAL_WS_PING_INTERVAL` | `10` | 心跳间隔(秒) |
| `SIGNAL_WS_PONG_WAIT` | `25` | 心跳超时(秒) |
| `SIGNAL_WS_RPS` | `20` | 每连接速率限制 |
| `SIGNAL_WS_BURST` | `40` | 突发限制 |
| `SIGNAL_REDIS_ENABLED` | `false` | 启用 Redis |
| `SIGNAL_PROM_ENABLED` | `true` | 启用 Prometheus |

#### 7. 可观测性 (internal/observability/metrics.go)

**Prometheus 指标:**

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `ws_connections` | Gauge | `tenant` | WebSocket 连接数 |
| `rooms` | Gauge | `tenant` | 房间数量 |
| `participants` | Gauge | `tenant` | 参与者数量 |
| `messages_in_total` | Counter | `tenant` | 接收消息总数 |
| `messages_out_total` | Counter | `tenant` | 发送消息总数 |
| `message_bytes_in_total` | Counter | `tenant` | 接收字节总数 |
| `message_bytes_out_total` | Counter | `tenant` | 发送字节总数 |
| `message_latency_ms` | Histogram | `tenant` | 消息延迟 |
| `errors_total` | Counter | `code, tenant` | 错误计数 |

## 📡 协议设计

### WebSocket 消息流

#### 1. 连接建立

```
Client                                Server
  |                                      |
  |-- WebSocket Upgrade ---------------->|
  |   GET /ws/v1?token=<JWT>            |
  |   Authorization: Bearer <JWT>       |
  |<-- 101 Switching Protocols ---------|
  |                                      |
```

#### 2. 加入房间

```
Client                                Server
  |                                      |
  |--- {"version":"v1",                |
  |    "type":"join",                  |
  |    "payload":{"roomId":"room-001", |
  |               "displayName":"Alice"}} -->|
  |                                      |
  |<-- {"version":"v1",                |
  |    "type":"joined",                |
  |    "payload":{"self":{"id":"p-a",  |
  |                        "role":"speaker"}, |
  |             "peers":[{"id":"p-b", |
  |                        "displayName":"Bob"}], |
  |             "iceServers":[...]}} --|
  |                                      |
  |<-- {"version":"v1",                |
  |    "type":"participant-joined",    |
  |    "payload":{"id":"p-c",          |
  |               "displayName":"Charlie"}} --|
```

#### 3. SDP 交换

```
Client A                              Server                              Client B
  |                                     |                                     |
  |--- {"type":"offer",                |                                     |
  |    "to":"p-b",                     |                                     |
  |    "payload":{"sdp":"..."}} ------>|                                     |
  |                                     |-- {"type":"offer",                |
  |                                     |    "from":"p-a",                  |
  |                                     |    "payload":{"sdp":"..."}} ----->|
  |                                     |                                     |
  |                                     |<-- {"type":"answer",               |
  |                                     |    "to":"p-a",                    |
  |                                     |    "payload":{"sdp":"..."}} ------|
  |<-- {"type":"answer",               |                                     |
  |    "from":"p-b",                   |                                     |
  |    "payload":{"sdp":"..."}} -------|                                     |
```

#### 4. ICE 候选交换

```
Client A                              Server                              Client B
  |                                     |                                     |
  |--- {"type":"trickle",              |                                     |
  |    "to":"p-b",                     |                                     |
  |    "payload":{"candidate":{...}}} ->|                                     |
  |                                     |-- {"type":"trickle",              |
  |                                     |    "from":"p-a",                  |
  |                                     |    "payload":{"candidate":{...}}} ->|
  |                                     |                                     |
```

### 消息信封规范

#### 通用字段

- `version`: 协议版本 (默认 "v1")
- `type`: 消息类型
- `id`: 可选的消息ID，用于幂等性
- `ts`: 可选的时间戳

#### 路由字段

- `from`: 发送者peer ID
- `to`: 接收者peer ID ("*" 表示广播)
- `roomId`: 房间ID

#### 有效负载

- `payload`: JSON 格式的特定于消息类型的数据

#### 错误响应

```json
{
  "type": "error",
  "payload": {
    "code": 2003,
    "message": "forbidden",
    "details": "user does not have permission"
  }
}
```

**错误码:**

| 代码 | 名称 | 说明 |
|------|------|------|
| 2001 | invalid_message | 消息格式无效 |
| 2002 | unauthorized | 未认证 |
| 2003 | forbidden | 无权限 |
| 2004 | room_not_found | 房间不存在 |
| 2005 | member_not_found | 成员不存在 |
| 2006 | unsupported_type | 不支持的消息类型 |
| 2007 | rate_limited | 触发速率限制 |
| 2008 | room_full | 房间已满 |
| 2009 | version_mismatch | 版本不匹配 |
| 2010 | bad_state | 状态错误 |
| 3000 | internal_error | 内部错误 |

## 🔐 安全设计

### 认证流程

1. **获取 JWT 令牌**
   ```http
   POST /api/v1/rooms/{roomId}/join-token
   Content-Type: application/json
   X-Admin-Key: <admin_key>

   {
     "userId": "user123",
     "displayName": "Alice",
     "role": "speaker",
     "ttlSeconds": 900
   }
   ```

2. **使用令牌连接 WebSocket**
   ```http
   GET /ws/v1?token=<JWT>
   Authorization: Bearer <JWT>
   ```

### 授权模型

| 角色 | 权限 |
|------|------|
| `viewer` | 仅接收消息，无法发起 offer/answer |
| `speaker` | 可发起 offer/answer/trickle，可聊天 |
| `moderator` | 所有权限 + 可静音他人 + 可踢人 |

### 安全措施

- ✅ **CORS**: Origin 白名单检查
- ✅ **速率限制**: 每连接 20 RPS，突发 40
- ✅ **消息大小限制**: 默认 64KB
- ✅ **心跳检测**: 10秒间隔，25秒超时
- ✅ **JWT 过期**: 可配置 TTL（默认15分钟）
- ✅ **Origin 检查**: WebSocket 升级时验证

## 🚀 部署架构

### 单节点模式 (默认)

```yaml
┌─────────────────────────────┐
│        Signal Server        │
│  ┌───────────────────────┐ │
│  │  - HTTP API           │ │
│  │  - WebSocket          │ │
│  │  - Room Manager       │ │
│  │  - Prometheus Metrics │ │
│  └───────────────────────┘ │
│                             │
│  Memory: Room State          │
│                             │
└─────────────────────────────┘
```

**适用场景:**
- 开发/测试
- 小规模部署（<100 并发连接）
- 单服务器部署

### 多节点模式 (Redis)

```yaml
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  Signal Server  │  │  Signal Server  │  │  Signal Server  │
│       A          │  │       B          │  │       C          │
│                 │  │                 │  │                 │
│  WebSocket      │  │  WebSocket      │  │  WebSocket      │
│  Room Manager   │  │  Room Manager   │  │  Room Manager   │
└────────┬────────┘  └────────┬────────┘  └────────┬────────┘
         │                    │                    │
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │  Redis Cluster   │
                    │                  │
                    │  Pub/Sub         │
                    │  - chan:room:*   │
                    │                  │
                    └──────────────────┘
```

**适用场景:**
- 生产环境
- 高可用性要求
- 水平扩展（>1000 并发连接）

### Docker Compose 部署

```yaml
version: "3.9"
services:
  server:
    build: ..
    image: lessup/signaling:dev
    environment:
      - SIGNAL_JWT_SECRET=dev-secret-change
      - SIGNAL_REDIS_ENABLED=true
      - SIGNAL_REDIS_ADDR=redis:6379
    ports:
      - "8080:8080"
    depends_on:
      - redis
      - coturn

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  coturn:
    image: instrumentisto/coturn:latest
    volumes:
      - ./coturn/turnserver.conf:/etc/coturn/turnserver.conf:ro
    network_mode: bridge
    ports:
      - "3478:3478/tcp"
      - "3478:3478/udp"
      - "49152-49200:49152-49200/udp"
```

**端口映射:**

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| Server | 8080 | TCP | HTTP/WS 服务 |
| Prometheus | 9090 | TCP | 指标抓取 |
| Redis | 6379 | TCP | Redis 服务 |
| coturn | 3478 | TCP/UDP | STUN/TURN |
| coturn | 49152-49200 | UDP | TURN 中继端口 |

## 🧪 测试策略

### 单元测试

**运行命令:**
```bash
go test ./...                    # 运行所有测试
go test ./internal/room/...       # 测试特定包
go test -v ./...                 # 详细输出
go test -cover ./...             # 测试覆盖率
go test -race ./...              # 竞态检测
```

**测试覆盖区域:**
- ✅ 房间管理器 (internal/room/manager_test.go)
- ✅ JWT 认证 (internal/auth/jwt_test.go)
- ⚠️ HTTP API 处理器（需要 mock）
- ⚠️ WebSocket 处理（需要集成测试）

### 集成测试

**k6 WebSocket 测试:**

```javascript
// k6/ws-smoke.js
import ws from 'k6/ws';

export let options = {
  vus: 1,
  duration: '10s',
};

const url = `${base.replace('http', 'ws')}/ws/v1?token=${encodeURIComponent(tok)}`;

export default function () {
  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function () {
      socket.send(JSON.stringify({
        version: 'v1',
        type: 'join',
        payload: { roomId, displayName: 'k6' }
      }));
    });

    socket.on('message', function (data) {
      const m = JSON.parse(data);
      if (m.type === 'joined') {
        socket.send(JSON.stringify({
          version: 'v1',
          type: 'chat',
          payload: { text: 'hello' }
        }));
      }
    });

    socket.setTimeout(function () { socket.close(); }, 5000);
  });

  check(res, { 'ws status 101': (r) => r && r.status === 101 });
}
```

**执行测试:**
```bash
# 基础测试
k6 run k6/ws-smoke.js

# 自定义参数
BASE_URL=http://localhost:8080 k6 run k6/ws-smoke.js
ROOM_ID=test-room k6 run k6/ws-smoke.js

# 高并发测试
k6 run -u 100 -d 60s k6/ws-smoke.js
```

### 预期测试场景

1. **房间生命周期**
   - 创建房间
   - 多客户端加入
   - 客户端离开
   - 空房间清理

2. **信令交换**
   - SDP Offer/Answer
   - ICE Trickle
   - 消息路由

3. **错误处理**
   - 无效令牌
   - 房间不存在
   - 消息格式错误

4. **性能测试**
   - 并发连接数
   - 消息吞吐
   - 内存使用

## 📊 监控指标

### 系统指标

| 指标 | 说明 |
|------|------|
| `go_goroutines` | Goroutine 数量 |
| `go_memstats_alloc_bytes` | 内存分配 |
| `go_memstats_sys_bytes` | 系统内存 |
| `go_gc_duration_seconds` | GC 暂停时间 |

### 业务指标

| 指标 | 描述 |
|------|------|
| `ws_connections` | 当前 WebSocket 连接数 |
| `rooms` | 当前活跃房间数 |
| `participants` | 当前参与者总数 |
| `messages_in_total` | 接收消息总数 |
| `messages_out_total` | 发送消息总数 |
| `message_bytes_in_total` | 接收字节数 |
| `message_bytes_out_total` | 发送字节数 |
| `message_latency_ms` | 消息处理延迟（直方图） |
| `errors_total{code}` | 错误计数（按错误码分组） |

### 告警建议

```yaml
# Prometheus 告警规则
groups:
- name: aurora-signal
  rules:
  - alert: HighErrorRate
    expr: rate(errors_total[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "高错误率"

  - alert: HighMemoryUsage
    expr: message_bytes_out_total > 1000000000
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "高内存使用"

  - alert: ManyConnectionsDropped
    expr: ws_connections < 10
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "连接数过低"
```

## 🔧 配置参考

### 完整环境变量

```bash
# ==== 服务器配置 ====
export SIGNAL_ADDR=":8080"
export SIGNAL_ALLOWED_ORIGINS="https://example.com,https://app.example.com"
export SIGNAL_READ_TIMEOUT="10"
export SIGNAL_WRITE_TIMEOUT="10"
export SIGNAL_MAX_MSG_BYTES="65536"
export SIGNAL_WS_PING_INTERVAL="10"
export SIGNAL_WS_PONG_WAIT="25"

# ==== 安全配置 ====
export SIGNAL_JWT_SECRET="your-secret-key-here"
export SIGNAL_ADMIN_KEY="optional-admin-key"
export SIGNAL_WS_RPS="20"
export SIGNAL_WS_BURST="40"

# ==== Redis 配置 (可选) ====
export SIGNAL_REDIS_ENABLED="true"
export SIGNAL_REDIS_ADDR="redis:6379"
export SIGNAL_REDIS_DB="0"
export SIGNAL_REDIS_PASSWORD=""

# ==== STUN/TURN 配置 ====
export SIGNAL_STUN="stun:stun.l.google.com:19302"
export SIGNAL_TURN_URLS="turn:turn.example.com:3478"
export SIGNAL_TURN_USERNAME="turnuser"
export SIGNAL_TURN_CREDENTIAL="turnpass"
export SIGNAL_TURN_TTL="600"

# ==== 可观测性 ====
export SIGNAL_PROM_ENABLED="true"
export SIGNAL_METRICS_ADDR=":9090"
```

### TURN 服务器配置

```conf
# docker/coturn/turnserver.conf
listening-port=3478
tls-listening-port=5349
fingerprint
lt-cred-mech
realm=example.com
user=turnuser:turnpass
no-stdout-log
log-file=/var/log/turnserver/turn.log
no-tlsv1
no-tlsv1_1
```

## 📚 API 参考

### REST API

#### 创建房间

```http
POST /api/v1/rooms
Content-Type: application/json

{
  "id": "room-001",
  "maxParticipants": 16,
  "metadata": {
    "topic": "meeting"
  }
}
```

**响应:**
```json
HTTP 201
{
  "id": "room-001"
}
```

#### 获取房间信息

```http
GET /api/v1/rooms/room-001
```

**响应:**
```json
HTTP 200
{
  "id": "room-001",
  "participants": 2,
  "maxParticipants": 16,
  "metadata": {
    "topic": "meeting"
  }
}
```

#### 签发加入令牌

```http
POST /api/v1/rooms/room-001/join-token
Content-Type: application/json
X-Admin-Key: <admin_key>

{
  "userId": "user123",
  "displayName": "Alice",
  "role": "speaker",
  "ttlSeconds": 900
}
```

**响应:**
```json
HTTP 200
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 900
}
```

#### 获取 ICE 服务器配置

```http
GET /api/v1/ice-servers
```

**响应:**
```json
HTTP 200
[
  {
    "urls": ["stun:stun.l.google.com:19302"]
  },
  {
    "urls": ["turn:turn.example.com:3478"],
    "username": "turnuser",
    "credential": "turnpass",
    "ttl": 600
  }
]
```

### WebSocket API

#### 连接

```http
GET /ws/v1?token=<JWT>
Authorization: Bearer <JWT>
Upgrade: websocket
Connection: upgrade
```

#### 消息格式

**加入房间:**
```json
{
  "version": "v1",
  "type": "join",
  "payload": {
    "roomId": "room-001",
    "displayName": "Alice",
    "role": "speaker"
  }
}
```

**发送 SDP Offer:**
```json
{
  "version": "v1",
  "type": "offer",
  "to": "peer-b-id",
  "payload": {
    "sdp": "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\n..."
  }
}
```

**发送 ICE 候选:**
```json
{
  "version": "v1",
  "type": "trickle",
  "to": "peer-b-id",
  "payload": {
    "candidate": {
      "candidate": "candidate:1 1 UDP 2122252543 192.168.1.1 12345 typ host",
      "sdpMid": "0",
      "sdpMLineIndex": 0
    }
  }
}
```

**发送聊天消息:**
```json
{
  "version": "v1",
  "type": "chat",
  "to": "peer-b-id",
  "payload": {
    "text": "Hello, World!"
  }
}
```

**静音指令:**
```json
{
  "version": "v1",
  "type": "mute",
  "payload": {
    "target": "peer-b-id"
  }
}
```

#### 接收消息

**加入成功:**
```json
{
  "version": "v1",
  "type": "joined",
  "payload": {
    "self": {
      "id": "peer-a-id",
      "role": "speaker"
    },
    "peers": [
      {
        "id": "peer-b-id",
        "displayName": "Bob",
        "role": "speaker"
      }
    ],
    "iceServers": [
      {
        "urls": ["stun:stun.l.google.com:19302"]
      }
    ]
  }
}
```

**成员加入通知:**
```json
{
  "version": "v1",
  "type": "participant-joined",
  "payload": {
    "id": "peer-c-id",
    "displayName": "Charlie",
    "role": "speaker"
  }
}
```

**成员离开通知:**
```json
{
  "version": "v1",
  "type": "participant-left",
  "payload": {
    "id": "peer-b-id"
  }
}
```

**接收 SDP Offer:**
```json
{
  "version": "v1",
  "type": "offer",
  "from": "peer-b-id",
  "payload": {
    "sdp": "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\n..."
  }
}
```

## 💡 使用示例

### 客户端实现 (JavaScript)

```javascript
class SignalClient {
  constructor(token) {
    this.token = token;
    this.ws = null;
    this.peerId = null;
  }

  connect() {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(`ws://localhost:8080/ws/v1?token=${this.token}`);

      this.ws.onopen = () => {
        console.log('Connected');
        this.send({
          version: 'v1',
          type: 'join',
          payload: {
            roomId: 'room-001',
            displayName: 'Alice',
            role: 'speaker'
          }
        });
      };

      this.ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        this.handleMessage(msg);
      };

      this.ws.onerror = (err) => reject(err);
    });
  }

  send(message) {
    this.ws.send(JSON.stringify(message));
  }

  handleMessage(msg) {
    switch (msg.type) {
      case 'joined':
        this.peerId = msg.payload.self.id;
        console.log('Joined room:', msg.payload);
        break;

      case 'participant-joined':
        console.log('Peer joined:', msg.payload);
        break;

      case 'offer':
        console.log('Received offer:', msg.payload);
        // 处理 SDP Offer
        break;

      case 'answer':
        console.log('Received answer:', msg.payload);
        // 处理 SDP Answer
        break;

      case 'trickle':
        console.log('Received ICE candidate:', msg.payload);
        // 处理 ICE 候选
        break;

      case 'chat':
        console.log('Chat message:', msg.payload);
        break;

      case 'error':
        console.error('Error:', msg.payload);
        break;
    }
  }

  sendOffer(toPeerId, sdp) {
    this.send({
      version: 'v1',
      type: 'offer',
      to: toPeerId,
      payload: { sdp }
    });
  }

  sendAnswer(toPeerId, sdp) {
    this.send({
      version: 'v1',
      type: 'answer',
      to: toPeerId,
      payload: { sdp }
    });
  }

  sendTrickle(toPeerId, candidate) {
    this.send({
      version: 'v1',
      type: 'trickle',
      to: toPeerId,
      payload: { candidate }
    });
  }

  sendChat(toPeerId, text) {
    this.send({
      version: 'v1',
      type: 'chat',
      to: toPeerId,
      payload: { text }
    });
  }

  leave() {
    this.send({
      version: 'v1',
      type: 'leave'
    });
    this.ws.close();
  }
}

// 使用示例
async function main() {
  // 1. 获取令牌
  const token = await getJoinToken('room-001', {
    userId: 'user123',
    displayName: 'Alice',
    role: 'speaker'
  });

  // 2. 连接
  const client = new SignalClient(token);
  await client.connect();

  // 3. 发送 SDP Offer
  const sdp = createOffer();
  client.sendOffer('peer-b-id', sdp);
}

async function getJoinToken(roomId, payload) {
  const res = await fetch(`/api/v1/rooms/${roomId}/join-token`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Admin-Key': 'admin-key'
    },
    body: JSON.stringify(payload)
  });

  const data = await res.json();
  return data.token;
}
```

### 与 WebRTC 集成

```javascript
const pc = new RTCPeerConnection({
  iceServers: iceServersFromSignal
});

// 添加本地媒体
const stream = await navigator.mediaDevices.getUserMedia({
  video: true,
  audio: true
});
stream.getTracks().forEach(track => pc.addTrack(track, stream));

// 处理远程媒体
pc.ontrack = (event) => {
  const [remoteStream] = event.streams;
  document.getElementById('remote-video').srcObject = remoteStream;
};

// 发送 ICE 候选
pc.onicecandidate = (event) => {
  if (event.candidate) {
    client.sendTrickle(toPeerId, event.candidate);
  }
};

// 创建 Offer
async function createOffer() {
  const offer = await pc.createOffer();
  await pc.setLocalDescription(offer);
  return offer;
}

// 接收 Offer
async function handleOffer(fromPeerId, sdp) {
  await pc.setRemoteDescription(new RTCSessionDescription(sdp));
  const answer = await pc.createAnswer();
  await pc.setLocalDescription(answer);
  client.sendAnswer(fromPeerId, answer);
}

// 接收 Answer
async function handleAnswer(sdp) {
  await pc.setRemoteDescription(new RTCSessionDescription(sdp));
}

// 接收 ICE 候选
async function handleTrickle(candidate) {
  try {
    await pc.addIceCandidate(candidate);
  } catch (err) {
    console.error('Error adding ICE candidate:', err);
  }
}
```

## 🐛 已知问题与限制

1. **模块名错误**
   - `go.mod` 中模块名为 `singal`（缺少 'g'）
   - 建议修正为 `signal`

2. **测试覆盖不足**
   - HTTP 处理器缺少单元测试
   - WebSocket 逻辑缺少集成测试

3. **资源清理**
   - 房间为空时自动删除，但缺少 TTL 机制
   - 断线客户端的状态清理依赖心跳

4. **错误码不统一**
   - 部分错误码在代码中硬编码
   - 建议集中定义错误常量

5. **多租户支持**
   - 当前版本未实现租户隔离
   - Redis key 缺少租户前缀

6. **消息幂等性**
   - 消息 ID 可选，但未完全实现去重逻辑
   - 可能导致重复消息处理

## 🔮 未来规划

### v2 计划功能

- [ ] **持久化存储**: PostgreSQL/MongoDB 集成
- [ ] **多租户**: tenantId 隔离和命名空间
- [ ] **房间类型**: 1v1、小型群聊、大型会议室
- [ ] **录制**: WebRTC 媒体录制和存储
- [ ] **转码**: 集成 SFU (Janus/mediasoup)
- [ ] **权限管理**: 细粒度权限控制
- [ ] **审计日志**: 完整操作日志
- [ ] **OpenTelemetry**: 分布式追踪
- [ ] **GraphQL**: 替代 REST API
- [ ] **gRPC**: 高性能内部通信

### 性能优化

- [ ] 连接池优化
- [ ] 消息批处理
- [ ] 零拷贝优化
- [ ] 内存池
- [ ] 垃圾回收调优

## 📖 参考资料

### 官方文档

- [WebRTC 1.0: Real-Time Communication Between Browsers](https://www.w3.org/TR/webrtc/)
- [Interactive Connectivity Establishment (ICE)](https://tools.ietf.org/html/rfc5245)
- [Session Description Protocol (SDP)](https://tools.ietf.org/html/rfc4566)
- [JSON Web Token (JWT)](https://tools.ietf.org/html/rfc7519)

### 相关项目

- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [Chi Router](https://github.com/go-chi/chi)
- [Redis Go Client](https://github.com/redis/go-redis)
- [Zap Logger](https://github.com/uber-go/zap)
- [Prometheus Client](https://github.com/prometheus/client_golang)

### 学习资源

- [WebRTC 入门与实践](https://webrtc.org/getting-started/overview)
- [Go Concurrency Patterns](https://go.dev/doc/effective_go#concurrency)
- [Redis Pub/Sub 指南](https://redis.io/docs/manual/pubsub/)

---

**生成时间**: 2026-01-10
**分析深度**: 深度分析 (DeepWiki)
**文档版本**: v1.0
