# WebRTC Signaling

**生产风格（production-style）的 WebRTC 信令服务** —— 用 Go 从零实现，聚焦认证、水平扩展、可观测性与部署等后端工程实践。个人练手作品，与姊妹项目 [webrtc-demo](https://github.com/build-workbench/webrtc-demo) 形成「最小实现 vs 工程化实现」的对照学习路径。

[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/build-workbench/webrtc-signaling/actions/workflows/ci.yml/badge.svg)](https://github.com/build-workbench/webrtc-signaling/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## 项目定位与场景目标

WebRTC 的媒体流在浏览器之间 P2P 直连，但"谁在哪个房间、如何互相发现、如何交换 SDP / ICE 候选"需要一条信令通道。**本仓库就是这个信令后端**，并刻意按生产工程的规格来写：

- **典型场景**：在线课堂 / 直播互动（speaker 发言、viewer 只观看、moderator 管控）、小规模会议与协作工具，以及任何需要"房间 + 角色权限 + 消息路由"的 RTC 应用后端
- **学习 / 作品定位**：每个特性均为手写实现并配套测试，作为 WebRTC 信令与 Go 后端工程实践的参考起点，而非开箱即用的云产品
- **边界（诚实声明）**：
  - 只做信令，媒体流不经过服务端（P2P）；内置 STUN，**不含 TURN**，严格 NAT 环境下可能无法打通
  - 无用户体系：Join Token 由管理 API（`X-Admin-Key`）手动签发，未接入第三方 IdP
  - `web/` 为单文件演示前端，非生产 UI
  - 未做大规模压测与集群真实验证，练手性质

### 姊妹项目

| 项目 | 定位 |
|------|------|
| [webrtc-demo](https://github.com/build-workbench/webrtc-demo) | **最小实现**：Go 信令 + 浏览器原生 WebRTC 音视频通话，适合先理解协议全貌 |
| **webrtc-signaling（本仓库）** | **工程化实现**：在信令之上补齐认证、水平扩展、可观测性、部署与测试 |

## 功能特性

### 信令核心
- **WebSocket 信令** -- 房间管理（人数上限、空房自动清理）、SDP/ICE 消息路由（offer / answer / trickle）、统一消息信封（`id` / `version` / `roomId` / `from` / `to` / `ts`）
- **角色权限** -- viewer / speaker / moderator 三级角色：viewer 不能发送媒体信令，仅 moderator 可静音他人，join payload 无法提权（以 JWT 中的角色为准）

### 工程实践
- **JWT 认证** -- Join Token 签发与验证（HS256）、管理密钥常量时间比较、Token TTL 收敛
- **Redis Pub/Sub 水平扩展** -- 本地优先路由，跨节点自动 fallback；按房间订阅 + 引用计数 + nodeID 过滤自消息
- **Prometheus 指标** -- `signal_*` 命名空间：连接 / 房间 / 参与者 / 消息收发 / 错误 / 处理延迟
- **结构化日志** -- zap JSON 日志、Request-ID 链路追踪
- **中间件** -- panic recovery、每连接速率限制、CORS Origin 白名单、安全响应头、访问日志
- **测试与 CI** -- 覆盖权限、并发、Origin 限制等核心链路；CI 执行 `go build` + `go vet` + `go test -race`

### 部署与运维
- **Docker 部署** -- 多阶段构建 + Distroless 非 root 镜像，docker-compose 一键编排（server + redis + 健康检查）
- **运维能力** -- 优雅关闭（主动关闭存量 WebSocket）、`/healthz` 存活探针、`/readyz` 就绪探针（检测 Redis）、容器内 `healthcheck` 子命令

## 架构

```
cmd/server/main.go              应用入口
internal/
├── auth/                       JWT 签发与验证
├── config/                     环境变量配置
├── httpapi/
│   ├── server.go               HTTP/WS 服务器组装
│   ├── router.go               消息路由（本地优先 + Redis Bus 跨节点 fallback）
│   ├── ws.go                   WebSocket 会话管理
│   ├── handler_room.go         REST API：房间 CRUD / Join Token 签发
│   ├── handler_health.go       健康探针与 ICE 服务器配置
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
| `SIGNAL_ADMIN_KEY` | - | 管理 API 密钥（签发 Join Token 用，可选） |
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
| POST | `/rooms/{id}/join-token` | 签发 Join Token（需 `X-Admin-Key`） |
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
| `mute` / `unmute` | 静音控制（moderator 可管控他人） |
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
