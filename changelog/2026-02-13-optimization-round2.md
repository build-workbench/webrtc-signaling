# 项目优化第二轮 (2026-02-13)

## 性能优化

- **Broadcast/SendTo 锁优化**：先快照连接列表后释放 RLock 再执行网络 I/O，避免长时间持锁阻塞 Join/Leave
- **消息自动填充**：所有转发消息自动添加 `ts`（UnixMilli 时间戳）和 `id`（UUID），提升链路追踪能力

## 可靠性

- **`/readyz` 增强**：启用 Redis 时检测连接可用性，不可达返回 503，区分于 `/healthz` 存活探针
- **安全响应头中间件**：自动添加 `X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy`
- **EOF 检查修复**：`handleCreateRoom` 使用 `io.EOF` 替代字符串比较

## 工程化

- **构建版本注入**：新增 `internal/version` 包，通过 ldflags 注入 Version/Commit/BuildTime，启动时打印
- **Makefile 增强**：新增 `test-race`、`test-cover`、`vet` 目标；`build` 自动注入版本信息
- **CI 更新**：Go 版本升级到 1.23、添加 `go vet` 步骤、启用 `-race` 检测
- **Config 单元测试**：覆盖默认值加载、环境变量覆盖、校验规则、`getEnvBool` 边界
