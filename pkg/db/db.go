/*
Package db - Multi-database support with connection pooling and streaming queries.
包 db - 支持多种数据库的连接池管理和流式查询。

This package provides unified database access for:
- MySQL
- PostgreSQL
- Oracle
- ClickHouse

Features / 功能特性:
- Factory pattern for database type selection
- Connection pool management
- Streaming query support for large datasets
- Batch processing capabilities
- Multiple database drivers support

Example / 示例:

	dbType := db.ParseDBType("mysql")
	database := db.NewDB(dbType)
	err := database.Connect(dsn, poolConfig)
*/
package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/sijms/go-ora/v2"
)

// ConnPoolConfig defines connection pool settings.
type ConnPoolConfig struct {
	MaxOpenConns    int           // Maximum open connections / 最大打开连接数
	MaxIdleConns    int           // Maximum idle connections / 最大空闲连接数
	ConnMaxLifetime time.Duration // Connection max lifetime / 连接最大生命周期
}

// RowHandler is a callback function type for streaming query row processing.
// 流式查询行处理回调函数类型。
type RowHandler func(row []interface{}) error

// DB interface defines the contract for database operations.
type DB interface {
	// Connect establishes a connection to the database.
	Connect(dataSourceName string, poolConfig *ConnPoolConfig) error
	// Query executes a SELECT query and returns all results.
	Query(query string, args ...interface{}) ([][]interface{}, error)
	// QueryStream executes a query and processes rows one at a time.
	QueryStream(query string, handler RowHandler, batchSize int) error
	// Execute executes a non-SELECT query (INSERT, UPDATE, DELETE).
	Execute(query string, args ...interface{}) error
	// Close closes the database connection.
	Close() error
}

// DBFactory interface defines the contract for creating database instances.
type DBFactory interface {
	// Create creates a new DB instance.
	Create() DB
}

// DBBase provides common database functionality for all DB implementations.
type DBBase struct {
	conn *sql.DB
}

// DBType enumerates supported database types.
type DBType int

const (
	ORACLE      DBType = iota // Oracle database / Oracle数据库
	MYSQL                     // MySQL database / MySQL数据库
	POSTGRESSQL               // PostgreSQL database / PostgreSQL数据库
	CLICKHOUSE                // ClickHouse database / ClickHouse数据库
)

// ParseDBType converts a driver name string to DBType.
// Supports both string ("mysql", "oracle", etc.) and numeric ("0", "1", etc.) formats.
func ParseDBType(driver string) DBType {
	switch driver {
	case "oracle", "0":
		return ORACLE
	case "mysql", "1":
		return MYSQL
	case "postgresql", "postgres", "2":
		return POSTGRESSQL
	case "clickhouse", "3":
		return CLICKHOUSE
	default:
		return -1
	}
}

// NewDBFactory creates a factory for the specified database type.
func NewDBFactory(t DBType) DBFactory {
	switch t {
	case ORACLE:
		return &OracleDBFactory{}
	case MYSQL:
		return &MysqlDBFactory{}
	case POSTGRESSQL:
		return &PostgreSQLDBFactory{}
	case CLICKHOUSE:
		return &ClickHouseDBFactory{}
	default:
		return nil
	}
}

// NewDB creates a new DB instance for the specified database type.
func NewDB(t DBType) DB {
	factory := NewDBFactory(t)
	if factory != nil {
		return factory.Create()
	}
	return nil
}

// Query executes a SELECT query and returns all results as a slice of rows.
// Each row is a slice of interface{} values.
func (d *DBBase) Query(query string, args ...interface{}) (results [][]interface{}, err error) {
	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column information: %w", err)
	}

	columnPointers := make([]interface{}, len(columns))
	results = make([][]interface{}, 0)

	for rows.Next() {
		line := make([]interface{}, len(columns))
		for i := range line {
			columnPointers[i] = &line[i]
		}

		err = rows.Scan(columnPointers...)
		if err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}

		results = append(results, line)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating result set: %w", err)
	}

	return results, nil
}

