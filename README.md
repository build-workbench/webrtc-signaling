<div align="center">

# Aurora Signal

**轻量级 WebRTC 信令服务，基于 Go 构建**

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/AICL-Lab/aurora-signal/actions/workflows/ci.yml/badge.svg)](https://github.com/AICL-Lab/aurora-signal/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/AICL-Lab/aurora-signal)](https://goreportcard.com/report/github.com/AICL-Lab/aurora-signal)

</div>

---

一个生产就绪的 WebRTC 信令服务，提供房间管理、会话协商（SDP/ICE）与基础媒体控制（聊天/静音）。支持单实例部署与 Redis Pub/Sub 多实例水平扩展，内置 Prometheus 指标与健康检查，附带 Web Demo 便于本地验证。

## ✨ 特性

- **房间管理** — REST API 创建/查询房间，支持 `maxParticipants` 人数上限，空房间自动清理
- **WebSocket 信令** — offer / answer / trickle / chat / mute / leave，消息自动填充 `id` + `ts` + `version` 便于追踪
- **角色权限** — 三级角色：`viewer`（不可发起媒体协商）/ `speaker` / `moderator`（可远程静音他人）
- **安全** — JWT Token 认证、Admin Key 常量时间比较、每连接速率限制、安全响应头
- **可观测性** — Prometheus 指标（`signal_*` 命名空间，含 `participants` gauge 和 `message_latency_seconds` histogram）、结构化 JSON 日志、请求日志中间件、Request-ID 链路追踪
- **高可用** — Redis Pub/Sub 多节点扩展、优雅关闭（含 WebSocket 连接排空）、panic recovery 中间件
- **Web Demo** — 断线指数退避重连、连接状态颜色指示、Enter 发送聊天
- **构建** — `ldflags` 版本注入、OCI 标签、Distroless 运行时镜像

## 🚀 快速开始

```bash
# 1. 设置 JWT Secret
export SIGNAL_JWT_SECRET="change-me-to-a-long-random-secret"

# 2. 运行服务
make run
# 或：go run ./cmd/server

# 3. 打开 Demo
# 浏览器访问 http://localhost:8080/demo
# 输入房间 ID 和显示名，点击加入
# 新开窗口重复操作即可 1v1 实测
```

## 📡 API 参考

### REST 端点

Base URL：`/api/v1`。错误响应统一为信封格式：

```json
{ "type": "error", "payload": { "code": 2004, "message": "room_not_found" } }
```

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/rooms` | 创建房间 |
| GET | `/rooms/{id}` | 查询房间 |
| POST | `/rooms/{id}/join-token` | 签发 Join Token |
| GET | `/ice-servers` | ICE 服务器配置 |
| GET | `/healthz` | 存活探针（返回 `ok`） |
| GET | `/readyz` | 就绪探针（检测 Redis，不可用时返回 503） |
| GET | `/metrics` | Prometheus 指标（`signal_*` 命名空间） |

**POST `/rooms` — 创建房间**：字段均可选，不指定 `id` 时自动生成 UUID。

请求：

```json
{ "id": "my-room-001", "maxParticipants": 8 }
```

`201 Created`：

```json
{ "id": "my-room-001", "maxParticipants": 8 }
```

**GET `/rooms/{id}` — 查询房间**（房间不存在时返回 `2004`）：

```json
{ "id": "my-room-001", "participants": 3 }
```

**POST `/rooms/{id}/join-token` — 签发令牌**：可选请求头 `X-Admin-Key: <key>`（常量时间比较）。

请求：

```json
{ "userId": "user-alice", "displayName": "Alice", "role": "speaker", "ttlSeconds": 600 }
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `userId` | 是 | 业务用户标识 |
| `displayName` | 否 | 显示名 |
| `role` | 否 | `viewer` / `speaker` / `moderator`，默认 `speaker` |
| `ttlSeconds` | 否 | Token 有效期（秒），默认 900，最大 3600 |

`200 OK`：

```json
{ "token": "eyJhbGciOiJIUzI1NiIs...", "expiresIn": 600 }
```

**GET `/ice-servers` — ICE 配置**：

```json
[ { "urls": ["stun:stun.l.google.com:19302"] } ]
```

### WebSocket 信令

连接：`GET /ws/v1?token=<JWT>`，或使用 Header `Authorization: Bearer <JWT>`。连接建立后**必须**首先发送 `join` 消息，否则连接将被关闭。

**消息信封** — `id`、`ts`、`from` 由服务端自动填充：

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "version": "v1",
  "type": "offer",
  "to": "peer-b",
  "from": "peer-a",
  "ts": 1707800000000,
  "payload": { "sdp": "v=0..." }
}
```

| 字段 | 说明 |
|------|------|
| `version` | 协议版本，固定 `v1` |
| `type` | 消息类型 |
| `to` | 目标 peer ID，省略或 `*` 表示广播 |
| `payload` | 消息体，结构因 `type` 而异 |

**客户端 → 服务端**：

| 类型 | payload | 说明 |
|------|---------|------|
| `join` | `{ roomId, displayName?, role? }` | 加入房间 |
| `offer` | `{ to, sdp }` | SDP Offer |
| `answer` | `{ to, sdp }` | SDP Answer |
| `trickle` | `{ to, candidate }` | ICE 候选 |
| `chat` | `{ text }` | 文本消息；`to` 放在 envelope 顶层，省略则广播 |
| `mute` / `unmute` | `{}` | 静音控制；定向控制时目标 peer 放在 envelope 顶层 `to` |
| `leave` | — | 离开房间 |

**服务端 → 客户端**：

| 类型 | payload | 说明 |
|------|---------|------|
| `joined` | `{ self, peers[], iceServers[] }` | 加入成功，返回成员列表与 ICE 配置 |
| `participant-joined` | `{ id, displayName, role }` | 新成员通知 |
| `participant-left` | `{ id }` | 成员离开通知 |
| `offer` / `answer` / `trickle` / `chat` / `mute` / `unmute` | 同上 | 转发对端消息 |
| `error` | `{ code, message, details? }` | 错误 |

**错误码**：

| 码 | 含义 | 说明 |
|------|------|------|
| 2001 | `invalid_message` | 消息格式不合法 |
| 2002 | `unauthorized` | 未认证或 Token 过期 |
| 2003 | `forbidden` | 权限不足 |
| 2004 | `room_not_found` | 房间不存在 |
| 2006 | `unsupported_type` | 不支持的消息类型 |
| 2007 | `rate_limited` | 超出速率限制 |
| 2010 | `bad_state` / `room_full` | 状态异常或房间已满 |
| 3000 | `internal_error` | 服务端内部错误 |

## ⚙️ 配置

所有配置项均通过环境变量设置：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SIGNAL_JWT_SECRET` | — | JWT 签名密钥（**必填**，未设置时服务启动失败） |
| `SIGNAL_ADDR` | `:8080` | 监听地址 |
| `SIGNAL_LOG_LEVEL` | `info` | 日志级别（`debug` / `info` / `warn` / `error`） |
| `SIGNAL_ADMIN_KEY` | — | 管理 API 密钥（可选） |
| `SIGNAL_ALLOWED_ORIGINS` | — | Origin 白名单（逗号分隔；留空时接受任意浏览器来源） |
| `SIGNAL_REDIS_ADDR` | — | Redis 地址；**非空即启用** Redis Pub/Sub 多节点扩展 |
| `SIGNAL_STUN` | `stun:stun.l.google.com:19302` | STUN 服务器列表（逗号分隔） |

其余运行参数为内置常量：HTTP 读写超时 10s、WS 心跳间隔 10s / 超时 25s、单消息上限 64KB、每连接限速 20 RPS（突发 40）。

## 🐳 Docker

```bash
# 构建镜像（支持版本注入）
docker build --build-arg VERSION=v0.2.0 -t lessup/signaling:v0.2.0 .

# 本地编排（含 Redis）
export SIGNAL_JWT_SECRET="change-me-to-a-long-random-secret"
cd docker && docker compose up --build
```

## 🧪 开发与测试

```bash
make test          # 单元测试
make test-race     # 竞态检测
make test-cover    # 覆盖率报告
make vet           # go vet
make build         # 编译（含版本注入）
```

## 📄 许可证

[MIT](LICENSE) © LessUp
