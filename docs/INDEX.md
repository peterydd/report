# 文档中心

> 所有项目文档的索引页。新贡献者请按 1 → 2 → 3 顺序阅读。

## 用户文档

| 文档 | 说明 | 适合谁 |
|------|------|--------|
| [README.md](../README.md) | 项目主入口：功能、快速开始、配置、部署、FAQ | 所有用户 |
| [README.en.md](../README.en.md) | 英文版主入口 | 英文用户 |
| [docs/configuration.md](configuration.md) | 完整配置参考（每个字段的含义与示例） | 运维、配置管理员 |
| [docs/operations.md](operations.md) | 部署、日志、监控、备份、故障排查 | SRE、运维 |
| [docs/security.md](security.md) | 安全实践、密钥管理、容器安全 | 安全工程师、运维 |
| [docs/migration.md](migration.md) | 历史迁移记录与破坏性变更说明 | 升级者 |

## 设计文档

| 文档 | 说明 |
|------|------|
| [docs/architecture.md](architecture.md) | 分层、并发模型、数据结构、错误处理、扩展点、决策记录 |
| [docs/development.md](development.md) | 本地开发、代码规范、测试策略、调试、提交规范 |

## 流程与社区

| 文档 | 说明 |
|------|------|
| [CONTRIBUTING.md](../CONTRIBUTING.md) | 贡献流程与规范 |
| [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md) | 社区行为准则 |
| [SECURITY.md](../SECURITY.md) | 报告安全漏洞的流程 |
| [CHANGELOG.md](../CHANGELOG.md) | 版本变更日志 |
| [ROADMAP.md](../ROADMAP.md) | 未来规划 |
| [DIRECTORY_STRUCTURE.md](../DIRECTORY_STRUCTURE.md) | 目录约定 |

## 包级文档

每个 `pkg/*` 子目录都有自己的 README（中英双语）：

| 包 | 文档 |
|----|------|
| `pkg/config` | [中文](../pkg/config/README.md) · [English](../pkg/config/README.en.md) |
| `pkg/db`     | [中文](../pkg/db/README.md) · [English](../pkg/db/README.en.md) |
| `pkg/excel`  | [中文](../pkg/excel/README.md) · [English](../pkg/excel/README.en.md) |
| `pkg/mail`   | [中文](../pkg/mail/README.md) · [English](../pkg/mail/README.en.md) |

## 文档约定

- 主 README 是「门面」，新用户第一个看
- `docs/` 放「深度内容」：架构、配置全解、运维、安全
- `pkg/*/README.md` 放「API 速查」：导出符号 + 用法
- 所有示例代码必须 `gofmt` 通过且能在 `go test` 中编译

## 文档更新规则

1. **新功能** → 必须同步更新 `README.md`（或对应子包 README） + `CHANGELOG.md`（Unreleased）
2. **破坏性变更** → 必须更新 `docs/migration.md`
3. **新配置字段** → 必须更新 `docs/configuration.md` 与 `configs/config.yaml.example`
4. **架构变更** → 必须更新 `docs/architecture.md`，并在 PR 描述中链接到对应章节
5. **安全相关** → 必须更新 `docs/security.md`，并附 CVE 编号（如有）

## 版本说明

- 当前最新稳定版：**v1.1.0**（2026-06-15）
- 文档与代码 **同版本号** 演进
- 文档过时（与代码不符）请开 PR 标注 `docs:` 前缀
- 重大版本切换会保留旧版文档于 `docs/archive/`（暂未启用）
