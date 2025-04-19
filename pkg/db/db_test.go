package db

import (
	"testing"
)

func execute(factory DBFactory, dataSourceName string) error {
	db := factory.Create()
	if err := db.Connect(dataSourceName); err != nil {
		return err
	}
	defer db.Close()

	if err := db.Execute("DROP TABLE IF EXISTS test"); err != nil {
		return err
	}

	if err := db.Execute("CREATE TABLE IF NOT EXISTS test (id int) engine=Memory"); err != nil {
		return err
	}

	if err := db.Execute("INSERT INTO test (id) VALUES (1)"); err != nil {
		return err
	}

	if _, err := db.Query("SELECT id FROM test"); err != nil {
		return err
	}

	return nil
}

func TestDB(t *testing.T) {
	var (
		factory DBFactory
	)

	//factory = MysqlDBFactory{}
	//if err := execute(factory, "test:password@tcp(localhost:3306)/test"); err != nil {
	//	t.Fatal("error with factory method pattern:", err)
	//}
	//
	//factory = OracleDBFactory{}
	//if err := execute(factory, "oracle://test:password@127.0.0.1:1521/FREEPDB1"); err != nil {
	//	t.Fatal("error with factory method pattern", err)
	//}
	//
	//factory = PostgreSQLDBFactory{}
	//if err := execute(factory, "postgres://test:password@127.0.0.1:5432/test?sslmode=disable"); err != nil {
	//	t.Fatal("error with factory method pattern", err)
	//}

	factory = ClickHouseDBFactory{}
	if err := execute(factory, "clickhouse://test:password@127.0.0.1:9000/test?dial_timeout=200ms&max_execution_time=60"); err != nil {
		t.Fatal("error with factory method pattern", err)
	}
}

//func TestDBError(t *testing.T) {
//	//var (
//	//	factory DBFactory
//	//)
//
//	//factory = MysqlDBFactory{}
//	//if err := execute(factory, "test1:password@tcp(localhost:3306)/test"); err == nil {
//	//	t.Fatal("error with factory method pattern:", err)
//	//}
//	//
//	//factory = OracleDBFactory{}
//	//if err := execute(factory, "oracle://test1:password@127.0.0.1:1521/FREEPDB1"); err == nil {
//	//	t.Fatal("error with factory method pattern", err)
//	//}
//	//
//	//factory = PostgreSQLDBFactory{}
//	//if err := execute(factory, "postgres://test1:password@127.0.0.1:5432/test?sslmode=disable"); err == nil {
//	//	t.Fatal("error with factory method pattern", err)
//	//}
//
//	//factory = ClickHouseDBFactory{}
//	//if err := execute(factory, "clickhouse://test1:password@127.0.0.1:9000/test?dial_timeout=200ms&max_execution_time=60"); err != nil {
//	//	t.Fatal("error with factory method pattern", err)
//	//}
//}
