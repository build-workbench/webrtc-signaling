---
title: API 参考
layout: default
nav_order: 2
description: "REST 端点与 WebSocket 信令协议完整说明"
---

# API 参考（v1）
{: .no_toc }

<details open markdown="block">
  <summary>目录</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## 通用约定

**Base URL**：`/api/v1`

所有 HTTP 响应均包含以下安全头：

| Header | 值 |
|:--|:--|
| `X-Request-ID` | 请求追踪 ID（自动生成或透传客户端值） |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |

错误响应统一格式：

```json
{
  "type": "error",
  "payload": { "code": 2004, "message": "room_not_found" }
}
```

---

## REST 端点

### POST `/rooms` — 创建房间

**请求体**（均可选）：

```json
{
  "id": "my-room-001",
  "maxParticipants": 8,
  "metadata": { "title": "Stand-up" }
}
```

**201 Created**：

```json
{
  "id": "my-room-001",
  "maxParticipants": 8
}
```

{: .note }
若不指定 `id`，服务端自动生成 UUID。

---

### GET `/rooms/{id}` — 查询房间

**200 OK**：

```json
{
  "id": "my-room-001",
  "participants": 3
}
```

**404 Not Found**：

```json
{
  "type": "error",
  "payload": { "code": 2004, "message": "room_not_found" }
}
```

---

### POST `/rooms/{id}/join-token` — 签发令牌

**请求头**（可选）：`X-Admin-Key: <key>`（常量时间比较）

**请求体**：

```json
{
  "userId": "user-alice",
  "displayName": "Alice",
  "role": "speaker",
  "ttlSeconds": 600
}
```

| 字段 | 必填 | 说明 |
|:--|:--|:--|
| `userId` | 是 | 业务用户标识 |
| `displayName` | 否 | 显示名 |
| `role` | 否 | `viewer` / `speaker` / `moderator` |
| `ttlSeconds` | 否 | Token 有效期（秒），默认 900 |

**200 OK**：

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expiresIn": 600
}
```

---

### GET `/ice-servers` — ICE 配置

**200 OK**：

```json
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

---

### GET `/healthz` — 存活探针

**200 OK**：`ok`

---

### GET `/readyz` — 就绪探针

检测 Redis 连接可用性（启用时）。

| 状态码 | 含义 |
|:--|:--|
| 200 | `ok` |
| 503 | `redis unreachable` |

---

### GET `/metrics` — Prometheus 指标

返回 Prometheus 文本格式指标，namespace 为 `signal_`。

---

## WebSocket 信令

### 连接

```
GET /ws/v1?token=<JWT>
```

或使用 Header：`Authorization: Bearer <JWT>`

{: .important }
连接建立后，客户端**必须**首先发送 `join` 消息，否则连接将被关闭。

---

### 消息信封

所有消息遵循统一信封格式。服务端自动填充 `id`、`ts`、`from` 字段。

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
|:--|:--|
| `id` | UUID，服务端自动生成 |
| `version` | 协议版本，当前固定 `v1` |
| `type` | 消息类型 |
| `to` | 目标 peer ID，省略或 `*` 表示广播 |
| `from` | 发送方 peer ID（服务端填充） |
| `ts` | Unix 毫秒时间戳（服务端填充） |
| `payload` | 消息体，结构因 `type` 而异 |

---

### 客户端 → 服务端

| 类型 | payload | 说明 |
|:--|:--|:--|
| `join` | `{ roomId, displayName?, role? }` | 加入房间 |
| `offer` | `{ to, sdp }` | SDP Offer |
| `answer` | `{ to, sdp }` | SDP Answer |
| `trickle` | `{ to, candidate }` | ICE 候选 |
| `chat` | `{ to?, text }` | 文本消息（`to` 省略则广播） |
| `mute` / `unmute` | `{ target? }` | 静音控制（需权限） |
| `leave` | — | 离开房间 |

**示例 — join**：

```json
{
  "version": "v1",
  "type": "join",
  "payload": { "roomId": "room-001", "displayName": "Alice" }
}
```

**示例 — offer**：

```json
{
  "version": "v1",
  "type": "offer",
  "to": "peer-b",
  "payload": { "sdp": "v=0\r\no=- 46117317..." }
}
```

---

### 服务端 → 客户端

| 类型 | payload | 说明 |
|:--|:--|:--|
| `joined` | `{ self, peers[], iceServers[] }` | 加入成功，返回成员列表与 ICE 配置 |
| `participant-joined` | `{ id, displayName, role }` | 新成员通知 |
| `participant-left` | `{ id }` | 成员离开通知 |
| `offer` / `answer` / `trickle` | 同上 | 转发对端消息 |
| `chat` | `{ text }` | 转发文本消息 |
| `mute-request` | `{ target? }` | 对端或系统请求静音 |
| `error` | `{ code, message, details? }` | 错误 |

**示例 — joined**：

```json
{
  "id": "...",
  "version": "v1",
  "type": "joined",
  "ts": 1707800000000,
  "payload": {
    "self": { "id": "peer-a", "role": "speaker" },
    "peers": [
      { "id": "peer-b", "role": "speaker", "displayName": "Bob" }
    ],
    "iceServers": [
      { "urls": ["stun:stun.l.google.com:19302"] }
    ]
  }
}
```

**示例 — error**：

```json
{
  "version": "v1",
  "type": "error",
  "payload": { "code": 2003, "message": "forbidden" }
}
```

---

### 错误码

| 码 | 含义 | 说明 |
|:--|:--|:--|
| 2001 | `invalid_message` | 消息格式不合法 |
| 2002 | `unauthorized` | 未认证或 Token 过期 |
| 2003 | `forbidden` | 权限不足 |
| 2004 | `room_not_found` | 房间不存在 |
| 2006 | `unsupported_type` | 不支持的消息类型 |
| 2007 | `rate_limited` | 超出速率限制 |
| 2010 | `bad_state` / `room_full` | 状态异常或房间已满 |
| 3000 | `internal_error` | 服务端内部错误 |

---

## 配置环境变量

| 变量 | 默认值 | 说明 |
|:--|:--|:--|
| `SIGNAL_LOG_LEVEL` | `info` | 日志级别（debug / info / warn / error） |
| `SIGNAL_ADDR` | `:8080` | 监听地址 |
| `SIGNAL_JWT_SECRET` | — | **必填**，JWT 签名密钥 |
| `SIGNAL_ADMIN_KEY` | — | 管理 API 密钥（可选） |
| `SIGNAL_ALLOWED_ORIGINS` | — | CORS Origin 白名单（逗号分隔） |
| `SIGNAL_MAX_MSG_BYTES` | `65536` | WebSocket 单消息上限（字节） |
| `SIGNAL_WS_PING_INTERVAL` | `10` | 心跳间隔（秒） |
| `SIGNAL_WS_PONG_WAIT` | `25` | 心跳超时（秒） |
| `SIGNAL_WS_RPS` / `SIGNAL_WS_BURST` | `20` / `40` | 每连接速率限制 |
| `SIGNAL_REDIS_ENABLED` | `false` | 启用 Redis 多节点扩展 |
| `SIGNAL_REDIS_ADDR` | `localhost:6379` | Redis 地址 |

完整列表见 [`env.example`](https://github.com/LessUp/aurora-signal/blob/main/env.example)
