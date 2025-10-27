# API 参考（v1）

Base URL: `/api/v1`

## 房间

- POST `/rooms`
  - Body: `{ id?: string, maxParticipants?: number, metadata?: object }`
  - 201: `{ id: string }`

- GET `/rooms/{id}`
  - 200: `{ id: string, participants: number }`

## 加入令牌

- POST `/rooms/{id}/join-token`
  - Headers: 可选 `X-Admin-Key: <key>`
  - Body: `{ userId: string, displayName?: string, role?: string, ttlSeconds?: number }`
  - 200: `{ token: string, expiresIn: number }`

## ICE 服务器

- GET `/ice-servers`
  - 200: `[{ urls: string[], username?: string, credential?: string, ttl?: number }]`

## 健康与指标

- GET `/healthz` 200: ok
- GET `/readyz` 200: ok
- GET `/metrics` Prometheus 指标

## WebSocket 信令（`/ws/v1`）

连接：`GET /ws/v1?token=<JWT>` 或 Header `Authorization: Bearer <JWT>`

- 首条消息：
```json
{"version":"v1","type":"join","payload":{"roomId":"room-001","displayName":"Alice"}}
```

- 支持类型：
  - `join`, `joined`, `participant-joined`, `participant-left`
  - `offer`, `answer`, `trickle`
  - `chat`, `mute`, `unmute`, `leave`, `error`

通用信封：
```json
{"version":"v1","type":"offer","to":"peer-b","payload":{"sdp":"..."}}
```

错误：
```json
{"type":"error","payload":{"code":2003,"message":"forbidden"}}
```
