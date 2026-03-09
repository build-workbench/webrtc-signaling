<div align="center">

# Aurora Signal

**轻量级 WebRTC 信令服务，基于 Go 构建**

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/LessUp/aurora-signal/actions/workflows/ci.yml/badge.svg)](https://github.com/LessUp/aurora-signal/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/LessUp/aurora-signal)](https://goreportcard.com/report/github.com/LessUp/aurora-signal)

简体中文 | [English](README.md)

</div>

---

一个生产就绪的 WebRTC 信令服务，提供房间管理、会话协商（SDP/ICE）与基础媒体控制（聊天/静音）。支持单实例部署与 Redis Pub/Sub 多实例水平扩展，内置 Prometheus 指标与健康检查，附带 Web Demo 便于本地验证。

## ✨ 特性

- **房间管理** — REST API 创建/查询房间，支持 `maxParticipants` 人数上限，空房间自动清理
- **WebSocket 信令** — offer / answer / trickle / chat / mute / leave，消息自动填充 `id` + `ts` + `version` 便于追踪
- **角色权限** — 三级角色：`viewer`（不可发起媒体协商）/ `speaker` / `moderator`（可远程静音他人）
- **安全** — JWT Token 认证、Admin Key 常量时间比较、每连接速率限制、安全响应头
- **可观测性** — Prometheus 指标（`signal_*` 命名空间，含 `participants` gauge 和 `message_latency` histogram）、结构化 JSON 日志、请求日志中间件、Request-ID 链路追踪
- **高可用** — Redis Pub/Sub 多节点扩展、优雅关闭（含 WebSocket 连接排空）、panic recovery 中间件
- **Web Demo** — 断线指数退避重连、连接状态颜色指示、Enter 发送聊天
- **构建** — `ldflags` 版本注入、OCI 标签、Distroless 运行时镜像

## 🚀 快速开始

```bash
# 1. 设置 JWT Secret
export SIGNAL_JWT_SECRET="dev-secret-change"

# 2. 运行服务
make run
# 或：go run ./cmd/server

# 3. 打开 Demo
# 浏览器访问 http://localhost:8080/demo
# 输入房间 ID 和显示名，点击加入
# 新开窗口重复操作即可 1v1 实测
```

## 📡 REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/rooms` | 创建房间 |
| GET | `/api/v1/rooms/{id}` | 查询房间 |
| POST | `/api/v1/rooms/{id}/join-token` | 签发 Join Token |
| GET | `/api/v1/ice-servers` | ICE 服务器配置 |
| GET | `/healthz` | 存活探针 |
| GET | `/readyz` | 就绪探针（检测 Redis） |
| GET | `/metrics` | Prometheus 指标 |

**WebSocket** — `GET /ws/v1?token=<JWT>`，首条消息：`{"type":"join","payload":{"roomId":"..."}}`

完整 API 文档 → [`docs/API.md`](docs/API.md)

## ⚙️ 配置

所有配置项均通过环境变量设置：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SIGNAL_LOG_LEVEL` | `info` | 日志级别（`debug` / `info` / `warn` / `error`） |
| `SIGNAL_ADDR` | `:8080` | 监听地址 |
| `SIGNAL_JWT_SECRET` | — | JWT 签名密钥（**必填**） |
| `SIGNAL_ADMIN_KEY` | — | 管理 API 密钥（可选） |
| `SIGNAL_ALLOWED_ORIGINS` | — | Origin 白名单（逗号分隔） |
| `SIGNAL_MAX_MSG_BYTES` | `65536` | WebSocket 单消息大小上限（字节） |
| `SIGNAL_WS_PING_INTERVAL` | `10` | 心跳间隔（秒） |
| `SIGNAL_WS_PONG_WAIT` | `25` | 心跳超时（秒） |
| `SIGNAL_WS_RPS` / `SIGNAL_WS_BURST` | `20` / `40` | 每连接速率限制 |
| `SIGNAL_REDIS_ENABLED` | `false` | 启用 Redis 扩展 |
| `SIGNAL_REDIS_ADDR` | `localhost:6379` | Redis 地址 |

完整列表 → [`env.example`](env.example)

## 🐳 Docker

```bash
# 构建镜像（支持版本注入）
docker build --build-arg VERSION=v0.2.0 -t lessup/signaling:v0.2.0 .

# 本地编排（含 Redis + coturn）
cd docker && docker compose up --build
```

## 🧪 开发与测试

```bash
make test          # 单元测试
make test-race     # 竞态检测
make test-cover    # 覆盖率报告
make vet           # go vet
make lint          # golangci-lint
make build         # 编译（含版本注入）
```

压力测试：`k6 run k6/ws-smoke.js`

## 📖 文档

- [API 参考](docs/API.md)
- [设计文档](docs/design.md)
- [更新日志](CHANGELOG.md)
- [贡献指南](CONTRIBUTING.md)

## 📄 许可证

[MIT](LICENSE) © LessUp
