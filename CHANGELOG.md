# 更新日志

WebRTC Signaling —— 用 Go 实现的 WebRTC 信令后端,聚焦认证、水平扩展、可观测性与部署。
格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/),版本号遵循[语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增

- WebSocket 信令核心:房间管理(人数上限、空房自动清理)与 SDP / ICE(offer / answer / trickle)消息路由,统一消息信封(`id` / `version` / `roomId` / `from` / `to` / `ts`)。
- 三级角色权限(viewer / speaker / moderator):viewer 不能发送媒体信令,仅 moderator 可静音他人,join payload 无法提权。
- JWT 认证:HS256 Join Token 签发与验证、管理密钥常量时间比较、Token TTL 收敛。
- Redis Pub/Sub 水平扩展:按房间订阅 + 引用计数 + nodeID 过滤自消息,本地优先路由并自动跨节点 fallback。
- Prometheus 指标(`signal_*` 命名空间)、zap 结构化 JSON 日志与 Request-ID 链路追踪。
- 中间件链:panic recovery、每连接速率限制、CORS Origin 白名单、安全响应头与访问日志。
- 房间 REST API(房间 CRUD 与 Join Token 签发,管理接口使用 `X-Admin-Key`),以及健康探针与 ICE 服务器配置下发接口。
- 部署与运维能力:Docker 多阶段构建 + Distroless 非 root 镜像、docker-compose 编排、`/healthz` 与 `/readyz` 探针、优雅关闭与容器内 `healthcheck` 子命令。
- 单文件浅色主题 Demo 前端(`web/index.html`)、自定义字体与配套产品截图文档。
- 覆盖权限、并发、Origin 限制等核心链路的测试,以及 `go build` + `go vet` + `go test -race` 的 CI。

### 变更

- 项目定位由教学项目调整为个人练手项目,README 重写为学习导向,并与姊妹项目 webrtc-call 形成「前端客户端 vs 后端信令服务」的对照。
- 仓库与 Go module 路径多次迁移,最终定名为 `vibe-knight/webrtc-signaling`;文档中文化并统一仓库链接。
- 架构演进:拆分为 `auth` / `config` / `httpapi` / `observability` / `room` 等模块,随后将 package 合并至 5 个并加深模块边界,简化 origin 校验与直连路由。
- 多轮整体优化:相继完成项目全面优化、第二轮优化与第三轮 server / WebSocket / UI 改进,并发布 v0.2.0、v0.3.0 两个里程碑。
- 工程瘦身:移除文档站与 GitHub Pages,简化 CI 与 Makefile,web demo 由三文件合并为单 `index.html`,环境变量收敛至 7 个。
- 更新 Dockerfile、Makefile、docker-compose、web demo 与各处配置,补充项目文档与 AI 编码代理配置,并让文档与运行时行为保持一致。
- 文档站历经 Jekyll(just-the-docs)搭建、GitHub Pages 工作流优化与后续重写,最终随工程瘦身一并移除。

### 修复

- 修复 WebSocket 相关缺陷,并加固运行时校验与交付默认值。
- 修复并发竞态与 CI 工作流执行失败问题,优化 CI 触发路径。
- 修复版本注入问题与 GitHub Pages 文档错误。
- 修复前端 Demo 标题与旧项目命名残留,统一仓库命名与链接。
- 修复 golangci-lint 的 errcheck / bodyclose 告警,统一 gofmt 格式。

### 移除

- 移除未使用的功能、冗余文档与旧模块名(aurora-signal)残留。
- 移除 GitHub Pages 文档站、Jekyll 配置及其旧残留。
