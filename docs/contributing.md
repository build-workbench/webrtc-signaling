---
title: 贡献指南
layout: default
nav_order: 4
description: "最小化的本地开发、测试与提交流程"
---

# 贡献指南
{: .no_toc }

Aurora Signal 当前只保留最小、直接的开发流程。先保证代码和文档与真实运行时一致，再考虑扩展。

<details open markdown="block">
  <summary>目录</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

## 1. 环境要求

| 工具 | 版本 |
|:--|:--|
| Go | `>= 1.22` |
| Docker（可选） | `>= 24` |
| golangci-lint（可选） | 当前仓库 CI 所用版本兼容即可 |

---

## 2. 常用命令

```bash
make build
make run
make test
make test-race
make test-cover
make vet
make lint
make fmt
make docker-build
make compose-up
make compose-down
```

本地最小检查：

```bash
go mod tidy
go test ./...
```

---

## 3. 分支与提交

推荐分支前缀：

| 前缀 | 用途 | 示例 |
|:--|:--|:--|
| `fix/` | 缺陷修复 | `fix/ws-ping-timeout` |
| `refactor/` | 重构清理 | `refactor/httpapi-simplify` |
| `docs/` | 文档更新 | `docs/api-reference` |
| `test/` | 测试补充 | `test/ws-edge-cases` |

推荐使用 Conventional Commits：

```text
fix: correct websocket timeout handling
refactor: remove unused config surface
docs: align API reference with runtime
test: add ws moderation coverage
```

---

## 4. 提交前检查

- 运行 `make test`
- 如修改了并发或连接逻辑，额外运行 `make test-race`
- 如修改了 Go 代码，运行 `make vet`
- 如修改了 API 或行为，更新 `README` / `docs/API.md` / `docs/design.md`

---

## 5. 提交原则

- 不保留未启用、未落地的功能分支
- 不让文档领先于代码实现
- 不引入新的 AI 控制框架、计划系统或冗余规范层

如无必要，不新增抽象层。
