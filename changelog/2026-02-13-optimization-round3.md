# 项目深度优化第三轮 (2026-02-13)

## 测试

- **WebSocket 集成测试** (`ws_test.go`)：覆盖完整信令流程
  - `TestWSJoinAndLeave` — 单用户加入/离开
  - `TestWSTwoPeersSignaling` — 双用户 join → offer → answer → trickle → leave，验证 ts/id 自动填充
  - `TestWSChat` — 广播聊天消息
  - `TestWSErrorCases` — missing_token / first_msg_not_join / offer_missing_to / unsupported_type
  - `TestRESTEndpoints` — healthz / readyz / rooms CRUD / ice-servers / 安全头 / request-id

## 可观测性

- **请求日志中间件** (`accessLogMiddleware`)：记录 method/path/status/duration/reqID，自动跳过 healthz/readyz/metrics 噪音端点
- **statusRecorder**：包装 ResponseWriter 捕获状态码

## 可靠性

- **Room TTL 自动清理**：`Manager.StartCleanup(interval, emptyTTL)` 后台定时清除空房间（默认每 30s 检查，空超 5min 清除）
- **优雅 WebSocket 关闭**：离开时发送 `CloseMessage(1000, "bye")` 通知客户端

## 基础设施

- **Dockerfile 优化**：ARG 版本注入（VERSION/COMMIT/BUILD_TIME）、OCI 标签、安装 git 用于版本信息
- **Room.CreatedAt 字段**：支持基于创建时间的 TTL 清理

## 前端 Web Demo

- **断线指数退避重连**：最多 5 次，延迟 1s → 2s → 4s → 8s → 16s
- **连接状态颜色指示**：info(蓝) / ok(绿) / warn(黄) / error(红)，CSS border-left 视觉反馈
- **安全发送**：`send()` 检查 WebSocket readyState，防止断连后发送异常
- **离开时完整清理**：停止媒体轨道、清空 ICE 缓存、重置 UI
- **Enter 键发送聊天**
- **中文化日志**：加入/离开/错误消息使用中文

## 文档

- **README.md 全面重写**：特性列表、表格化 API 摘要和配置变量、构建/测试命令
- **docs/API.md 更新**：通用响应头、错误码表、readyz 增强说明、消息 id/ts 说明、配置变量表
