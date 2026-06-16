# 目录结构

> 本目录遵循 [Standard Go Project Layout](https://github.com/golang-standards/project-layout)。  
> 详细架构与设计：[docs/architecture.md](docs/architecture.md)

## 顶层结构

```
report/
├── cmd/
│   └── report/
│       └── main.go                  # 程序入口（仅 main 函数）
│
├── internal/
│   └── app/                         # 业务编排
│       ├── app.go                   # App / Run / Start
│       ├── app_test.go              # parseDuration 等单元测试
│       └── integration_test.go      # 端到端 Mock 流程
│
├── pkg/                             # 公共库（可被外部导入）
│   ├── config/
│   │   ├── config.go                # Config 结构 + viper + 校验
│   │   ├── config_test.go
│   │   ├── README.md / README.en.md
│   │
│   ├── db/
│   │   ├── db.go                    # 4 驱动 + 接口 + 工厂
│   │   ├── mock_db.go               # MockDB
│   │   ├── db_test.go               # 集成（默认 skip）
│   │   ├── db_mock_test.go          # Mock 单元测试
│   │   ├── test.xlsx                # 测试附件
│   │   ├── README.md / README.en.md
│   │
│   ├── excel/
│   │   ├── excel.go                 # excelize 封装
│   │   ├── excel_test.go
│   │   ├── test.xlsx                # 测试输出
│   │   ├── README.md / README.en.md
│   │
│   └── mail/
│       ├── mail.go                  # SMTP 实现
│       ├── mock_mail.go             # MockMail
│       ├── mail_test.go             # 集成（默认 skip）
│       ├── test.xlsx                # 测试附件
│       ├── README.md / README.en.md
│
├── scripts/
│   └── build.sh                     # 跨平台构建脚本
│
├── docs/                            # 设计文档
│   ├── INDEX.md                     # 文档总索引
│   ├── architecture.md
│   ├── configuration.md
│   ├── development.md
│   ├── operations.md
│   ├── security.md
│   └── migration.md
│
├── test/
│   ├── README.md
│   └── fixtures/
│       └── config/
│           └── test.yaml            # 集成配置示例
│
├── configs/
│   └── config.yaml.example
│
├── build/                           # 构建输出（gitignore）
│
├── .github/
│   ├── workflows/
│   │   └── ci.yml                   # GitHub Actions
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   ├── feature_request.md
│   │   └── config.yml
│   └── PULL_REQUEST_TEMPLATE.md
│
├── .gitignore
├── CHANGELOG.md
├── CODE_OF_CONDUCT.md
├── CONTRIBUTING.md
├── DIRECTORY_STRUCTURE.md           # 本文件
├── Dockerfile
├── LICENSE
├── Makefile
├── README.md / README.en.md
├── ROADMAP.md
├── SECURITY.md
├── VERSION
├── go.mod / go.sum
```

## 分层与依赖

```
cmd/report ──▶ internal/app ──▶ pkg/{db,excel,mail,config}
```

- `cmd/report` 不放业务逻辑，仅 flag 解析
- `internal/app` 编排层：连接池、并发、邮件发送
- `pkg/*` 互相独立，仅通过标准库协作

## 目录约定

### `/cmd`

- 每个可执行程序一个目录
- 目录名 = 二进制名
- 严禁放业务代码

### `/internal`

- 私有应用代码，Go 编译器强制保护
- **只有** `internal/app`，承载业务编排
- 历史曾用 `internal/report`，已在 v1.0.x 迁移（见 [docs/migration.md](docs/migration.md)）

### `/pkg`

- 可被外部项目导入的公共库
- 每个子目录是独立包，自带：
  - 实现（`*.go`）
  - 单元测试（`*_test.go`）
  - Mock（`mock_*.go` 或 `*_mock.go`）
  - README（中英）

### `/scripts`

- 构建/发布/部署脚本
- 当前：`build.sh`（跨平台）

### `/docs`

- 设计与运维文档
- 入口：[docs/INDEX.md](docs/INDEX.md)

### `/test`

- 共享 fixture / 数据
- 单元测试在各自包内
- 集成测试可放此目录或包内带 `t.Skip` 保护

### `/configs`

- 配置示例
- 不放真实凭证

### `/build`

- 构建输出
- 已加入 `.gitignore`

### `/.github`

- CI / Issue 模板 / PR 模板

## 命名约定

### Go 文件

- 小写 + 下划线
- 测试：`xxx_test.go`
- Mock：`mock_xxx.go` 或 `xxx_mock.go`

### 文档文件

- PascalCase（`README.md`、`CHANGELOG.md`）
- 全部 Markdown

### 包名

- 小写、单数、有意义
- 避免 `util` / `common` / `helpers`

## 优化历史

### 2026-06-06（v1.0.1）

- ✅ 文档中心化：新增 5 篇设计文档
- ✅ 修正所有 `pkg/*/README.md` 顶部语言链接
- ✅ 修正 `internal/report` → `internal/app` 引用一致性
- ✅ 清理残留 `test.xlsx` 与空 `api/` 目录

### 2024-02-06（v0.x 末期）

- ✅ 清理重复 mock 文件
- ✅ 创建 `scripts/` `docs/` `test/` `build/` 目录
- ✅ 为每个包添加 README

## 关联

- [README.md](README.md) — 项目主入口
- [docs/architecture.md](docs/architecture.md) — 分层与并发模型
- [docs/development.md](docs/development.md) — 本地开发与代码规范
