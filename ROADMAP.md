# 路线图 (Roadmap)

> 本文档描述 `report` 项目的未来规划。  
> 时间表是 **预期**，实际节奏取决于社区贡献与维护者带宽。

## 状态图例

- ✅ 已完成
- 🚧 进行中
- 📋 计划中
- 💡 提案 / 待评估

## 当前版本 v1.1.x

### ✅ v1.0.0（2024）

- 多数据库支持：MySQL / PostgreSQL / Oracle / ClickHouse
- Excel 多 sheet 报表生成
- SMTP 邮件发送（STARTTLS、附件、CC、BCC）
- YAML 配置 + viper 热重载
- Docker + Kubernetes CronJob 部署
- Mock 测试支持
- 流式查询（百万行低内存）

### ✅ v1.0.1（2026-06-06）

- 修复 BCC 收件人丢失（[GH#1](https://github.com/peterydd/report/issues)）
- sheet 并发限流（信号量 8）
- 集成测试默认跳过
- Windows 兼容

### ✅ v1.1.0（2026-06-15）

> 架构演进 + 安全加固 + 生产部署套件。详见
> [CHANGELOG](CHANGELOG.md) / [docs/migration.md](docs/migration.md)。

**架构**

- `internal/app` 依赖注入端口（`SpreadSheetCreator` / `MailFactory` / `ReportDeps`）
- `NewAppWithDeps()` 构造函数支持纯 Mock 单元测试
- `MockDB` / `MockMail` 增强（按 SQL 注入错误 / panic）

**安全**

- `REPORT_*` 环境变量覆盖 yaml（敏感字段不落盘）
- MIME header 注入防御（`sanitizeHeaders` + Subject QEncoding）
- workbook 路径遍历防御（`validateWorkBookName`）
- TLS 最低版本强制 1.2
- goroutine panic 恢复 + 零 sheet 死锁防御
- `mapstructure/v2` 升级 v2.2.1 → v2.5.0 修复 2 CVE

**部署**

- `deploy/docker/Dockerfile.production`（多阶段 scratch）
- `deploy/docker/docker-compose.yaml`（单机）
- `deploy/k8s/` 完整 CronJob + ConfigMap + Secret
- `deploy/README.md` 安全检查清单

**文档**

- 19 篇重写 + 5 篇新增（90 KB → 130 KB）
- `docs/INDEX.md` 文档中心

## v1.2 — 可靠性与可观测性

> 目标：在 v1.1 安全基线之上补齐可观测性与失败恢复能力。

### 📋 功能

- 邮件模板（HTML + 变量插值）
- 邮件发送 worker pool（并发发送）
- Excel 流式写 `excelize.StreamWriter`（O(1) 内存）
- 失败重试（指数退避 + 抖动）
- BCC 隐私强化（从 `Bcc:` 头移除）
- 主题/正文模板中的 `{{date}}` 等占位符
- 配置变更通知（`SetOnChange` 主动回调）
- 自定义冻结行列

### 📋 可观测性

- 结构化日志（zap/zerolog，可选）
- Prometheus metrics 导出
- 审计日志 JSON 行
- OpenTelemetry tracing（DB / SMTP 段）

### 📋 配置

- 每报表独立 DSN（覆盖全局）
- 报表依赖声明（顺序执行）
- Dry-run 模式（只生成 xlsx，不发邮件）

### 💡 评估中

- 切换到 `gomail.v2`（维护性 vs. 兼容性）
- 支持 OAuth2 SMTP（Gmail 等）

## v1.3 — 任务调度与 HTTP API

> 目标：把工具从「单次 CLI」演进为「可被编排的服务」。

### 📋 功能

- 内置 HTTP API：
  - `POST /reports/run` 触发执行
  - `GET  /reports/{id}/status` 查询状态
  - `GET  /reports/{id}/artifact` 下载 xlsx
- Grafana 仪表板 JSON
- 告警通道：邮件 / Slack / 钉钉 / Webhook

### 📋 安全

- API Key 鉴权
- 审计日志持久化到文件 / DB
- 密钥保险箱（Hashicorp Vault 适配器）

## v1.4 — 扩展性

> 目标：允许社区贡献驱动 / 导出器 / 处理器。

### 📋 插件系统

- 驱动插件（自定义数据源）
- 导出器插件（PDF / CSV / HTML）
- 处理器插件（行级 transform）
- 通过 Go `plugin` 包或 Wasm 加载

### 📋 多数据源

- HTTP API 数据源
- CSV 文件数据源
- 多个数据源 JOIN

## v2.0 — 分布式 / 云原生

> 目标：水平扩展、企业级。

### 📋 分布式

- 任务队列（Redis / NATS）
- 集群模式（多副本分片）
- Leader 选举（保证同一报表不会并发）

### 📋 云原生

- Helm Chart
- Operator（CRD 定义 `Report`）
- 自动扩缩容（HPA）
- 多租户隔离

### 📋 生态

- Airflow Operator
- DataX Reader
- Python / Java SDK

## 长期愿景

- 可视化报表设计器（Web UI）
- AI 辅助：自然语言生成 SQL / 配置
- 完整的报表订阅平台

## 社区驱动

我们根据以下因素确定优先级：

1. **用户需求** — GitHub Issues / Discussions 投票
2. **技术债务** — 重构 / 安全
3. **生态兼容** — Go 1.x 新特性、依赖升级
4. **维护成本** — 长期可持续

### 投票方式

在 [Discussions → Roadmap](https://github.com/peterydd/report/discussions/categories/roadmap) 中用 👍 反应你最期待的功能。

## 已废弃的方向

- ❌ 自建 ORM — 不会做，保持 `database/sql` 直接调用
- ❌ 浏览器端预览 — 不会做，专注后端
- ❌ 完整的 BI 平台 — 不会做，定位是「数据→邮件」

## 版本与发布

- **MAJOR**：破坏性变更（如 v2.0 分布式）
- **MINOR**：新功能（v1.1 / v1.2 / v1.3）
- **PATCH**：修复 / 文档 / 内部重构

发布频率：PATCH 按需、MINOR 每 1-2 个季度、MAJOR 看里程碑达成。

## 关联

- [CHANGELOG.md](CHANGELOG.md) — 历史变更
- [docs/migration.md](docs/migration.md) — 升级指南
- [CONTRIBUTING.md](CONTRIBUTING.md) — 参与贡献
