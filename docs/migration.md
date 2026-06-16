# 迁移与变更记录

> 跟踪所有破坏性变更、配置字段重命名、API 调整。  
> 升级前请查阅此文档，对照版本号定位影响。

## 语义化版本

- **MAJOR** — 破坏性变更，必读本文档
- **MINOR** — 向后兼容的新功能
- **PATCH** — 修复 / 文档

## 当前未发布版本

## 历史版本

### v1.0.1 — 2026-06-06

#### 修复

- `pkg/mail`: `buildRecipients` 误用 `m.cc` 重复，导致 BCC 收件人丢失  
  位置：`pkg/mail/mail.go:186` → 已改为 `m.bcc`  
  影响：**所有 1.0.0 用户** —— 配置中设置的 `bcc` 实际未投递
- `internal/app`: sheet 并发查询无上限，可能耗尽连接池  
  解决：新增 `maxConcurrentSheets=8` 信号量
- `pkg/db.TestDB` / `pkg/mail.TestSend` 在无真实环境下导致 `go test` 失败  
  解决：默认 `t.Skip`，需 `REPORT_INTEGRATION=1` 启用
- `pkg/config.TestNewConfig` / `internal/app.TestApp` 依赖 `config.yaml` 存在  
  解决：缺失时跳过
- `pkg/excel` `pkg/mail` 测试附件 `text/plain;charset=utf-8` 不是合法 xlsx MIME  
  解决：示例配置改为 `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `Makefile` clean 目标在 Windows 平台运行失败  
  解决：增加 `if exist` / `del /F /Q` 兼容分支

#### 文档

- 新增 `docs/INDEX.md` `docs/configuration.md` `docs/operations.md` `docs/security.md` `docs/migration.md`
- 重写 `README.md` / `README.en.md` / `docs/architecture.md` / `docs/development.md`
- 修正所有 `pkg/*/README.md` 顶部语言链接反向
- 重写 `CONTRIBUTING.md` / `ROADMAP.md` / `CHANGELOG.md` / `DIRECTORY_STRUCTURE.md`

#### 内部（不影响 API）

- `internal/app/app.go` 引入信号量；解析 `app.go` 时 `parseDuration` 函数前增加常量
- 移除 `internal/report/` 包，所有逻辑迁移到 `internal/app/`
- 升级依赖：Go 1.26.4、`excelize/v2 v2.9.0`、`viper v1.20.1`
- `cfg.yaml.example` attachment MIME 修正

### v1.1.0 — 2026-06-15

> 次版本：依赖注入架构演进 + 安全加固 + 生产部署套件。  
> **破坏性变更**：内部包路径调整（`internal/report` → `internal/app`）。  
> 公开 API（`pkg/*` 导出符号、`config.yaml` schema）保持兼容。

#### 破坏性变更

- 删除 `internal/report` 包，调用方如有 import 需改为 `internal/app`  
  ```diff
  -import "github.com/peterydd/report/internal/report"
  +import "github.com/peterydd/report/internal/app"
  ```
- `App` 公开字段（`App.Config`）保持不变；新增 `App.Deps` 与
  `NewAppWithDeps()` 构造函数。`pkg/app.NewApp()` 行为不变

#### 新增

- 依赖注入端口：`SpreadSheetCreator` / `MailFactory` / `ReportDeps`
  （在 `internal/app/ports.go`）
- 环境变量覆盖：所有 `config.yaml` 字段支持 `REPORT_*` 形式覆盖
  （点号转下划线）。详细列表见 `pkg/config/config.go:bindEnvOverrides`
- 生产部署套件：`deploy/docker/` + `deploy/k8s/` + `deploy/README.md`
- Mock 扩展：`MockDB.SetQueryError` / `SetStreamError` / `SetQueryPanic`
  / `SetStreamPanic`
- 部署前自检清单：`deploy/README.md § 4 安全检查清单`

#### 安全修复

- `mapstructure/v2` 升级 v2.2.1 → v2.5.0（GO-2025-3787、GO-2025-3900）
- MIME header 注入防御：`pkg/mail.Send` 调 `sanitizeHeaders` 剥离 `\r\n`
  ，`Subject` 改用 `mime.QEncoding`
- 路径遍历防御：`Config.Validate` 拒绝 workbook `prefix` / `suffix`
  含 `/` `\` `..`
- TLS 加固：`pkg/mail.Send` 强制 `tls.Config.MinVersion = TLS 1.2`
- panic 恢复：`internal/app.Run` sheet goroutine 内 `recover()`，
  单 sheet 失败不再死锁
- 零 sheet 死锁防御：`internal/app.Run` 跳过空报表
- `InsecureSkipVerify=true` 启动警告

#### 升级步骤

```bash
git pull
go mod tidy
make build VERSION=v1.1.0
```

#### 配置变化（推荐但非必须）

1. **敏感字段改用环境变量**：
   ```yaml
   # configs/config.yaml 不再包含密码
   smtp:
     password: ""          # 留空
   ```
   运行时：
   ```bash
   export REPORT_SMTP_PASSWORD='real-app-password'
   ./report
   ```

2. **镜像版本**：compose / k8s 清单默认 tag 已更新到 `1.1.0`。
   旧部署修改 image 字段或重新 `kubectl apply -k deploy/k8s`

3. **不再使用 `internal/report`**：如有自定义脚本 import，迁移到
   `internal/app`

#### 行为变化

1. **BCC 投递**（v1.0.1 修复）继续生效
2. **Sheet 并发上限**（v1.0.1 修复）继续生效
3. `go test ./...` 不再要求真实数据库/SMTP（v1.0.1 行为）
4. `MIME Subject` 在含非 ASCII 字符时显示为 RFC 2047 编码（之前是裸 UTF-8）
5. 强制 TLS 1.2：与旧版 SMTP 服务器（SSLv3 / TLS 1.0）不兼容时需升级服务端
6. workbook `prefix` / `suffix` 含 `/` `\` `..` 时启动直接报错

#### 兼容性承诺

- **配置文件**：1.x 内 schema 不破坏；新增字段保持向后兼容
- **Go API**：`pkg/*` 导出符号尽量保持稳定；破坏性变更将标记
  `@Deprecated` 一版本再删除
- **CLI**：仅增加，不删除既有 flag
- **镜像 / 二进制**：`vX.Y.Z` tag 永久保留

## 升级指南

### 从 v1.0.x 升级到 v1.1.0

**无必须的配置修改**。推荐步骤：

```bash
git pull
go mod tidy
make build VERSION=v1.1.0
```

可选优化（强烈建议生产环境）：

- 把 SMTP 密码移出 yaml，用 `REPORT_SMTP_PASSWORD` 环境变量
- 把镜像 tag 锁定到具体版本（已默认 `1.1.0`）
- 启用 `readOnlyRootFilesystem: true`（k8s 清单已默认）

行为变化（无需用户操作）：

1. 强制 TLS 1.2（与不支持的服务端会失败）
2. workbook prefix/suffix 路径分隔符被拒绝
3. Subject 中文用 RFC 2047 编码

### 从 v1.0.0 升级到 v1.0.1

**无需任何配置修改**。升级步骤：

```bash
git pull
go mod tidy
make build VERSION=v1.0.1
```

行为变化：

1. BCC 现在能正常投递（v1.0.0 实际只发给了 CC + To）
2. sheet 并发上限变为 8（之前无上限）
3. `go test ./...` 不再要求真实数据库/SMTP
4. Windows 上 `make clean` 正常工作

### 从 v0.x 升级到 v1.0

> v0.x 内部版本，不公开提供迁移路径。

## 兼容性承诺

- **配置文件**：1.x 内不会破坏；新增字段保持向后兼容
- **Go API**：1.x 内 `pkg/*` 导出符号尽量保持稳定；破坏性变更将标记 `@Deprecated` 一版本再删除
- **CLI**：仅增加，不删除既有 flag
- **镜像 / 二进制**：`vX.Y.Z` tag 永久保留

## 升级前 Checklist

- [ ] 读 [CHANGELOG.md](../CHANGELOG.md) 与本文件
- [ ] 在测试环境跑 `make test`
- [ ] 灰度一台服务器运行 `make run`（或 `kubectl apply` 一次 CronJob）
- [ ] 检查日志确认 sheet / 报表数与之前一致
- [ ] 收件人侧确认收到邮件
- [ ] 24 小时后全量推送
