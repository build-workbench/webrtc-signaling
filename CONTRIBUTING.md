# 贡献指南

欢迎贡献！请遵循以下流程：

- 先在 Issue 讨论需求或 Bug，再提交 PR。
- Fork 仓库并创建特性分支。
- 本地运行：`go mod tidy && go build ./... && go test ./...`。
- 遵循项目代码风格与 lint（可选安装 golangci-lint）。
- PR 请附带描述、测试结果与影响范围。

## 开发脚本
- `make build|run|test|docker-build|compose-up|compose-down`

## 行为准则
请遵守 `CODE_OF_CONDUCT.md`。
