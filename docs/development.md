# 开发指南

> 面向贡献者与二次开发者的完整工作流。先读 [架构文档](architecture.md) 再看本篇。

## 1. 环境准备

### 1.1 必备工具

| 工具 | 最低版本 | 用途 |
|------|----------|------|
| Go   | 1.26.4+ | 编译、测试 |
| Git  | 2.30+    | 版本控制 |
| Make | 任意     | `make` 任务（Windows 可用 Git Bash / WSL） |
| Docker (可选) | 20+ | 镜像构建、e2e |

### 1.2 推荐 IDE

- **VS Code** + `golang.go` 扩展
- **GoLand** 内置开箱即用

`.vscode/launch.json` 示例：

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch report",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/report",
      "env": {
        "CONFIG_PATH": "${workspaceFolder}"
      }
    }
  ]
}
```

### 1.3 拉取代码

```bash
git clone https://github.com/peterydd/report.git
cd report
go mod download
```

## 2. 目录速览

```
.
├── cmd/report/        # 程序入口（仅 main.go）
├── internal/app/      # 编排：连接池 / 并发 / 邮件
├── pkg/
│   ├── config/        # viper 配置 + 校验
│   ├── db/            # 4 驱动 + Mock
│   ├── excel/         # excelize 封装
│   └── mail/          # SMTP + Mock
├── configs/           # 配置示例
├── docs/              # 设计文档（本目录）
├── scripts/           # 构建/发布脚本
├── test/fixtures/     # 集成测试数据
└── Makefile           # 任务入口
```

详细约定见 [DIRECTORY_STRUCTURE.md](../DIRECTORY_STRUCTURE.md)。

## 3. 工作流

### 3.1 标准循环

```bash
# 1. 同步上游
git checkout main && git pull

# 2. 创建分支
git checkout -b feat/awesome-thing

# 3. 写代码
$EDITOR pkg/excel/excel.go

# 4. 质量门禁（必跑）
make fmt         # gofmt -w -s .
make vet         # go vet ./...
make test        # go test -race ./...

# 5. 提交
git add -A
git commit -m "feat: support frozen second column"

# 6. 推送并开 PR
git push origin feat/awesome-thing
gh pr create --fill
```

### 3.2 本地端到端验证

```bash
cp configs/config.yaml.example config.yaml
# 修改 config.yaml 指向你本地的 MySQL + SMTP
./report
ls -lh *.xlsx   # 检查输出
```

## 4. 代码规范

### 4.1 命名

- 包名：单数、小写、有意义（`config` / `db` / `excel` / `mail`）
- 文件名：小写 + 下划线；测试文件以 `_test.go` 结尾
- 类型：导出用 PascalCase（`ConnPoolConfig`），未导出用 camelCase
- 错误变量：以 `Err` 前缀（建议）

### 4.2 错误处理

```go
// ✅ 包装上下文 + 保留 cause
if err := db.Connect(dsn, cfg); err != nil {
    return fmt.Errorf("connect %s: %w", driver, err)
}

// ❌ 吞错
if err != nil { /* ignore */ }

