# API 参考（v1）

Base URL: `/api/v1`

## 通用

所有响应包含以下 Header：
- `X-Request-ID` — 请求追踪 ID（自动生成或透传客户端 `X-Request-ID`）
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`

## 房间

- POST `/rooms`
  - Body: `{ id?: string, maxParticipants?: number, metadata?: object }`
  - 201: `{ id: string, maxParticipants: number }`

- GET `/rooms/{id}`
  - 200: `{ id: string, participants: number }`
  - 404: `{ type: "error", payload: { code: 2004, message: "room_not_found" } }`

## 加入令牌

- POST `/rooms/{id}/join-token`
  - Headers: 可选 `X-Admin-Key: <key>`（使用常量时间比较）
  - Body: `{ userId: string, displayName?: string, role?: string, ttlSeconds?: number }`
  - 200: `{ token: string, expiresIn: number }`

## ICE 服务器

- GET `/ice-servers`
  - 200: `[{ urls: string[], username?: string, credential?: string, ttl?: number }]`

## 健康与指标

- GET `/healthz` — 存活探针，200: ok
- GET `/readyz` — 就绪探针，检测 Redis 连接可用性（启用时），200: ok / 503: redis unreachable
- GET `/metrics` — Prometheus 指标（namespace: `signal_`）

## WebSocket 信令（`/ws/v1`）

连接：`GET /ws/v1?token=<JWT>` 或 Header `Authorization: Bearer <JWT>`

- 首条消息必须为 `join`：
```json
{"version":"v1","type":"join","payload":{"roomId":"room-001","displayName":"Alice"}}
```

- 支持类型：
  - `join`, `joined`, `participant-joined`, `participant-left`
  - `offer`, `answer`, `trickle`
  - `chat`, `mute`, `unmute`, `leave`, `error`

通用信封（服务端自动填充 `id`、`ts`、`from`）：
```json
{"id":"uuid","version":"v1","type":"offer","to":"peer-b","from":"peer-a","ts":1707800000000,"payload":{"sdp":"..."}}
```

错误码：
| 码 | 含义 |
|---|---|
| 2001 | invalid_message |
| 2002 | unauthorized |
| 2003 | forbidden |
| 2004 | room_not_found |
| 2006 | unsupported_type |
| 2007 | rate_limited |
| 2010 | bad_state / room_full |
| 3000 | internal_error |

## 配置环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `SIGNAL_LOG_LEVEL` | `info` | 日志级别 (debug/info/warn/error) |
| `SIGNAL_ADDR` | `:8080` | 监听地址 |
| `SIGNAL_JWT_SECRET` | — | JWT 签名密钥（必填） |
| `SIGNAL_ADMIN_KEY` | — | 管理 API 密钥（可选） |
| `SIGNAL_WS_RPS` / `SIGNAL_WS_BURST` | 20 / 40 | 每连接速率限制 |
| `SIGNAL_REDIS_ENABLED` | `false` | 启用 Redis 多节点扩展 |
| `SIGNAL_REDIS_ADDR` | `redis:6379` | Redis 地址 |
