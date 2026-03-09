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

## v0.1.0 — 初始发布
{: .d-inline-block }

Latest
{: .label .label-green }

### 新增

- **REST API** — 房间创建/查询、Join Token 签发、ICE 配置下发、健康探针与 Prometheus 指标
- **WebSocket 信令** — `join` / `offer` / `answer` / `trickle` / `chat` / `leave` / `mute` / `unmute`
- **安全** — JWT 认证、Admin Key、速率限制、安全响应头
- **可观测性** — Prometheus 指标（`signal_` namespace）、结构化 JSON 日志、Request-ID 追踪
- **多实例扩展** — Redis Pub/Sub 消息转发
- **容器化** — Dockerfile（Distroless）与 docker-compose（Signal + Redis + coturn）
- **Web Demo** — 断线退避重连、连接状态指示、Enter 发送聊天
- **测试** — 单元测试与 k6 压测脚本