// ❌ 仅字符串
return errors.New("something wrong")
```

### 4.3 并发

- 多 goroutine 共享资源 → 用 `sync.Mutex` / `sync.RWMutex`
- 多 goroutine 等待完成 → `sync.WaitGroup` + `defer wg.Done()`
- 限流 → **buffered channel 作为信号量**：

```go
sem := make(chan struct{}, maxConcurrent)
for _, item := range items {
    wg.Add(1)
    sem <- struct{}{}
    go func(x T) {
        defer wg.Done()
        defer func() { <-sem }()
        // work on x
    }(item)
}
wg.Wait()
```

### 4.4 资源释放

- `defer rows.Close()` 后必须 `if err := rows.Err(); err != nil`
- `defer f.Close()` 配合错误日志
- SMTP 客户端在任意错误分支都 `c.Close()`

### 4.5 注释

- 导出符号必须有 godoc 注释（`golint` 检查）
- 中文 / 英文双语注释更佳（参考现有包风格）

## 5. 测试策略

### 5.1 单元测试

每个 `pkg` 至少应有：
- 主入口/正常路径
- 错误路径
- 边界（空值、超大值）

```go
func TestParseDBType(t *testing.T) {
    cases := []struct{ in string; want DBType }{
        {"mysql", MYSQL},
        {"postgresql", POSTGRESSQL},
        {"unknown", -1},
    }
    for _, c := range cases {
        if got := ParseDBType(c.in); got != c.want {
            t.Errorf("ParseDBType(%q) = %d, want %d", c.in, got, c.want)
        }
    }
}
```

### 5.2 Mock 测试

```go
func TestRun_WithMock(t *testing.T) {
    db := db.NewMockDB()
    db.SetQueryResult("SELECT 1", [][]interface{}{ {"a"} })

    mail := mail.NewMockMail()

    // 注入到 app（见下文「依赖注入」）
    err := app.NewAppWithDeps(cfg, db, mail).Run()
    if err != nil { t.Fatal(err) }
    if mail.GetSendCount() != 1 { t.Errorf("want 1 email sent") }
}
```

### 5.3 集成测试

需要真实环境，默认 **跳过**：

```go
func TestDB(t *testing.T) {
    if os.Getenv("REPORT_INTEGRATION") == "" {
        t.Skip("set REPORT_INTEGRATION=1 to run against real database")
    }
    // ...
}
```

启用方式：

```bash
REPORT_INTEGRATION=1 go test ./...
```

### 5.4 基准测试

```go
func BenchmarkQueryStream(b *testing.B) {
    db := db.NewMockDB()
    // 准备大数据集
    var rows [][]interface{}
    for i := 0; i < 1_000_000; i++ {
        rows = append(rows, []interface{}{i, i*2})
    }
    db.SetStreamResult("SELECT * FROM big", rows)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = db.QueryStream("SELECT * FROM big", func(r []interface{}) error { return nil }, 10_000)
    }
}
```

## 6. 常见任务

### 6.1 添加新的数据库驱动

1. 在 `pkg/db/db.go` 追加常量：

```go
const (
    ORACLE DBType = iota
    MYSQL
    POSTGRESSQL
    CLICKHOUSE
    SQLSERVER   // 新增
)
```

2. 更新 `ParseDBType` 与 `NewDBFactory`
3. 实现 `SQLServerDB` 结构体与 `Connect` 等方法
4. `go get github.com/microsoft/go-mssqldb`
5. 添加 Mock + 单元 + 集成测试
6. 更新 `docs/configuration.md` 的连接串示例

### 6.2 添加新的导出格式（PDF / CSV）

参考目录约定：

```
pkg/pdf/
├── pdf.go
├── pdf_test.go
└── README.md
```

接口对齐 `excel.SpreadSheet`：

```go
type Exporter interface {
    Create() error
}
```

在 `internal/app` 中根据 `report.format` 字段分派：

```go
var ex Exporter
switch cfg.Format {
case "xlsx": ex = excel.NewSpreadSheet(name, sheets)
case "pdf":  ex = pdf.NewPDF(name, sheets)
default:     return fmt.Errorf("unknown format: %s", cfg.Format)
}
return ex.Create()
```

### 6.3 依赖注入（让 app 可被无副作用测试）

将 `internal/app/app.go` 改造为：

```go
type App struct {
    Config *config.Config
    DB     db.DB         // 注入
    Mail   mail.Mail     // 注入
}

func NewAppWithDeps(cfg *config.Config, d db.DB, m mail.Mail) *App {
    return &App{Config: cfg, DB: d, Mail: m}
}
```

`Run()` 中使用 `a.DB` / `a.Mail`，测试时传入 `MockDB` / `MockMail` 即可端到端验证。

### 6.4 调试邮件发送

- 临时把 `smtp.host` 指向 `mailhog`/`smtp4dev` 容器
- 抓包：

```bash
tcpdump -i lo -w smtp.pcap port 587
```

- 用 `mail.SetError` 强制 `MockMail` 返回错误，验证编排层的失败分支

## 7. 故障排查

| 症状 | 可能原因 | 排查方法 |
|------|----------|----------|
| `failed to read configuration file` | 路径找不到 | `ls configs/` / `echo $PWD` |
| `unsupported database driver` | `driver` 拼写错 | 见 `db.ParseDBType` 支持值 |
| Sheet 失败但 Run 仍返回 nil | 故意行为 | 检查日志，错误被记录但被吞 |
| Excel 打开后表头变 `#NAME?` | 公式兼容性 | 把 `SUM(...)` 改为值聚合（v1.1） |
| 邮件主题乱码 | 主题含中文未编码 | 当前已 B 编码文件名；如有问题查 RFC 2047 |
| `RACE` 数据竞争 | 共享变量未加锁 | `go test -race` 定位 |

## 8. 提交与发布

### 8.1 Commit 规范

```
<type>(<scope>): <subject>

<body>

<footer>
```

- `type`: `feat` / `fix` / `docs` / `refactor` / `test` / `chore` / `perf` / `build` / `ci`
- `scope` 可选，如 `feat(db): add sqlserver driver`
- subject 不超过 72 字符
- body 解释 *what* 与 *why*

### 8.2 PR 流程

1. fork → 创建分支 → 提交
2. 推送后用 `gh pr create` 开 PR
3. CI 必跑：`go test ./...` + `go vet ./...` + `gofmt -l .` 必须空输出
4. 至少 1 位维护者 review

### 8.3 发布

```bash
make release VERSION=v1.1.0
```

`make release` 会：
1. 清理
2. 跑 lint + test
3. 跨平台构建（linux/windows）
4. 构建 Docker 镜像
5. 打印产物清单

发布完手动 push tag：

```bash
git push origin v1.1.0
```

## 9. 参考

- [Effective Go](https://go.dev/doc/effective_go)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [go-sqlmock](https://github.com/DATA-DOG/go-sqlmock) （未来可选，用于 sql.DB 级别的 mock）

## 10. 获取帮助

- 提 [Issue](https://github.com/peterydd/report/issues)
- 参与 [Discussions](https://github.com/peterydd/report/discussions)
- 安全问题见 [SECURITY.md](../SECURITY.md)
