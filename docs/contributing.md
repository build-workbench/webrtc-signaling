---
title: 贡献指南
layout: default
nav_order: 5
description: "开发环境、分支规范与提交流程"
---

# 贡献指南
{: .no_toc }

<details open markdown="block">
  <summary>目录</summary>
  {: .text-delta }
- TOC
{:toc}
</details>

---

欢迎贡献！请遵循以下流程。

## 开始之前

1. 先在 [Issue](https://github.com/LessUp/aurora-signal/issues) 讨论需求或 Bug
2. Fork 仓库并创建特性分支
3. 确保本地通过所有检查后再提交 PR

## 开发环境

**前置要求**：

| 工具 | 版本 |
|:--|:--|
| Go | ≥ 1.23 |
| Docker（可选） | ≥ 24 |
| golangci-lint（可选） | ≥ 1.55 |

**快速启动**：

```bash
git clone https://github.com/<your-fork>/aurora-signal.git
cd aurora-signal
cp env.example .env
export SIGNAL_JWT_SECRET="dev-secret"
make run
```

## Make 命令

```bash
make build          # 编译（含版本注入）
make run            # 运行服务
make test           # 单元测试
make test-race      # 竞态检测
make test-cover     # 覆盖率报告
make vet            # go vet
make lint           # golangci-lint
make docker-build   # 构建 Docker 镜像
make compose-up     # 启动本地编排（Signal + Redis + coturn）
make compose-down   # 停止本地编排
```

## 分支命名

| 前缀 | 用途 | 示例 |
|:--|:--|:--|
| `feat/` | 新功能 | `feat/room-metadata` |
| `fix/` | Bug 修复 | `fix/ws-ping-timeout` |
| `docs/` | 文档 | `docs/api-examples` |
| `refactor/` | 重构 | `refactor/room-store` |
| `test/` | 测试 | `test/k6-concurrent` |

## 提交规范

推荐使用 [Conventional Commits](https://www.conventionalcommits.org/)：

```
feat: add room metadata support
fix: correct WebSocket ping interval
docs: update API reference examples
test: add k6 concurrent room test
```

## PR 检查清单

- [ ] 代码通过 `make vet` 与 `make lint`
- [ ] 所有测试通过 `make test-race`
- [ ] 新功能附带单元测试
- [ ] PR 描述包含变更说明与影响范围
- [ ] 如涉及 API 变更，同步更新 `docs/API.md`

## 行为准则

请遵守 [CODE_OF_CONDUCT.md](https://github.com/LessUp/aurora-signal/blob/main/CODE_OF_CONDUCT.md)。
