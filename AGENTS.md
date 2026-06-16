# AGENTS.md — 项目工作上下文

> 本文件供 opencode / 其他 AI 助手在 IDE 重启后快速接续工作。  
> 用户偏好：中文交流；中文 commit；风格简洁、避免冗长解释。

## 1. 项目概况

**项目名**：`report` (模块路径 `github.com/peterydd/report`)  
**类型**：Go 编写的高性能数据报表生成 + 邮件发送 CLI 工具  
**Go 版本**：`1.26.4`（已升级；`go.mod` 第 3 行 `go 1.26.4`）  
**平台**：Windows（PowerShell 5.1），但代码要跨平台  
**最近一次提交**：`99acc45 修复bug`（在 main 分支）  
**当前状态**：大量本地未提交改动（见 §6）

## 2. 用户当前的工作模式

1. 用户给出任务（中文）
2. 我先 `question` 工具确认范围/深度（除非任务明确）
3. 用 `todowrite` 拆分子任务
4. 一次性批量调用工具（独立操作并行，依赖操作串行）
5. 关键节点跑 `go vet` / `go build` / `go test` 验证
6. 不主动 `git commit`，等用户明确指示

## 3. 关键路径速查

```
D:\codes\report\
├── cmd/report/main.go               # CLI 入口（74 行）
├── internal/app/                    # 业务编排（替代原 internal/report/）
│   ├── app.go                       # App 结构 + Run()
│   ├── app_test.go                  # parseDuration 单元测试
│   └── integration_test.go          # 端到端 Mock 测试
├── pkg/
│   ├── config/                      # viper 配置 + 校验 + 热重载
│   ├── db/                          # 4 数据库 + MockDB
│   ├── excel/                       # excelize 封装
│   └── mail/                        # SMTP + MockMail
├── configs/config.yaml.example      # 配置示例
├── docs/                            # 设计文档
├── scripts/build.sh                 # 跨平台构建
├── test/fixtures/                   # 共享 fixture
├── Dockerfile                       # 多阶段构建
├── Makefile                         # 任务入口
└── go.mod / go.sum
```

## 4. 已知问题与修复记录（已完成）

| 问题 | 修复位置 |
|------|----------|
| `pkg/mail/mail.go:186` `buildRecipients` 误用 `m.cc` 重复，导致 BCC 失效 | 已改为 `m.bcc` |
| `internal/app/app.go` sheet 并发无上限 | 引入 `maxConcurrentSheets=8` 信号量 |
| `pkg/db.TestDB` / `pkg/mail.TestSend` 真实环境失败 | 加 `t.Skip`，需 `REPORT_INTEGRATION=1` 启用 |
| `pkg/config.TestNewConfig` 依赖 config.yaml | 缺失时跳过 |
| `internal/app.TestApp` 同样依赖 | 改为 `parseDuration` 单元测试 |
| `integration_test.go:168` `mail.SetMessage` 返回值未使用 | 修正使用 msg |
| `pkg/db/db_test.go` 文件结构损坏（重复声明） | 重写整个文件 |
| `configs/config.yaml.example` attachment MIME 错 | 改为 `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` |
| Makefile `clean` 在 Windows 失败 | 加 `if exist` / `del` 兼容分支 |
| `pkg/config/README*`、`pkg/db/README*`、`pkg/excel/README*`、`pkg/mail/README*` 顶部语言链接反向 | 已修正 |
| `docs/architecture.md` 引用已删除的 `internal/report` | 同步到 `internal/app` |
| `docs/development.md` 同样问题 | 已修正 |
| `CONTRIBUTING.md` 同样问题 | 已修正 |
| `SECURITY.md` `security@example.com` 与 README 邮箱不一致 | 统一为 `peterydd@example.com` |
| 空 `api/proto/` `api/v1/` 目录 | 已删除 |

## 5. 文档结构（27 个 Markdown）

### 入口
- `README.md`（中文主入口）· `README.en.md`（英文）

### docs/
- `INDEX.md`（**新增** 文档总索引）
- `architecture.md`（分层、并发模型、决策记录）
- `configuration.md`（**新增** 完整配置参考）
- `development.md`（本地开发、代码规范、troubleshooting）
- `operations.md`（**新增** 部署、监控、故障排查）
- `security.md`（**新增** 安全实践）
- `migration.md`（**新增** 迁移记录）

### 包文档（中英）
- `pkg/{config,db,excel,mail}/README.md` + `README.en.md`

### 治理
- `CHANGELOG.md` `ROADMAP.md` `CONTRIBUTING.md` `SECURITY.md` `CODE_OF_CONDUCT.md` `DIRECTORY_STRUCTURE.md`
- `test/README.md`

