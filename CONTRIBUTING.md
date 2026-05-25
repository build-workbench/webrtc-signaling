# 贡献指南

欢迎贡献！请遵循以下流程：

- 先在 Issue 讨论需求或 Bug，再提交 PR。
- Fork 仓库并创建特性分支。
- 本地最小检查：`go mod tidy && go test ./...`。
- 如修改 Go 代码，建议额外运行 `make vet && make lint`。
- PR 请附带描述、测试结果与影响范围。

## 分支命名

- `fix/*` — 缺陷修复
- `refactor/*` — 重构清理
- `docs/*` — 文档更新
- `test/*` — 测试补充

## 提交建议

- 推荐使用 Conventional Commits，例如：`refactor: remove unused config surface`
- 如修改 API 或行为，请同步更新 `README.md`、`docs/API.md` 与 `docs/design.md`
- 不保留未启用、未落地的功能分支或文档承诺

## 开发脚本
- `make build|run|test|docker-build|compose-up|compose-down`

## 行为准则
请遵守 `CODE_OF_CONDUCT.md`。
