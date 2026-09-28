# AGENTS.md — WebRTC Signaling

生产风格（production-style）的 WebRTC 信令服务，用 Go 从零实现：聚焦 JWT 认证、角色权限、Redis 水平扩展、可观测性与部署。只做信令，媒体流浏览器间 P2P 直连不经过服务端。

## 常用命令

- `make run` — 本地启动（需先 `export SIGNAL_JWT_SECRET=...`），Demo 前端在 `http://localhost:8080/demo`
- `make build` — 构建二进制到 `bin/signal-server`（`go build -trimpath`）
- `make test` — 运行全部测试，带 `-race -count=1`（与 CI 一致）
- `make vet` / `make fmt` — `go vet ./...` 静态检查 / `go fmt ./...` 格式化（项目未配置 golangci-lint）
- `cd docker && docker compose up --build` — 多实例编排（server + redis + 健康检查，须先设置 `SIGNAL_JWT_SECRET`）

## 代码结构

- `cmd/server/main.go` — 应用入口：日志与配置初始化、优雅关闭、容器内 `healthcheck` 子命令
- `internal/httpapi/` — 最核心包：服务器组装（server.go）、消息路由本地优先 + Redis fallback（router.go）、WebSocket 会话与速率限制（ws.go）、REST 处理器（handler_room.go / handler_health.go）、中间件链（middleware.go）、角色权限策略（permission.go）、Redis Pub/Sub（redisbus.go）
- `internal/auth/` — JWT（HS256）Join Token 签发与验证、管理密钥常量时间比较
- `internal/config/` — 环境变量配置（`SIGNAL_*` 前缀，共 7 个变量）
- `internal/room/` — 房间与 Peer 生命周期管理（人数上限、空房自动清理）
- `internal/observability/` — Prometheus 指标（`signal_*` 命名空间）；日志为 zap JSON 结构化输出并带 Request-ID 链路追踪
- 健康探针：`/healthz`（存活）与 `/readyz`（就绪，检测 Redis）；`/metrics` 暴露 Prometheus 指标
- `web/index.html` — 单文件演示前端（浅色主题，非生产 UI）
- `Dockerfile` + `docker/docker-compose.yml` — 多阶段构建 + Distroless 非 root 镜像与一键编排
- 测试与源码同目录：`ws_test.go`（最大）、`router_test.go`、`jwt_test.go`、`config_test.go`、`manager_test.go`，覆盖权限、并发、Origin 限制等核心链路

## 关键约束

- Go 1.22（go.mod 与 CI 的 setup-go 一致；Dockerfile 构建镜像用 1.23）；模块路径 `github.com/build-workbench/webrtc-signaling`
- CI（`.github/workflows/ci.yml`）执行 `go build ./...` + `go test -race -count=1 ./...` + `go vet ./...`，提交前本地至少跑 `make test` 与 `make vet`
- 配置只走环境变量：`SIGNAL_JWT_SECRET` 必填；`SIGNAL_REDIS_ADDR` 非空即启用 Redis Pub/Sub 多节点扩展，`SIGNAL_ALLOWED_ORIGINS` 控制 Origin 白名单
- WebSocket 消息统一信封（`id` / `version` / `roomId` / `from` / `to` / `ts`），错误码 2001~3000 与消息类型定义见 README「API 参考」，改协议须同步更新
- 角色权限 viewer / speaker / moderator 以 JWT 中的角色为准，join payload 不可提权；REST Base URL 为 `/api/v1`，WS 入口 `/ws/v1?token=<JWT>`
- 边界（README 诚实声明）：仅信令、内置 STUN 无 TURN、无用户体系（Join Token 由 `X-Admin-Key` 管理 API 手动签发）、`web/` 仅演示
- 运维行为：优雅关闭时会主动关闭存量 WebSocket 连接；Docker 镜像以 nonroot 用户运行并监听 `:8080`
- 姊妹项目 webrtc-call 是浏览器端通话客户端，与本仓库构成「前端客户端 vs 后端信令服务」的对照学习路径；信令消息格式改动需两侧保持兼容

## 文档约定

- CHANGELOG.md：面向用户的变更在合入时写入 [Unreleased]（Keep a Changelog zh-CN 格式）
- 文档全中文
