# 变更日志

## v0.3.0 — Final Optimization

### Fixes
- **Dockerfile** — 切换运行时镜像为 `gcr.io/distroless/static-debian12:nonroot`，修复 `ARG VERSION` 在第二阶段丢失导致 OCI LABEL 未插值的 Bug，添加 `org.opencontainers.image.licenses` 标签
- **config.go** — Redis 默认地址从 `redis:6379` 修正为 `localhost:6379`，与 `env.example` 保持一致
- **ws.go** — 移除 `joinRoom` 方法中未使用的 `claims` 参数
- **CODEOWNERS** — 替换占位符 `@your-github-username` 为 `@LessUp/maintainers`
- **API.md** — 修正 `ttlSeconds` 默认值文档（300 → 900），与代码一致
- **design.md** — 修正 Distroless 镜像引用为 `gcr.io/distroless/static-debian12`

### Improvements
- **Makefile** — 构建添加 `-trimpath` 标志，新增 `fmt` / `clean` 目标
- **.golangci.yml** — 新增 `gosec` / `gofmt` linter
- **.editorconfig** — 新增 `Makefile`（tab 缩进）、YAML、Markdown 规则
- **.gitignore** — 新增 `coverage.html`
- **CI** — 新增独立 `lint` job（golangci-lint-action）、Go module 缓存、覆盖率产物上传、Docker 构建依赖 lint 通过
- **OpenAPI** — 标题更名为 Aurora Signal API，版本升至 0.2.0，添加 license/description/operationId/tags，修正 healthz/readyz/metrics 的 server 路径，新增 Error schema 与 components/responses

### Documentation
- **docs/changelog.md** — 补充 v0.2.0 完整变更内容
- **docs/security.md** — 更新支持版本表（v0.2.x 当前支持，v0.1.x 仅安全补丁）
- **docs/contributing.md** — 补充 `make fmt` / `make clean` 命令说明
- **README** — 新增 Go Report Card 徽章（中英文双 README）

---

## v0.2.0 — Refactor

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

## v0.1.0
- 初始发布：
  - REST：房间与令牌 API、ICE 下发、健康与指标
  - WebSocket：join/offer/answer/trickle/chat/leave/mute/unmute
  - 可观测性：Prometheus 指标
  - 多实例：Redis Pub/Sub 消息转发
  - 容器化：Dockerfile 与 docker-compose
  - Demo：Web 前端示例
  - 测试：单元测试与 k6 压测脚本