// QueryStream executes a query and processes rows incrementally using a callback handler.
// This is suitable for large datasets where loading all data into memory is impractical.
func (d *DBBase) QueryStream(query string, handler RowHandler, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 10000
	}

	rows, err := d.conn.Query(query)
	if err != nil {
		return fmt.Errorf("streaming query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get column information: %w", err)
	}

	columnPointers := make([]interface{}, len(columns))
	rowCount := 0
	for rows.Next() {
		line := make([]interface{}, len(columns))
		for i := range line {
			columnPointers[i] = &line[i]
		}

		err = rows.Scan(columnPointers...)
		if err != nil {
			return fmt.Errorf("row scan failed: %w", err)
		}

		if err := handler(line); err != nil {
			return fmt.Errorf("row handler error: %w", err)
		}

		rowCount++
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating result set: %w", err)
	}

	return nil
}

// Execute executes a non-SELECT query (INSERT, UPDATE, DELETE, etc.).
func (d *DBBase) Execute(query string, args ...interface{}) error {
	_, err := d.conn.Exec(query, args...)
	return err
}

// Close closes the database connection.
func (d *DBBase) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// MysqlDB implements DB interface for MySQL database.
type MysqlDB struct {
	*DBBase
}

// MysqlDBFactory creates MySQL database instances.
type MysqlDBFactory struct{}

func (MysqlDBFactory) Create() DB {
	return &MysqlDB{
		DBBase: &DBBase{},
	}
}

// Connect establishes connection to MySQL database.
func (m *MysqlDB) Connect(dataSourceName string, poolConfig *ConnPoolConfig) error {
	var err error
	m.conn, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		return err
	}
	if err := m.conn.Ping(); err != nil {
		return err
	}
	if poolConfig != nil {
		m.conn.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
		m.conn.SetMaxOpenConns(poolConfig.MaxOpenConns)
		m.conn.SetMaxIdleConns(poolConfig.MaxIdleConns)
	}
	return nil
}

// OracleDB implements DB interface for Oracle database.
type OracleDB struct {
	*DBBase
}

// OracleDBFactory creates Oracle database instances.
type OracleDBFactory struct{}

func (OracleDBFactory) Create() DB {
	return &OracleDB{
		DBBase: &DBBase{},
	}
}

// Connect establishes connection to Oracle database.
func (o *OracleDB) Connect(dataSourceName string, poolConfig *ConnPoolConfig) error {
	var err error
	o.conn, err = sql.Open("oracle", dataSourceName)
	if err != nil {
		return err
	}
	if err := o.conn.Ping(); err != nil {
		return err
	}
	if poolConfig != nil {
		o.conn.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
		o.conn.SetMaxOpenConns(poolConfig.MaxOpenConns)
		o.conn.SetMaxIdleConns(poolConfig.MaxIdleConns)
	}
	return nil
}

// PostgreSQLDB implements DB interface for PostgreSQL database.
type PostgreSQLDB struct {
	*DBBase
}

// PostgreSQLDBFactory creates PostgreSQL database instances.
type PostgreSQLDBFactory struct{}

func (PostgreSQLDBFactory) Create() DB {
	return &PostgreSQLDB{
		DBBase: &DBBase{},
	}
}

// Connect establishes connection to PostgreSQL database.
func (p *PostgreSQLDB) Connect(dataSourceName string, poolConfig *ConnPoolConfig) error {
	var err error
	p.conn, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		return err
	}
	if err := p.conn.Ping(); err != nil {
		return err
	}
	if poolConfig != nil {
		p.conn.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
		p.conn.SetMaxOpenConns(poolConfig.MaxOpenConns)
		p.conn.SetMaxIdleConns(poolConfig.MaxIdleConns)
	}
	return nil
}

// ClickHouseDB implements DB interface for ClickHouse database.
type ClickHouseDB struct {
	*DBBase
}

// ClickHouseDBFactory creates ClickHouse database instances.
type ClickHouseDBFactory struct{}

func (ClickHouseDBFactory) Create() DB {
	return &ClickHouseDB{
		DBBase: &DBBase{},
	}
}

// Connect establishes connection to ClickHouse database.
func (c *ClickHouseDB) Connect(dataSourceName string, poolConfig *ConnPoolConfig) error {
	var err error
	c.conn, err = sql.Open("clickhouse", dataSourceName)
	if err != nil {
		return err
	}
	if err := c.conn.Ping(); err != nil {
		return err
	}
	if poolConfig != nil {
		c.conn.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
		c.conn.SetMaxOpenConns(poolConfig.MaxOpenConns)
		c.conn.SetMaxIdleConns(poolConfig.MaxIdleConns)
	}
	return nil
}
