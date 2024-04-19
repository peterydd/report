package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/sijms/go-ora/v2"
	"log"
	"time"
)

// 工厂方法模式
// DB 是被封装的实际类接口
type DB interface {
	Connect(dataSourceName string) error
	Query(query string, args ...interface{}) ([][]interface{}, error)
	Execute(query string, args ...interface{}) error
	Close() error
}

// DBFactory 是工厂接口
type DBFactory interface {
	Create() DB
}

// DBBase 是DB 接口实现的基类，封装公用方法
type DBBase struct {
	conn *sql.DB
}

// DBType 是工厂类型
type DBType int

const (
	// ORACLE 是 Oracle 类型
	ORACLE DBType = iota
	// MYSQL 是 Mysql 类型
	MYSQL
	// POSTGRESSQL 是 Postgressql 类型
	POSTGRESSQL
)

// NewDBFactory 是工厂方法
func NewDBFactory(t DBType) DBFactory {
	switch t {
	case ORACLE:
		return &OracleDBFactory{}
	case MYSQL:
		return &MysqlDBFactory{}
	case POSTGRESSQL:
		return &PostgreSQLDBFactory{}
	default:
		return nil
	}
}

// NewDB 是工厂方法
func NewDB(t DBType) DB {
	factory := NewDBFactory(t)
	if factory != nil {
		return factory.Create()
	}
	return nil
}

// Query
func (d *DBBase) Query(query string, args ...interface{}) (results [][]interface{}, err error) {
	rows, err := d.conn.Query(query, args...)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		line := make([]interface{}, len(columns))
		columnPointers := make([]interface{}, len(columns))
		for i := range line {
			columnPointers[i] = &line[i]
		}

		err = rows.Scan(columnPointers...)
		if err != nil {
			return nil, err
		}

		results = append(results, line)
	}

	// 检查扫描过程中是否有错误
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	return results, nil
}

// Execute
func (d *DBBase) Execute(query string, args ...interface{}) error {
	_, err := d.conn.Exec(query, args...)
	return err
}

// Close 关闭数据库连接
func (d *DBBase) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// MysqlDB
type MysqlDB struct {
	*DBBase
}

// MysqlDBFactory 是 MysqlDB 的工厂类
type MysqlDBFactory struct{}

func (MysqlDBFactory) Create() DB {
	return &MysqlDB{
		DBBase: &DBBase{},
	}
}

// MysqlDB Connect方法具体实现
func (m *MysqlDB) Connect(dataSourceName string) error {
	var err error
	m.conn, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		return err
	}
	if err := m.conn.Ping(); err != nil {
		return err
	}
	// 连接池配置示例
	m.conn.SetConnMaxLifetime(time.Minute * 3)
	m.conn.SetMaxOpenConns(25)
	m.conn.SetMaxIdleConns(5)
	return nil
}

// OracleDB
type OracleDB struct {
	*DBBase
}

// OracleDBFactory 是 OracleDB 的工厂类
type OracleDBFactory struct{}

func (OracleDBFactory) Create() DB {
	return &OracleDB{
		DBBase: &DBBase{},
	}
}

// OracleDB Connect方法具体实现
func (o *OracleDB) Connect(dataSourceName string) error {
	var err error
	o.conn, err = sql.Open("oracle", dataSourceName)
	if err != nil {
		return err
	}
	if err := o.conn.Ping(); err != nil {
		return err
	}
	// 连接池配置示例
	o.conn.SetConnMaxLifetime(time.Minute * 3)
	o.conn.SetMaxOpenConns(25)
	o.conn.SetMaxIdleConns(5)
	return nil
}

// PostgreSQLDB
type PostgreSQLDB struct {
	*DBBase
}

// PostgreSQLDBFactory 是 PostgreSQLDB 的工厂类
type PostgreSQLDBFactory struct{}

func (PostgreSQLDBFactory) Create() DB {
	return &PostgreSQLDB{
		DBBase: &DBBase{},
	}
}

// PostgreSQLDB Connect方法具体实现
func (p *PostgreSQLDB) Connect(dataSourceName string) error {
	var err error
	p.conn, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		return err
	}
	if err := p.conn.Ping(); err != nil {
		return err
	}
	// 连接池配置示例
	p.conn.SetConnMaxLifetime(time.Minute * 3)
	p.conn.SetMaxOpenConns(25)
	p.conn.SetMaxIdleConns(5)
	return nil
}
