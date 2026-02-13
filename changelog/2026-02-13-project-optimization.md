# 项目全面优化 (2026-02-13)

## 关键 Bug 修复

- **模块名拼写错误**：`go.mod` 中 `singal` → `signal`，同步更新所有 Go 文件的 import 路径
- **Manager.Join 竞态条件**：将 getOrCreate 和 join 合并到同一把锁内，避免房间在两次加锁间被删除
- **Graceful shutdown 崩溃**：`http.ErrServerClosed` 不再触发 `log.Fatal`，改为正常退出
- **Admin key 时序攻击**：使用 `crypto/subtle.ConstantTimeCompare` 替代 `!=` 比较
- **Dockerfile 缺少 go.sum**：`COPY go.mod go.sum ./` 确保依赖校验完整
- **handleCreateRoom 忽略解码错误**：现在正确返回 400 错误
- **Ping goroutine 泄漏**：所有提前 return 路径均 `close(done)` 关闭 ping goroutine

## 代码质量

- 移除冗余 `zapError` 辅助函数，直接使用 `zap.Error()`
- 提取 `buildICEServers()` 消除 `handleICEServers` 与 `handleWS` 的重复 ICE 构建逻辑
- 提取 `routeMessage()` 合并 chat/mute/unmute 的重复消息路由代码
- 添加 `Config.Validate()` 方法，校验 JWT secret 强度和心跳参数合理性
- 支持通过 `SIGNAL_LOG_LEVEL` 环境变量配置日志级别

## 性能与可靠性

- 添加 **Recovery 中间件**：捕获 panic 并记录堆栈，返回 500 而非进程崩溃
- 添加 **Request-ID 中间件**：自动生成或透传 `X-Request-ID`，便于链路追踪
- 添加 **CORS 中间件**：为 REST API 提供 CORS 支持（复用 WebSocket 的 origin 检查逻辑）
- 添加 join/leave 结构化日志，便于运维排查

## 基础设施

- **Go 版本升级**：1.21 → 1.23（go.mod + Dockerfile 同步）
- **docker-compose.yml**：移除已废弃的 `version: "3.9"` 字段
- **golangci-lint 配置**：移除无效的 `version` 字段，新增 errcheck/unused/misspell/bodyclose/nilerr/prealloc 等 linter
- **Prometheus 指标**：所有指标统一 `signal_` namespace 前缀，新增 `signal_message_latency_seconds` 直方图

## 功能增强

- **maxParticipants 房间人数上限**：Room 结构体新增 `MaxParticipants` 字段，`Join` 时自动检查，`CreateRoom` API 支持设置
- **Redis Bus context 可取消**：`Bus.Close()` 先 cancel context 再关闭订阅和连接，防止资源泄漏

## 配置变更

- 新增环境变量 `SIGNAL_LOG_LEVEL`（默认 `info`，支持 debug/warn/error 等）
