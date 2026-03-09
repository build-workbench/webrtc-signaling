# GitHub Pages 优化 (2026-03-10)

## Pages 构建优化

1. **pages.yml — sparse-checkout**：仅检出 `docs/` 目录，跳过 Go 源码、internal/、web/、k6/、docker/ 等，加速 CI 构建
2. **_config.yml — SEO 增强**：添加 `lang: zh-CN`，改善搜索引擎和浏览器语言识别
3. **_config.yml — kramdown 配置**：显式配置 rouge 语法高亮
4. **_config.yml — exclude 扩展**：排除 `*.go`、`*.mod`、`*.sum`、Dockerfile、Makefile 等非文档文件

## 文档内容优化

5. **docs/index.md — 着陆页增强**：添加"系统设计"按钮、补充角色权限行、`version` 字段、技术栈表格
6. **docs/index.md — 版权年份**：2025 → 2025-2026

## README 更新

7. **README.md + README.zh-CN.md**：添加 Pages workflow 徽章
