/*
Package db - Mock database implementation for testing.
包 db - 用于测试的模拟数据库实现。

This file provides MockDB for unit testing without real database connections.
本文件提供 MockDB，用于在不建立真实数据库连接的情况下进行单元测试。
*/
package db

import (
	"fmt"
)

// MockDB implements the DB interface for testing purposes.
// It allows setting predefined query results and simulating errors.
// MockDB 实现 DB 接口，用于测试目的。
// 它允许设置预定义的查询结果并模拟错误。
type MockDB struct {
	queryResults   map[string][][]interface{}
	queryStreams   map[string][]func() ([]interface{}, bool)
	streamErrors   map[string]string // per-query stream errors / 按查询的流式错误
	queryErrors    map[string]string // per-query errors / 按查询的错误
	streamPanics   map[string]string // per-query panics / 按查询的 panic
	queryPanics    map[string]string // per-query panics / 按查询的 panic
	shouldError    bool              // global fallback error / 全局兜底错误
	errorMessage   string
	connected      bool
	closed         bool
	queryCalled    map[string]int
	streamCalled   map[string]int
}

// NewMockDB creates a new MockDB instance.
func NewMockDB() *MockDB {
	return &MockDB{
		queryResults: make(map[string][][]interface{}),
		queryStreams: make(map[string][]func() ([]interface{}, bool)),
		queryErrors:  make(map[string]string),
		streamErrors: make(map[string]string),
		queryPanics:  make(map[string]string),
		streamPanics: make(map[string]string),
		queryCalled:  make(map[string]int),
		streamCalled: make(map[string]int),
	}
}

// Connect simulates database connection.
// 模拟数据库连接。
func (m *MockDB) Connect(dataSourceName string, poolConfig *ConnPoolConfig) error {
	if m.shouldError {
		return fmt.Errorf("mock connection error: %s", m.errorMessage)
	}
	m.connected = true
	return nil
}

// Query simulates a database query and returns predefined results.
// 模拟数据库查询并返回预定义结果。
func (m *MockDB) Query(query string, args ...interface{}) ([][]interface{}, error) {
	if msg, ok := m.queryPanics[query]; ok {
		panic(msg)
	}
	if m.shouldError {
		return nil, fmt.Errorf("mock query error: %s", m.errorMessage)
	}
	if msg, ok := m.queryErrors[query]; ok {
		return nil, fmt.Errorf("mock query error: %s", msg)
	}
	m.queryCalled[query]++

	if results, ok := m.queryResults[query]; ok {
		return results, nil
	}
	return [][]interface{}{}, nil
}

// QueryStream simulates a streaming query.
// 模拟流式查询。
func (m *MockDB) QueryStream(query string, handler RowHandler, batchSize int) error {
	if msg, ok := m.streamPanics[query]; ok {
		panic(msg)
	}
	if m.shouldError {
		return fmt.Errorf("mock stream query error: %s", m.errorMessage)
	}
	if msg, ok := m.streamErrors[query]; ok {
		return fmt.Errorf("mock stream query error: %s", msg)
	}
	m.streamCalled[query]++

	if stream, ok := m.queryStreams[query]; ok {
		for _, next := range stream {
			row, hasMore := next()
			if !hasMore {
				break
			}
			if err := handler(row); err != nil {
				return err
			}
		}
	}
	return nil
}

// Execute simulates executing a non-query statement.
// 模拟执行非查询语句。
func (m *MockDB) Execute(query string, args ...interface{}) error {
	if m.shouldError {
		return fmt.Errorf("mock execute error: %s", m.errorMessage)
	}
	return nil
}

// Close simulates closing the database connection.
// 模拟关闭数据库连接。
func (m *MockDB) Close() error {
	m.closed = true
	m.connected = false
	return nil
}

// SetQueryResult sets the predefined results for a query.
// 设置查询的预定义结果。
func (m *MockDB) SetQueryResult(query string, results [][]interface{}) {
	m.queryResults[query] = results
}

// SetStreamResult sets the predefined results for a streaming query.
// 设置流式查询的预定义结果。
func (m *MockDB) SetStreamResult(query string, rows [][]interface{}) {
	var stream []func() ([]interface{}, bool)
	for _, row := range rows {
		stream = append(stream, func(r []interface{}) func() ([]interface{}, bool) {
			called := false
			return func() ([]interface{}, bool) {
				if called {
					return nil, false
				}
				called = true
				return r, true
			}
		}(row))
	}
	m.queryStreams[query] = stream
}

// SetQueryError registers a per-query error so Query() fails only for
// the given SQL without affecting other queries or Connect().
// SetQueryError 为指定 SQL 注册按查询错误，Query() 失败但不影响
// 其他查询和 Connect()。
func (m *MockDB) SetQueryError(query, message string) {
	m.queryErrors[query] = message
}

// SetStreamError registers a per-query error for streaming queries so
// QueryStream() fails only for the given SQL.
// SetStreamError 为流式查询注册按查询错误，QueryStream() 失败但不影响
// 其他流式查询。
func (m *MockDB) SetStreamError(query, message string) {
	m.streamErrors[query] = message
}

// SetQueryPanic registers a per-query panic so Query() panics for the
// given SQL. Used to verify that callers recover gracefully.
// SetQueryPanic 为指定 SQL 注册按查询 panic，Query() 会 panic。
// 用于验证调用方是否正确恢复。
func (m *MockDB) SetQueryPanic(query, message string) {
	m.queryPanics[query] = message
}

// SetStreamPanic registers a per-query panic for streaming queries.
// SetStreamPanic 为流式查询注册按查询 panic。
func (m *MockDB) SetStreamPanic(query, message string) {
	m.streamPanics[query] = message
}

// SetError simulates an error condition.
// 模拟错误条件。
func (m *MockDB) SetError(message string) {
	m.shouldError = true
	m.errorMessage = message
}

// ClearError clears the simulated error condition.
func (m *MockDB) ClearError() {
	m.shouldError = false
	m.errorMessage = ""
}

// IsConnected checks if the mock database is connected.
func (m *MockDB) IsConnected() bool {
	return m.connected
}

// IsClosed checks if the mock database is closed.
func (m *MockDB) IsClosed() bool {
	return m.closed
}

// GetQueryCount returns the number of times a query was called.
func (m *MockDB) GetQueryCount(query string) int {
	return m.queryCalled[query]
}

// GetStreamCount returns the number of times a streaming query was called.
func (m *MockDB) GetStreamCount(query string) int {
	return m.streamCalled[query]
}
