# pkg/db

<p align="center">
  <a href="README.md">中文</a> | <a href="README.en.md">English</a>
</p>

Unified multi-database access layer with factory pattern. Supports MySQL, PostgreSQL, Oracle, and ClickHouse.

> Project home: [README](../../README.md) · Architecture: [docs/architecture.md](../../docs/architecture.md)

## Features

- ✅ 4 databases behind one `DB` interface
- ✅ Connection pool: `MaxOpenConns` / `MaxIdleConns` / `ConnMaxLifetime`
- ✅ Streaming: `QueryStream` row-level callback, low memory at any scale
- ✅ Mock: `MockDB` implements the full interface for dependency-free tests
- ✅ Driver parsing: `ParseDBType("mysql" | "1" | ...)` accepts both string and numeric forms

## Install

```bash
go get github.com/peterydd/report/pkg/db
```

## Quick Start

```go
import (
    "log"
    "time"
    "github.com/peterydd/report/pkg/db"
)

database := db.NewDB(db.ParseDBType("mysql"))
if err := database.Connect(
    "user:pass@tcp(127.0.0.1:3306)/dbname",
    &db.ConnPoolConfig{
        MaxOpenConns:    25,
        MaxIdleConns:    5,
        ConnMaxLifetime: 3 * time.Minute,
    },
); err != nil {
    log.Fatal(err)
}
defer database.Close()

// One-shot query
rows, err := database.Query("SELECT id, name FROM users WHERE active = ?", true)

// Streaming query
err = database.QueryStream("SELECT * FROM big_table", func(row []interface{}) error {
    log.Printf("row: %v", row)
    return nil
}, 10000)
```

## Supported Drivers

| `DBType` | String | Num | Driver | Typical DSN |
|----------|--------|-----|--------|-------------|
| `ORACLE` | `oracle` | `0` | `github.com/sijms/go-ora/v2` | `oracle://user:pass@host:1521/ServiceName` |
| `MYSQL` | `mysql` | `1` | `github.com/go-sql-driver/mysql` | `user:pass@tcp(host:3306)/db?charset=utf8mb4` |
| `POSTGRESSQL` | `postgresql` / `postgres` | `2` | `github.com/lib/pq` | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `CLICKHOUSE` | `clickhouse` | `3` | `github.com/ClickHouse/clickhouse-go/v2` | `clickhouse://user:pass@host:9000/db?dial_timeout=10s` |

> Numeric forms are kept for backward compatibility; new deployments should use strings.

## Exported API

### Interface

```go
type DB interface {
    Connect(dataSourceName string, poolConfig *ConnPoolConfig) error
    Query(query string, args ...interface{}) ([][]interface{}, error)
    QueryStream(query string, handler RowHandler, batchSize int) error
    Execute(query string, args ...interface{}) error
    Close() error
}
```

### Common Types

```go
type ConnPoolConfig struct {
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

type RowHandler func(row []interface{}) error
```

### Factory

```go
func ParseDBType(driver string) DBType
func NewDB(t DBType) DB
func NewDBFactory(t DBType) DBFactory
```

## Streaming vs One-shot

| Aspect | `Query` | `QueryStream` |
|--------|---------|----------------|
| Return | `[][]interface{}` | callback, no return |
| Memory | all rows in RAM | row-level |
| Suitable for | < 100K rows | any size |
| Error | `error` | handler return or scan error |

```go
err := db.QueryStream("SELECT * FROM big", func(row []interface{}) error {
    if invalid(row) { return errors.New("bad data") }  // aborts
    return process(row)
}, 50000)
```

## Mock Testing

```go
import "github.com/peterydd/report/pkg/db"

mock := db.NewMockDB()

// Pre-set one-shot results
mock.SetQueryResult("SELECT * FROM users", [][]interface{}{
    {1, "alice"}, {2, "bob"},
})
rows, _ := mock.Query("SELECT * FROM users")

// Pre-set streaming results
mock.SetStreamResult("SELECT * FROM events", [][]interface{}{
    {"e1"}, {"e2"},
})
mock.QueryStream("SELECT * FROM events", func(r []interface{}) error {
    fmt.Println(r); return nil
}, 10)

// Force an error
mock.SetError("connection lost")
_, err := mock.Query("SELECT 1")  // mock query error: connection lost

// Inspect call counts
count := mock.GetQueryCount("SELECT * FROM users")
```

## Testing

```bash
# Unit + mock tests
go test ./pkg/db

# Integration (real DB)
REPORT_INTEGRATION=1 go test -run TestDB ./pkg/db
```

## Related

- [`pkg/config`](../config) — parses `database.source`
- [`internal/app`](../../internal/app) — orchestration layer
- [Adding a new driver](../../docs/development.md#添加新的数据库驱动)
