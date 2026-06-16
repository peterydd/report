# pkg/config

<p align="center">
  <a href="README.md">中文</a> | <a href="README.en.md">English</a>
</p>

YAML configuration management built on [viper](https://github.com/spf13/viper): loading, validation, hot-reload.

> Project home: [README](../../README.md) · Full reference: [docs/configuration.md](../../docs/configuration.md)

## Features

- ✅ Multi-path search: `/` → `./` → `./configs` → `../configs` → `../../configs`
- ✅ Strict validation: `Config.Validate()` covers all required fields
- ✅ Hot-reload: edits to `config.yaml` take effect immediately
- ✅ Env-var interpolation: `password: ${SMTP_PASSWORD}`
- ✅ Thread-safe via internal `sync.RWMutex`

## Install

```bash
go get github.com/peterydd/report/pkg/config
```

## Quick Start

```go
import (
    "log"
    "github.com/peterydd/report/pkg/config"
)

cfg, err := config.NewConfig()
if err != nil {
    log.Fatalf("load config: %v", err)
}

log.Printf("driver=%s host=%s", cfg.Database.Driver, cfg.Smtp.Host)
for _, r := range cfg.Reports {
    log.Printf("report %s with %d sheets", r.Name, len(r.Sheets))
}
```

## Exported API

| Symbol | Description |
|--------|-------------|
| `NewConfig() (*Config, error)` | Load + validate; start viper hot-reload watcher |
| `(*Config).Validate() error` | Manual validation (called by `NewConfig`) |
| `(*Config).Clone() *Config` | Read-only copy (thread-safe) |
| `(*Config).SetOnChange(func(*Config))` | Register hot-reload callback |
| `(*Config).NotifyChange()` | Trigger callback (internal) |
| `Database / Smtp / Reports` | See below |

### Type Sketch

```go
type Config struct {
    Database *Database
    Smtp     *Smtp
    Reports  []*Reports
    // mu + onChange are unexported
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
```

> See [docs/configuration.md](../../docs/configuration.md) for the full field reference.

## Config Example

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

## Hot Reload

```go
cfg, _ := config.NewConfig()

cfg.SetOnChange(func(c *config.Config) {
    log.Printf("config reloaded: %d reports", len(c.Reports))
    // reconnect clients, refresh caches, etc.
})
```

## Validation Errors

`Validate()` returns the first error encountered:

```
configuration validation failed: reports[0].sheets cannot be empty
```

## Testing

```bash
go test ./pkg/config
```

No real config required: `TestNewConfig` skips when `config.yaml` is missing; `TestConfigValidate` covers 8 scenarios.

## Related

- [`pkg/db`](../db) — database access
- [`pkg/excel`](../excel) — Excel generation
- [`pkg/mail`](../mail) — email delivery
- [`docs/architecture.md`](../../docs/architecture.md) — overall design
- [`docs/configuration.md`](../../docs/configuration.md) — field reference
