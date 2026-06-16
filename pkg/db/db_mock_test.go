package db

import (
	"testing"
	"time"
)

func TestMockDBConnect(t *testing.T) {
	db := NewMockDB()

	poolConfig := &ConnPoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 3 * time.Minute,
	}

	// 测试正常连接
	err := db.Connect("test@tcp(localhost:3306)/test", poolConfig)
	if err != nil {
		t.Errorf("连接失败: %v", err)
	}

	if !db.IsConnected() {
		t.Error("应该已连接")
	}

	// 测试连接错误
	db.SetError("connection refused")
	err = db.Connect("test@tcp(localhost:3306)/test", poolConfig)
	if err == nil {
		t.Error("应该返回错误")
	}

	db.ClearError()
}

func TestMockDBQuery(t *testing.T) {
	db := NewMockDB()

	// 设置预期结果
	expectedResults := [][]interface{}{
		{"id1", "name1", 100},
		{"id2", "name2", 200},
		{"id3", "name3", 300},
	}

	db.SetQueryResult("SELECT * FROM test_table", expectedResults)

	// 执行查询
	results, err := db.Query("SELECT * FROM test_table")
	if err != nil {
		t.Errorf("查询失败: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("期望返回3行，实际返回%d行", len(results))
	}

	// 验证查询次数
	if db.GetQueryCount("SELECT * FROM test_table") != 1 {
		t.Error("查询应该被调用一次")
	}

	// 测试错误
	db.SetError("query failed")
	_, err = db.Query("SELECT * FROM error_table")
	if err == nil {
		t.Error("应该返回错误")
	}
}

func TestMockDBQueryStream(t *testing.T) {
	db := NewMockDB()

	// 设置流式查询结果
	streamData := [][]interface{}{
		{"row1_col1", "row1_col2"},
		{"row2_col1", "row2_col2"},
		{"row3_col1", "row3_col2"},
	}
	db.SetStreamResult("SELECT * FROM stream_table", streamData)

	// 收集结果
	var collected []interface{}
	rowCount := 0

	err := db.QueryStream("SELECT * FROM stream_table", func(row []interface{}) error {
		collected = append(collected, row)
		rowCount++
		return nil
	}, 100)

	if err != nil {
		t.Errorf("流式查询失败: %v", err)
	}

	if rowCount != 3 {
		t.Errorf("期望处理3行，实际处理%d行", rowCount)
	}

	// 验证流式查询次数
	if db.GetStreamCount("SELECT * FROM stream_table") != 1 {
		t.Error("流式查询应该被调用一次")
	}
}

func TestMockDBExecute(t *testing.T) {
	db := NewMockDB()

	// 测试正常执行
	err := db.Execute("INSERT INTO test VALUES (1)")
	if err != nil {
		t.Errorf("执行失败: %v", err)
	}

	// 测试错误
	db.SetError("execution failed")
	err = db.Execute("INSERT INTO test VALUES (1)")
	if err == nil {
		t.Error("应该返回错误")
	}
}

func TestMockDBClose(t *testing.T) {
	db := NewMockDB()

	poolConfig := &ConnPoolConfig{
		MaxOpenConns:    10,
		ConnMaxLifetime: 3 * time.Minute,
	}

	// 连接
	db.Connect("test@tcp(localhost:3306)/test", poolConfig)

	// 关闭
	err := db.Close()
	if err != nil {
		t.Errorf("关闭失败: %v", err)
	}

	if !db.IsClosed() {
		t.Error("应该已关闭")
	}

	if db.IsConnected() {
		t.Error("应该未连接")
	}
}

func TestMockDBMultipleQueries(t *testing.T) {
	db := NewMockDB()

	// 设置多个查询结果
	db.SetQueryResult("SELECT * FROM table1", [][]interface{}{
		{"t1_id1"},
	})
	db.SetQueryResult("SELECT * FROM table2", [][]interface{}{
		{"t2_id1"},
		{"t2_id2"},
	})

	// 执行多次查询
	db.Query("SELECT * FROM table1")
	db.Query("SELECT * FROM table1")
	db.Query("SELECT * FROM table2")

	// 验证调用次数
	if db.GetQueryCount("SELECT * FROM table1") != 2 {
		t.Errorf("期望查询table1两次，实际%d次", db.GetQueryCount("SELECT * FROM table1"))
	}

	if db.GetQueryCount("SELECT * FROM table2") != 1 {
		t.Errorf("期望查询table2一次，实际%d次", db.GetQueryCount("SELECT * FROM table2"))
	}
}
