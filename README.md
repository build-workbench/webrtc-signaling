# 实时音视频信令服务（Go）

一个基于 Go 的 WebRTC 信令服务，提供房间管理、会话协商（SDP/ICE）与基础控制（聊天/静音）。支持单实例与 Redis Pub/Sub 的多实例水平扩展，并集成 Prometheus 指标与健康检查。附带 Web Demo 便于本地验证。

## 快速开始

1) 准备环境变量（至少设置 JWT Secret）：
```bash
export SIGNAL_JWT_SECRET="dev-secret-change"
```

2) 运行服务：
```bash
# 下载依赖并编译（如遇网络问题，请使用国内代理）
# env GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.org go mod tidy

go run ./cmd/server
```

3) 打开 Demo：
- 浏览器访问 http://localhost:8080/demo
- 输入房间 ID（如 room-001）和显示名，点击加入
- 新开一个浏览器窗口重复操作即可进行 1v1 实测

## REST API 摘要
- POST `/api/v1/rooms` 创建房间（可选）
- GET `/api/v1/rooms/{id}` 查询房间
- POST `/api/v1/rooms/{id}/join-token` 签发 Join Token（JWT）
- GET `/api/v1/ice-servers` 下发 ICE 服务器配置
- GET `/healthz`、`/readyz`、`/metrics`

WebSocket：`GET /ws/v1?token=<JWT>`，首条消息为 `{"type":"join","payload":{"roomId":"..."}}`

## 配置（环境变量）
- `SIGNAL_ADDR` 监听地址，默认 `:8080`
- `SIGNAL_ALLOWED_ORIGINS` 逗号分隔的 Origin 白名单
- `SIGNAL_MAX_MSG_BYTES` WebSocket 单消息大小上限（默认 65536）
- `SIGNAL_WS_PING_INTERVAL` 心跳间隔秒（默认 10）
- `SIGNAL_WS_PONG_WAIT` 心跳超时秒（默认 25）
- `SIGNAL_JWT_SECRET` Join Token 签名密钥（必填）
- `SIGNAL_ADMIN_KEY` 管理接口 AdminKey（可选）
- `SIGNAL_WS_RPS`/`SIGNAL_WS_BURST` 每连接速率限制（默认 20/40）
- `SIGNAL_REDIS_ENABLED` 是否启用 Redis 扩展（默认 false）
- `SIGNAL_REDIS_ADDR` `host:port`
- `SIGNAL_STUN` 逗号分隔 STUN 列表
- `SIGNAL_TURN_URLS` 逗号分隔 TURN URLs，配合 `SIGNAL_TURN_USERNAME` 与 `SIGNAL_TURN_CREDENTIAL`

## 容器与编排

构建镜像：
```bash
docker build -t lessup/signaling:dev .
```

本地编排：
```bash
cd docker
docker compose up --build
```
- 服务: http://localhost:8080
- Redis: localhost:6379
- coturn: turn:coturn:3478（见 `docker/coturn/turnserver.conf`）

## 运行测试
```bash
# 如遇依赖下载问题，添加 GOPROXY
# env GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.org go test ./...

go test ./...
```

k6 压测示例：
```bash
k6 run k6/ws-smoke.js
```

## 设计文档
详见 `docs/design.md`。
