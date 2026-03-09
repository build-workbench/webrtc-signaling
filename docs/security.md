---
title: 安全策略
layout: default
nav_order: 6
description: "漏洞报告与处理流程"
---

# 安全策略

## 支持版本

| 版本 | 支持状态 |
|:--|:--|
| v0.1.x | 当前支持 |

## 报告漏洞

{: .warning }
**请不要在公开 Issue 中报告安全漏洞。**

请通过以下方式私下联系维护者：

1. **GitHub Security Advisories** — 在仓库 Security 标签页中 [创建安全公告](https://github.com/LessUp/aurora-signal/security/advisories/new)
2. **邮件** — 发送至项目维护者邮箱（详见仓库 Profile）

报告中请包含：
- 漏洞描述与影响范围
- 复现步骤（尽可能详细）
- 受影响的版本
- 可能的修复建议（如有）

## 处理流程

| 阶段 | 时间 |
|:--|:--|
| 确认收到并初步响应 | **48 小时内** |
| 评估影响与制定修复计划 | 1 周内 |
| 发布补丁 | 视严重程度而定 |
| 公开披露 | 补丁发布后（视情况延迟） |

## 安全最佳实践

使用 Aurora Signal 时，建议遵循以下安全实践：

- **JWT Secret** — 使用高强度随机密钥，不少于 32 字符
- **TLS** — 生产环境强制 HTTPS / WSS
- **CORS** — 配置 `SIGNAL_ALLOWED_ORIGINS` 限制受信来源
- **Admin Key** — 如启用管理 API，务必配置 `SIGNAL_ADMIN_KEY`
- **速率限制** — 保持默认限流配置或根据负载调整
- **Redis** — 使用密码认证，避免暴露到公网
