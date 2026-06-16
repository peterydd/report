# 贡献指南

感谢你对 `report` 项目的关注！我们欢迎各种形式的贡献：

- 🐛 提交 Bug 报告
- 💡 提交功能建议
- 🔧 提交代码修复
- 📖 改进文档
- 💬 分享使用经验

> 安全相关问题请直接走 [SECURITY.md](SECURITY.md)，**不要**在公开 Issue 中披露。

## 1. 报告 Bug

通过 [GitHub Issues](https://github.com/peterydd/report/issues/new?template=bug_report.md) 提交，并包含：

### 1.1 环境信息

```
- OS / Arch:        e.g. Ubuntu 22.04 amd64
- Go version:       go version go1.26.4 linux/amd64
- report version:   v1.1.0 (commit xxx)
- Database:         MySQL 8.0.32
- SMTP:             smtp.gmail.com:587
```

### 1.2 复现步骤

1. 配置 `config.yaml`（**脱敏后**附在 Issue 中）
2. 执行 `report`
3. 看到错误 `xxx`

### 1.3 预期 vs 实际

- **预期**：邮件正常发出
- **实际**：日志报 `email xxx failed: dial tcp: timeout`

### 1.4 错误日志

```text
2024/05/01 09:00:00 configuration loaded successfully
2024/05/01 09:00:01 email Daily Sales failed: ...
```

## 2. 提交功能建议

通过 [Feature Request 模板](https://github.com/peterydd/report/issues/new?template=feature_request.md)：

1. 描述使用场景与价值
2. 给出期望的 API / 配置示例
3. 标注是否愿意提交 PR

## 3. 提交代码

### 3.1 开发环境

1. Go 1.26.4+
2. Git 2.30+
3. Make（Windows: Git Bash / WSL）
4. Docker（可选）

详细：[docs/development.md § 环境准备](docs/development.md#1-环境准备)

### 3.2 流程

```bash
# 1. Fork & clone
git clone https://github.com/<you>/report.git
cd report

# 2. 同步主分支
git checkout main && git pull upstream main

# 3. 创建特性分支
git checkout -b feat/awesome-thing

# 4. 写代码 + 测试
$EDITOR ...

# 5. 质量门禁
make fmt
make vet
make test

# 6. 提交
git add -A
git commit -m "feat: add awesome thing"

# 7. 推送 + 开 PR
git push origin feat/awesome-thing
gh pr create --fill
```

### 3.3 代码规范

- 命名：包名小写单数；导出符号 PascalCase；未导出 camelCase
- 错误：`fmt.Errorf("...: %w", err)` 包装；不要吞错
- 并发：信号量限流（`chan struct{}` buffer）
- 资源：`defer Close()` + 错误日志
- 注释：导出符号必须有 godoc；建议中英双语

完整规范：[docs/development.md § 代码规范](docs/development.md#4-代码规范)

### 3.4 Commit 规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

- `type`：`feat` / `fix` / `docs` / `refactor` / `test` / `chore` / `perf` / `build` / `ci`
- `scope` 可选，如 `feat(db): add sqlserver driver`
- subject ≤ 72 字符
- body 解释 *what* 与 *why*
- footer 用 `Closes #123` / `BREAKING CHANGE: ...`

示例：

```
feat(mail): add OAuth2 SMTP authentication

- Support XOAUTH2 mechanism for Gmail / Office365
- New smtp.authType config field, default "PLAIN"
- Backward compatible

Closes #45
```

### 3.5 测试要求

- **新功能** 必须包含单元测试
- **修改行为** 必须更新现有测试
- 所有 PR 必须 `go test ./...` 通过
- 推荐同时 `REPORT_INTEGRATION=1 go test ./...`（需要真实环境）
- 关键包覆盖率 > 80%

### 3.6 文档要求

- 新配置字段 → 更新 [docs/configuration.md](docs/configuration.md) 与 `configs/config.yaml.example`
- 新 API → 更新对应 `pkg/*/README.md`
- 架构变更 → 更新 [docs/architecture.md](docs/architecture.md) 并在 PR 描述中链接
- 破坏性变更 → 更新 [docs/migration.md](docs/migration.md)
- 任何变更 → 更新 [CHANGELOG.md](CHANGELOG.md)（Unreleased）

## 4. Pull Request

### 4.1 标题与描述

- 标题简洁、动词开头
- 描述含：动机、改动摘要、影响范围、测试方式、关联 Issue
- 若有 UI 变更，附截图 / GIF

### 4.2 CI 检查

PR 触发 GitHub Actions：

- `go test ./...`（含 -race）
- `go vet ./...`
- `gofmt -l .` 必须无输出
- 镜像构建（仅 main 分支）

### 4.3 Review 流程

1. 至少 1 位维护者 approve
2. CI 全绿
3. 无未解决的 review comment
4. 提交者点击 **Squash and merge**

## 5. 本地开发循环

```bash
# 监视运行（开发期）
go run ./cmd/report -version

# 跑测试
go test -race ./...

# 覆盖率
make test-coverage

# 基准
make benchmark
```

## 6. 目录结构

```
.
├── cmd/report/        # 入口
├── internal/app/      # 业务编排
├── pkg/               # 公共库
│   ├── config/
│   ├── db/
│   ├── excel/
│   └── mail/
├── configs/           # 配置示例
├── docs/              # 设计文档
├── scripts/           # 脚本
└── test/              # 共享 fixture
```

详见 [DIRECTORY_STRUCTURE.md](DIRECTORY_STRUCTURE.md)。

## 7. 社区规范

请阅读 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。核心：

- 保持友好与尊重
- 接受建设性批评
- 关注社区最大利益
- 展现同理心

## 8. 联系方式

- 🐛 [Issues](https://github.com/peterydd/report/issues)
- 💬 [Discussions](https://github.com/peterydd/report/discussions)
- 🔒 安全：见 [SECURITY.md](SECURITY.md)

## 9. 许可证

通过提交代码，你同意贡献使用 [MIT License](LICENSE)。

---

再次感谢！🎉
