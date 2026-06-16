# Report

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.4-00ADD8?style=for-the-badge&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker" alt="Docker Ready">
  <img src="https://img.shields.io/badge/Kubernetes-Supported-326CE5?style=for-the-badge&logo=kubernetes" alt="Kubernetes">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License">
</p>

<p align="center">一个高性能的数据库报表生成与邮件发送系统，支持百万行流式处理。</p>

<p align="center">
  <a href="README.en.md">English</a> ·
  <a href="docs/INDEX.md">文档中心</a> ·
  <a href="ROADMAP.md">路线图</a> ·
  <a href="CHANGELOG.md">更新日志</a>
</p>

## 目录

- [功能特性](#功能特性)
- [快速开始](#快速开始)
- [命令行参数](#命令行参数)
- [环境变量](#环境变量)
- [配置说明](#配置说明)
- [架构概览](#架构概览)
- [性能与调优](#性能与调优)
- [部署指南](#部署指南)
- [测试](#测试)
- [常见问题](#常见问题)
- [贡献](#贡献)
- [许可证](#许可证)

## 功能特性

- **多数据库支持** — MySQL、PostgreSQL、Oracle、ClickHouse 统一接口
- **流式查询** — 百万级数据量低内存占用（基于 `database/sql` 的 `rows.Next()` 回调）
- **并发 Sheet** — 单报表内多 Sheet 并行查询，受信号量限制保护连接池
- **邮件发送** — SMTP + STARTTLS 加密、附件 base64 编码、抄送/密送、超时控制
- **配置灵活** — YAML 配置，强校验，支持 viper 热重载（运行时改文件即可生效）
- **Mock 测试** — 内置 `db.MockDB`、`mail.MockMail`，纯单元测试无需数据库
- **容器化** — 多阶段 Dockerfile，体积小，支持 K8s CronJob 调度
- **可观测** — 关键流程含 `log.Printf` 日志，文件输出可对接采集器

## 快速开始

### 前置要求

- Go 1.26.4 （见 `go.mod`）
- 一个可达的数据库（MySQL / PostgreSQL / Oracle / ClickHouse 任一）
- 一个可达的 SMTP 服务

### 1. 拉取与构建

```bash
git clone https://github.com/peterydd/report.git
cd report
go mod tidy
make build         # Linux/macOS
# Windows: go build -o report.exe ./cmd/report
```

### 2. 准备配置

```bash
cp configs/config.yaml.example config.yaml
# 编辑 config.yaml，填入真实的数据库与 SMTP 信息
```

### 3. 运行

```bash
./report           # Linux/macOS
# Windows: .\report.exe
```

## 命令行参数

| 参数 | 说明 |
|------|------|
| `-version` | 打印版本号与构建时间（注入自 `main.version` / `main.buildTime`） |
| `-help`    | 打印帮助信息 |

不带任何参数时进入正常流程：加载配置 → 连接数据库 → 执行报表 → 发送邮件。

> 编译时通过 `LDFLAGS` 注入版本：
> `go build -ldflags "-X main.version=v1.1.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./cmd/report`

## 环境变量

| 变量 | 作用 | 默认 |
|------|------|------|
| `REPORT_INTEGRATION` | 设置为非空时启用集成测试（真实数据库 / SMTP） | 未设置 → 跳过集成测试 |
| `CONFIG_PATH`        | 自定义配置文件目录（通过 viper 路径列表） | `./`、`./configs/`、`/`

集成测试示例：

```bash
REPORT_INTEGRATION=1 go test ./...
```

## 配置说明

`config.yaml` 必须位于以下任一路径（按顺序查找）：

```
/             (容器内挂载点)
./
./configs
../configs
../../configs
```

最小配置示例（完整示例见 `configs/config.yaml.example`）：

```yaml
database:
  driver: "mysql"                              # mysql | postgresql | oracle | clickhouse
  source: "user:pass@tcp(localhost:3306)/db"
  maxOpenConns: 25
  maxIdleConns: 5
  connMaxLifetime: "3m"

smtp:
  host: "smtp.example.com"
  port: "587"
  username: "sender@example.com"
  password: "your-password"
  insecureSkipVerify: false
  timeout: "30s"

reports:
  - name: "daily_sales"
    workBook:
      prefix: "sales_"
      dateFormat: "20060102"
      suffix: ".xlsx"
    sheets:
      - name: "sales"
        sql: "SELECT date, product, amount FROM sales"
        column: "Date,Product,Amount"
        isSum: true
        sumBeginColumn: 3
        enableStream: true      # 大数据量时启用
        batchSize: 50000
    message:
      from: "sender@example.com"
      to: ["manager@example.com"]
      cc: ["team@example.com"]
      subject: "每日销售报表"
      body: "请查收附件"
      contentType: "text/plain;charset=utf-8"
      attachment:
        contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        withFile: true
```

完整配置参考见 [docs/configuration.md](docs/configuration.md)。

## 架构概览

```
                +-----------------+
                │   cmd/report    │   入口：解析 flag → 调用 app.Run()
                +--------+--------+
                         │
                +--------v--------+
                │  internal/app   │   编排：连接池 / sheet 并发 / 邮件发送
                +--------+--------+
                         │
   +---------+-----------+-----------+---------+
   │         │           │           │         │
+--v---+ +---v---+   +---v---+   +---v---+ +---v---+
│ db   │ | excel |   │ mail  │   │ cfg   │ (future)
| pkg  │ |  pkg  |   |  pkg  |   |  pkg  |
+------+ +-------+   +-------+   +-------+
   4 驱动    excelize    smtp         viper
   Mock      冻结+SUM   附件+TLS    热重载
```

详细分层与并发模型见 [docs/architecture.md](docs/architecture.md)。

## 性能与调优

### 大数据量

| 数据量 | 模式 | 内存参考 |
|--------|------|----------|
| < 10 万行  | 普通 | ~100MB |
| 10-100 万  | 流式（`enableStream: true`）| ~100MB |
| > 100 万   | 流式 + 较大 `batchSize` | ~200MB |

### 连接池建议

- 单报表多 sheet：`maxOpenConns >= sheet 数`，但受内部信号量 `maxConcurrentSheets=8` 约束
- 高频调度：增大 `maxIdleConns` 减少握手
- 跨地域数据库：缩短 `connMaxLifetime` 主动重建

### 邮件性能

- 单次发送受 `timeout` 约束；并发发送需要将 `Run()` 拆分为 goroutine 池（v1.1 计划）
- 附件大小受 SMTP 协议影响，超过 20MB 建议改用对象存储 + 链接

## 部署指南

### Docker

```bash
make docker-build
docker run -d \
  -v $(pwd)/config.yaml:/config.yaml \
  --name report peterydd/report:latest
```

### Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-report
spec:
  schedule: "0 9 * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: report
              image: peterydd/report:latest
              volumeMounts:
                - name: cfg
                  mountPath: /config.yaml
                  subPath: config.yaml
              resources:
                requests: { memory: "256Mi", cpu: "250m" }
                limits:   { memory: "2Gi",   cpu: "1000m" }
          restartPolicy: OnFailure
      volumes:
        - name: cfg
          configMap: { name: report-config }
```

完整运维指南见 [docs/operations.md](docs/operations.md)。

### 生产部署

- **Docker Compose**（单机 / 边缘节点）：`deploy/docker/docker-compose.yaml` + `Dockerfile.production`
- **Kubernetes CronJob**（集群 / 多节点）：`deploy/k8s/`（含 ConfigMap / Secret / PVC / CronJob）

详细使用、安全检查清单与故障排查见 [deploy/README.md](deploy/README.md)。

## 测试

```bash
# 单元测试（无需数据库 / SMTP）
make test

# 集成测试（需要真实环境）
REPORT_INTEGRATION=1 go test ./...

# 覆盖率
make test-coverage

# 基准
make benchmark
```

Mock 用法：

```go
db := db.NewMockDB()
db.SetQueryResult("SELECT 1", [][]interface{}{ {"x"} })
results, _ := db.Query("SELECT 1")

mail := mail.NewMockMail()
mail.SetError("smtp down")
err := mail.Send(msg)  // 返回错误
```

## 常见问题

### Q: 邮件中文乱码？
A: `contentType` 使用 `text/plain;charset=utf-8` 或 `text/html;charset=utf-8`；附件文件名经过 MIME B 编码。

### Q: 大数据量 OOM？
A: 开启 `enableStream: true`，调整 `batchSize`（默认 10000，10-50k 较常用）。

### Q: 配置改了没生效？
A: 工具内置 viper 热重载，保存 `config.yaml` 即生效；下一个 sheet/报表读取时使用新配置（不会中断正在运行的报表）。

### Q: 如何添加新数据库？
A: 见 [docs/development.md#添加新的数据库驱动](docs/development.md)。

### Q: 数据库慢但报表没失败？
A: 单 sheet 超时需要 SQL 端优化；当前实现复用同一连接池，并发受 `maxConcurrentSheets=8` 限制。

## 贡献

欢迎 Issue 和 PR。详细规范见 [CONTRIBUTING.md](CONTRIBUTING.md)。安全相关问题请按 [SECURITY.md](SECURITY.md) 处理。

## 许可证

[MIT](LICENSE)

## 联系方式

- GitHub: <https://github.com/peterydd/report>
- Issues: <https://github.com/peterydd/report/issues>
- Discussions: <https://github.com/peterydd/report/discussions>

---

<p align="center">如果这个项目对您有帮助，请给个 ⭐️ Star！</p>
