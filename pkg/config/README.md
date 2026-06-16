# pkg/config

<p align="center">
  <a href="README.md">中文</a> | <a href="README.en.md">English</a>
</p>

YAML 配置管理包，基于 [viper](https://github.com/spf13/viper) 实现加载、校验、热重载。

> 项目主页：[README](../../README.md) · 完整配置参考：[docs/configuration.md](../../docs/configuration.md)

## 特性

- ✅ 多路径搜索 `/` → `./` → `./configs` → `../configs` → `../../configs`
- ✅ 强校验：`Config.Validate()` 覆盖所有必填字段
- ✅ 热重载：保存 `config.yaml` 即生效
- ✅ 环境变量插值：`password: ${SMTP_PASSWORD}`
- ✅ 线程安全：内部 `sync.RWMutex` 保护

## 安装

```bash
go get github.com/peterydd/report/pkg/config
```

## 快速上手

```go
import (
    "log"
    "github.com/peterydd/report/pkg/config"
)

cfg, err := config.NewConfig()
if err != nil {
    log.Fatalf("load config: %v", err)
}

// 访问配置
log.Printf("driver=%s host=%s", cfg.Database.Driver, cfg.Smtp.Host)
for _, r := range cfg.Reports {
    log.Printf("report %s with %d sheets", r.Name, len(r.Sheets))
}
```

## 导出 API

| 符号 | 说明 |
|------|------|
| `NewConfig() (*Config, error)` | 加载并校验配置；启动 viper 热重载监听 |
| `(*Config).Validate() error` | 手动校验（一般 NewConfig 已调用） |
| `(*Config).Clone() *Config` | 获取只读副本（线程安全） |
| `(*Config).SetOnChange(func(*Config))` | 注册热重载回调 |
| `(*Config).NotifyChange()` | 触发回调（包内部使用） |
| `Database / Smtp / Reports` | 配置结构体（见下） |

### 类型概览

```go
type Config struct {
    Database *Database
    Smtp     *Smtp
    Reports  []*Reports
    // mu + onChange 内部使用
}

type Database struct {
    Driver          string  // mysql | postgresql | oracle | clickhouse
    Source          string  // DSN
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime string  // duration, e.g. "3m"
}

type Smtp struct {
    Host               string
    Port               string
    Username           string
    Password           string
    InsecureSkipVerify bool
    Timeout            string  // duration
}

type Reports struct {
    Name     string
    WorkBook *WorkBook
    Sheets   []*Sheet
    Message  *Message
}
```

> 完整字段说明见 [docs/configuration.md](../../docs/configuration.md)。

## 配置示例

```yaml
database:
  driver: mysql
  source: "user:pass@tcp(127.0.0.1:3306)/db"
  maxOpenConns: 25
  maxIdleConns: 5
  connMaxLifetime: 3m

smtp:
  host: smtp.example.com
  port: "587"
  username: report@example.com
  password: ${SMTP_PASSWORD}
  insecureSkipVerify: false
  timeout: 30s

reports:
  - name: daily_sales
    workBook: { prefix: "sales_", dateFormat: "20060102", suffix: ".xlsx" }
    sheets:
      - name: orders
        sql: "SELECT * FROM orders"
        column: "ID,Date,Amount"
        isSum: true
        sumBeginColumn: 3
    message:
      from: report@example.com
      to: ["manager@example.com"]
      subject: "Daily Sales"
```

## 热重载示例

```go
cfg, _ := config.NewConfig()

cfg.SetOnChange(func(c *config.Config) {
    log.Printf("config reloaded: %d reports", len(c.Reports))
    // 触发重连、刷新客户端等
})
```

## 校验失败信息

`Validate()` 返回的错误形如：

```
configuration validation failed: reports[0].sheets cannot be empty
```

第一个错误即返回，便于定位。

## 测试

```bash
go test ./pkg/config
```

不依赖真实配置文件：`TestNewConfig` 在 `config.yaml` 缺失时跳过；`TestConfigValidate` 覆盖 8 种典型场景。

## 关联

- [`pkg/db`](../db) — 数据库访问
- [`pkg/excel`](../excel) — Excel 生成
- [`pkg/mail`](../mail) — 邮件发送
- [`docs/architecture.md`](../../docs/architecture.md) — 整体架构
- [`docs/configuration.md`](../../docs/configuration.md) — 字段详解
