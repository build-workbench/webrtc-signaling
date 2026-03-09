---
title: 变更日志
layout: default
nav_order: 4
description: "Aurora Signal 版本发布历史"
---

# 变更日志
{: .no_toc }

本页记录每个版本的主要变更。格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

---

## v0.2.0 — Refactor
{: .d-inline-block }

Latest
{: .label .label-green }

### Breaking Changes

- Go module 重命名为 `github.com/LessUp/aurora-signal`
- `config.Validate()` 签名变更为 `([]string, error)`，返回警告列表
- `room.Manager.CreateRoom()` 现接受可选 `maxParticipants` 参数，在锁内原子设置

### Architecture

- `internal/httpapi/server.go` 拆分为：
  - `server.go` — Server 结构体、构造函数、生命周期
  - `middleware.go` — recovery / requestID / CORS / securityHeaders / accessLog
  - `handler_health.go` — healthz / readyz / ice-servers
  - `handler_room.go` — rooms CRUD / join-token
- `ws.go` 重构：提取 `wsSession` 结构体，消除 `goto`，统一 `routeMessage` 路由逻辑，拆分 Redis 订阅/退订为独立方法

### Features

- **角色权限** — viewer 不可发起 offer/answer/trickle；仅 moderator 可远程 mute/unmute 他人
- **Envelope version** — 所有路由消息自动填充 `version: "v1"`
- **MessageLatency** — `routeMessage` 记录处理延迟到 Prometheus histogram
- **ParticipantsGauge** — 新增 `signal_participants` Prometheus gauge，Join/Leave 时自动增减
- **优雅关闭 WS** — Server.Shutdown 时向所有活跃 WebSocket 连接发送 CloseGoingAway 帧

### Code Quality

- 全局 `interface{}` → `any`
- `statusRecorder` 实现 `http.Hijacker`，修复 WebSocket 升级通过 accessLog 中间件时的 bad handshake bug
- 删除未使用类型 `SDP` / `Trickle` / `Chat`

### Tests

- 新增 6 个集成测试：AdminKeyAuth / RoomFull / ViewerCannotSignal / ModeratorCanMuteOthers / ConcurrentJoinLeave / EnvelopeVersionPopulated

---

## v0.1.0 — 初始发布

### 新增

- **REST API** — 房间创建/查询、Join Token 签发、ICE 配置下发、健康探针与 Prometheus 指标
- **WebSocket 信令** — `join` / `offer` / `answer` / `trickle` / `chat` / `leave` / `mute` / `unmute`
- **安全** — JWT 认证、Admin Key、速率限制、安全响应头
- **可观测性** — Prometheus 指标（`signal_` namespace）、结构化 JSON 日志、Request-ID 追踪
- **多实例扩展** — Redis Pub/Sub 消息转发
- **容器化** — Dockerfile（Distroless）与 docker-compose（Signal + Redis + coturn）
- **Web Demo** — 断线退避重连、连接状态指示、Enter 发送聊天
- **测试** — 单元测试与 k6 压测脚本
