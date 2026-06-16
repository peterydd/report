# pkg/db

<p align="center">
  <a href="README.md">中文</a> | <a href="README.en.md">English</a>
</p>

多数据库统一访问层，工厂模式 + 接口抽象，支持 MySQL / PostgreSQL / Oracle / ClickHouse。

> 项目主页：[README](../../README.md) · 架构：[docs/architecture.md](../../docs/architecture.md)

## 特性

- ✅ 4 种数据库统一 `DB` 接口
- ✅ 连接池：`MaxOpenConns` / `MaxIdleConns` / `ConnMaxLifetime`
- ✅ 流式查询：`QueryStream` 行级回调，百万行低内存
- ✅ Mock：`MockDB` 完整实现接口，零外部依赖单元测试
- ✅ 驱动识别：`ParseDBType("mysql" | "1" | ...)` 字符串和数字皆可

## 安装

```bash
go get github.com/peterydd/report/pkg/db
```

## 快速上手

```go
import (
    "log"
    "time"
    "github.com/peterydd/report/pkg/db"
)

database, err := db.NewDB(db.ParseDBType("mysql")), nil
_ = err
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

// 普通查询
rows, err := database.Query("SELECT id, name FROM users WHERE active = ?", true)

// 流式查询
err = database.QueryStream("SELECT * FROM big_table", func(row []interface{}) error {
    log.Printf("row: %v", row)
    return nil
}, 10000)
```

## 支持的驱动

| `DBType` | 字符串 | 数字 | 驱动包 | 典型 DSN |
|----------|--------|------|--------|----------|
| `ORACLE` | `oracle` | `0` | `github.com/sijms/go-ora/v2` | `oracle://user:pass@host:1521/ServiceName` |
| `MYSQL` | `mysql` | `1` | `github.com/go-sql-driver/mysql` | `user:pass@tcp(host:3306)/db?charset=utf8mb4` |
| `POSTGRESSQL` | `postgresql` / `postgres` | `2` | `github.com/lib/pq` | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `CLICKHOUSE` | `clickhouse` | `3` | `github.com/ClickHouse/clickhouse-go/v2` | `clickhouse://user:pass@host:9000/db?dial_timeout=10s` |

> 数字格式仅用于**配置兼容旧值**，新部署应使用字符串。

## 导出 API

### 接口

```go
type DB interface {
    Connect(dataSourceName string, poolConfig *ConnPoolConfig) error
    Query(query string, args ...interface{}) ([][]interface{}, error)
    QueryStream(query string, handler RowHandler, batchSize int) error
    Execute(query string, args ...interface{}) error
    Close() error
}
```

### 公共类型

```go
type ConnPoolConfig struct {
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

type RowHandler func(row []interface{}) error
```

### 工厂

```go
func ParseDBType(driver string) DBType      // 解析驱动字符串
func NewDB(t DBType) DB                       // 工厂
func NewDBFactory(t DBType) DBFactory         // 显式工厂
```

## 流式 vs 普通查询

| 维度 | `Query` | `QueryStream` |
|------|---------|----------------|
| 返回 | `[][]interface{}` 全量 | 回调，无返回值 |
| 内存 | 全部驻留 | 每行处理后释放 |
| 适用 | < 10 万行 | 任意规模 |
| 错误返回 | `error` | handler 返回或扫描错误 |

```go
// 大数据量：流式
err := db.QueryStream("SELECT * FROM big", func(row []interface{}) error {
    if invalid(row) {
        return errors.New("bad data")  // 中断流
    }
    return process(row)  // 返回 nil 继续
}, 50000)
```

## Mock 测试

```go
import "github.com/peterydd/report/pkg/db"

mock := db.NewMockDB()

// 预置普通查询结果
mock.SetQueryResult("SELECT * FROM users", [][]interface{}{
    {1, "alice"},
    {2, "bob"},
})
rows, _ := mock.Query("SELECT * FROM users")

// 预置流式结果
mock.SetStreamResult("SELECT * FROM events", [][]interface{}{
    {"e1"}, {"e2"},
})
mock.QueryStream("SELECT * FROM events", func(r []interface{}) error {
    fmt.Println(r); return nil
}, 10)

// 强制错误
mock.SetError("connection lost")
_, err := mock.Query("SELECT 1")
fmt.Println(err)  // mock query error: connection lost

// 检查调用次数
count := mock.GetQueryCount("SELECT * FROM users")
```

详见 `mock_db.go` 与 `db_mock_test.go`。

## 测试

```bash
# 单元 + Mock 测试
go test ./pkg/db

# 集成测试（需要真实数据库）
REPORT_INTEGRATION=1 go test -run TestDB ./pkg/db
```

## 关联

- [`pkg/config`](../config) — 解析 `database.source` 等字段
- [`internal/app`](../../internal/app) — 编排层使用本包
- [`docs/development.md` → 添加新驱动](../../docs/development.md#添加新的数据库驱动)
