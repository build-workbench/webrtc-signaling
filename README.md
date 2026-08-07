# WebRTC Signaling

**Go 语言 WebRTC 信令服务教学项目** -- 涵盖 JWT 认证、Redis Pub/Sub 水平扩展、Prometheus 指标、结构化日志、速率限制等后端工程模式。

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/vibe-knight/webrtc-signaling/actions/workflows/ci.yml/badge.svg)](https://github.com/vibe-knight/webrtc-signaling/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## 这个项目教你什么

- **WebSocket 信令** -- 房间管理、SDP/ICE 消息路由、消息信封设计
- **JWT 认证** -- Token 签发与验证、Join Token 流程、常量时间比较
- **角色权限** -- viewer / speaker / moderator 三级角色与权限控制
- **Redis Pub/Sub** -- 多实例水平扩展，跨节点消息转发
- **Prometheus 指标** -- `signal_*` 命名空间，gauge + histogram
- **结构化日志** -- zap JSON 日志、Request-ID 链路追踪
- **中间件** -- panic recovery、速率限制、CORS、安全响应头、请求日志
- **Docker 部署** -- Distroless 镜像、docker-compose 编排

## 架构

```
cmd/server/main.go              应用入口
internal/
├── auth/                       JWT 签发与验证
├── config/                     环境变量配置
├── httpapi/
│   ├── server.go               HTTP/WS 服务器组装
│   ├── router.go               消息路由（本地 + Redis Bus）
│   ├── ws.go                   WebSocket 会话管理
│   ├── handler_room.go         REST API：房间 CRUD
│   ├── handler_health.go       健康探针
│   ├── middleware.go           中间件链
│   ├── permission.go           角色权限策略
│   ├── redisbus.go             Redis Pub/Sub 实现
│   └── utils.go                工具函数
├── observability/metrics.go    Prometheus 指标
└── room/manager.go             房间与 Peer 生命周期管理
web/index.html                  单文件 Demo 前端
```

## 快速开始

### 单实例

```bash
export SIGNAL_JWT_SECRET="change-me-to-a-long-random-secret"
make run
```

打开 `http://localhost:8080/demo`，输入房间和昵称即可体验。

### 多实例（Redis 扩展）

```bash
export SIGNAL_JWT_SECRET="change-me-to-a-long-random-secret"
cd docker && docker compose up --build
```

## 配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SIGNAL_JWT_SECRET` | - | JWT 签名密钥（**必填**） |
| `SIGNAL_ADDR` | `:8080` | 监听地址 |
| `SIGNAL_LOG_LEVEL` | `info` | 日志级别 |
| `SIGNAL_ADMIN_KEY` | - | 管理 API 密钥（可选） |
| `SIGNAL_ALLOWED_ORIGINS` | - | Origin 白名单（逗号分隔） |
| `SIGNAL_REDIS_ADDR` | - | Redis 地址；非空即启用多节点扩展 |
| `SIGNAL_STUN` | `stun:stun.l.google.com:19302` | STUN 服务器列表 |

## API 参考

### REST 端点

Base URL：`/api/v1`

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/rooms` | 创建房间 |
| GET | `/rooms/{id}` | 查询房间 |
| POST | `/rooms/{id}/join-token` | 签发 Join Token |
| GET | `/ice-servers` | ICE 服务器配置 |
| GET | `/healthz` | 存活探针 |
| GET | `/readyz` | 就绪探针（检测 Redis） |
| GET | `/metrics` | Prometheus 指标 |

### WebSocket 信令

连接：`GET /ws/v1?token=<JWT>`，连接后必须先发 `join` 消息。

消息信封（`id`/`ts`/`from` 由服务端填充）：

```json
{ "id": "uuid", "version": "v1", "type": "offer", "to": "peer-b", "from": "peer-a", "ts": 1707800000000, "payload": {} }
```

| 客户端 -> 服务端 | 说明 |
|---|---|
| `join` | 加入房间 |
| `offer` / `answer` | SDP 协商 |
| `trickle` | ICE 候选 |
| `chat` | 文本消息 |
| `mute` / `unmute` | 静音控制 |
| `leave` | 离开房间 |

| 错误码 | 含义 |
|---|---|
| 2001 | 消息格式不合法 |
| 2002 | 未认证或 Token 过期 |
| 2003 | 权限不足 |
| 2004 | 房间不存在 |
| 2006 | 不支持的消息类型 |
| 2007 | 超出速率限制 |
| 2010 | 状态异常或房间已满 |
| 3000 | 服务端内部错误 |

## 许可证

[MIT](LICENSE)
