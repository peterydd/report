# Changelog

> 所有项目显著变更记录在此文件。  
> 格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/)。  
> 升级指南：[docs/migration.md](docs/migration.md)

## [Unreleased]

## [1.1.0] - 2026-06-15

> 次版本：依赖注入架构演进 + 安全加固 + 生产部署套件。  
> 详细升级指南：[docs/migration.md#从-v106-升级到-v110](docs/migration.md)。

### Added

- **依赖注入端口**：`internal/app` 引入 `SpreadSheetCreator` / `MailFactory` /
  `ReportDeps`，新增 `NewAppWithDeps()` 构造函数，支持纯 Mock 单元测试
  无需真实 DB / SMTP
- **Mock 扩展**：`pkg/db` 新增 `MockDB.SetQueryError` / `SetStreamError` /
  `SetQueryPanic` / `SetStreamPanic`，支持按 SQL 注入错误或 panic
- **环境变量覆盖**：所有 `config.yaml` 字段均支持 `REPORT_*` 环境变量
  覆盖（点号转下划线）。敏感字段（SMTP 密码 / DB 密码）可不写入 yaml
- **路径遍历防御**：`Config.Validate` 拒绝 workbook `prefix` / `suffix`
  中包含 `/`、`\` 或 `..`
- **MIME header 注入防御**：`pkg/mail` 新增 `sanitizeHeaders` 剥离
   `\r\n`，防止 SMTP header 注入；`Subject` 改用 `mime.QEncoding`
- **TLS 加固**：`pkg/mail.Send` 强制 `tls.Config.MinVersion = TLS 1.2`
- **安全启动警告**：`InsecureSkipVerify=true` 时 `Auth()` 输出警告日志
- **生产部署套件**：
  - `deploy/docker/Dockerfile.production`（多阶段 scratch，~20 MB）
  - `deploy/docker/docker-compose.yaml` + `.env.example`（单机 one-shot）
  - `deploy/k8s/` 完整 k8s 清单（Namespace / ServiceAccount / ConfigMap /
    Secret / PVC / CronJob / kustomization）
  - `deploy/README.md` 部署指南、安全检查清单、故障排查
- **文档重写**：新增 `docs/{INDEX,configuration,operations,security,
  migration}.md`（19 篇重写 + 5 篇新增，总量 90 KB → 130 KB）

### Changed

- `internal/app` 引入 `goroutine panic recover`：单个 sheet panic 不再
  死锁 `WaitGroup` / `sheetChan`
- `internal/app` 引入 `len(rp.Sheets) == 0` 短路：防御 `NewAppWithDeps`
  路径下的信号量死锁
- `internal/app.Run` 在所有 sheet 失败时跳过整个报表并记录警告
- `pkg/mail.Send` 调用 `Auth()` 时即检查 `InsecureSkipVerify` 并打日志
- `configs/config.yaml.example` 标注 `REPORT_*` env 覆盖用法
- `Makefile` `VERSION` 默认值更新为 `v1.1.0`

### Fixed

- `internal/report` 包删除，全部逻辑迁移到 `internal/app`（DI 端口层）
- `pkg/db` / `pkg/mail` 集成测试在缺少真实环境时默认 `t.Skip`，
  需 `REPORT_INTEGRATION=1` 启用
- `pkg/db/db_test.go` 文件结构破坏导致的重复声明
- `pkg/excel` 早期 `pkg/report` 引用全部改为 `pkg/app`
- 镜像 tag 同步：所有 `report:1.0.x` → `report:1.1.0`

### Security

- **CVE 修复**：`github.com/go-viper/mapstructure/v2` v2.2.1 → **v2.5.0`
  修复 GO-2025-3787、GO-2025-3900（govulncheck 复测 0 可达）
- **MIME header 注入防御**：见 Added
- **路径遍历防御**：见 Added
- **TLS 最低版本强制**：见 Added
- **panic 恢复**：见 Changed
- `docs/security.md` 列出部署前自检清单

## [1.0.1] - 2026-06-06

### Fixed

- **重要** `pkg/mail`: 修正 BCC 收件人丢失（参数误传 `m.cc`）—— **强烈建议所有 1.0.0 用户升级**
- `internal/app`: sheet 并发查询无上限
- 多个集成测试在无真实环境时阻塞 CI

## [1.0.0] - 2024-XX-XX

### Added

- 初始版本发布
- 支持 MySQL, PostgreSQL, Oracle, ClickHouse
- Excel 多 sheet 报表生成（含冻结、SUM）
- SMTP 邮件发送（STARTTLS、附件、CC、BCC）
- YAML 配置 + viper 热重载
- Mock 测试支持（`db.MockDB` / `mail.MockMail`）
- 连接池配置 + 超时控制
- Dockerfile + Kubernetes CronJob

[Unreleased]: https://github.com/peterydd/report/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/peterydd/report/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/peterydd/report/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/peterydd/report/releases/tag/v1.0.0