### .github/
- `workflows/ci.yml`（GO_VERSION=1.26.4）
- `ISSUE_TEMPLATE/{bug_report,feature_request}.md` + `config.yml`
- `PULL_REQUEST_TEMPLATE.md`

## 6. 当前未提交的本地改动

**修改的文件**（17）：
```
.gitignore
Dockerfile
Makefile
README.md
cmd/report/main.go
configs/config.yaml.example
go.mod (1.26.4)
go.sum
internal/report/report.go        (deleted)
internal/report/report_test.go   (deleted)
pkg/config/config.go
pkg/config/config_test.go
pkg/db/db.go
pkg/db/db_test.go
pkg/excel/excel.go
pkg/excel/excel_test.go
pkg/mail/mail.go
pkg/mail/mail_test.go
```

**新增的文件**（约 30）：
```
.github/  CHANGELOG.md  CODE_OF_CONDUCT.md  CONTRIBUTING.md
DIRECTORY_STRUCTURE.md  README.en.md  ROADMAP.md  SECURITY.md  VERSION
docs/INDEX.md  docs/architecture.md  docs/configuration.md  docs/development.md
docs/migration.md  docs/operations.md  docs/security.md
internal/app/{app.go, app_test.go, integration_test.go}
pkg/config/README.md  pkg/config/README.en.md
pkg/db/README.md  pkg/db/README.en.md  pkg/db/db_mock_test.go  pkg/db/mock_db.go
pkg/excel/README.md  pkg/excel/README.en.md  pkg/excel/test.xlsx
pkg/mail/README.md  pkg/mail/README.en.md  pkg/mail/mock_mail.go  pkg/mail/test.xlsx
scripts/build.sh
test/README.md  test/fixtures/config/test.yaml
```

> **注意**：以上未提交不是错误状态，是用户偏好（不主动 commit）。

## 7. 验证命令

```powershell
cd D:\codes\report

# 静态检查
go vet ./...

# 编译
go build -o report.exe ./cmd/report

# 单元测试（默认跳过集成）
go test -count=1 ./...

# 集成测试（需真实 DB / SMTP）
$env:REPORT_INTEGRATION=1; go test -count=1 ./...

# 覆盖率
make test-coverage
```

**预期结果**：5 个包全 `ok`，主二进制 ~34 MB。

## 8. 风格偏好

- 中文交流、简洁回答
- `todowrite` 跟踪 3+ 步任务
- 工具调用：独立操作并行，依赖操作串行
- 一次回复 < 4 行非工具文本（除非用户要求详细）
- 错误处理：包内用 `fmt.Errorf("...: %w", err)`，CLI 层用 `log.Fatalf`
- 并发：用 buffered channel 作信号量
- 资源释放：`defer Close()` + 错误日志
- 不主动 `git commit` / `git push` / 创建 PR

## 9. 上次完成的工作

**任务**：完善项目文档说明信息（用户确认"彻底重写所有文档"）

**结果**：重写 19 篇 + 新增 5 篇；总文档量 90KB → 130KB；全部内部链接验证通过；构建/测试保持绿。

**Go 版本升级**：从 1.25.4 → 1.26.4（go.mod + Dockerfile + CI + 10 处文档）。

## 10. 下次可能的工作方向

候选（等用户明确指示）：

1. **提交当前改动并打 tag**：`git add -A && git commit -m "feat: 全面文档化与 Go 1.26.4 升级" && git tag v1.0.1`
2. **实现 ROADMAP v1.1 项**：邮件模板 / worker pool / StreamWriter / 重试
3. **依赖注入改造** `internal/app` 让其可纯 Mock 测试
4. **添加 CI 缓存** 让 `make test` 更快
5. **补 SQL Server / 达梦等驱动**（如果有用户需求）
6. **写更多 E2E 测试**：本地 docker-compose 起 MySQL + mailhog

## 11. 关键引用片段

`internal/app/app.go` 信号量（不要删除）：
```go
const maxConcurrentSheets = 8

// ...
sem := make(chan struct{}, concurrency)
for _, st := range rp.Sheets {
    wg.Add(1)
    sem <- struct{}{}
    go func(sheetConfig *config.Sheet) {
        defer wg.Done()
        defer func() { <-sem }()
        // ... QueryStream / Query
    }(st)
}
```

`pkg/mail/mail.go` BCC 修复（不要回退）：
```go
rcpt := buildRecipients(m.to, m.cc, m.bcc)  // 第三个参数是 bcc
```

集成测试跳过模式：
```go
if os.Getenv("REPORT_INTEGRATION") == "" {
    t.Skip("set REPORT_INTEGRATION=1 to run against real service")
}
```

---

最后更新：2026-06-06  
由 opencode 维护
