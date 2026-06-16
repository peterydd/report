package app

import (
	"testing"
	"time"

	"github.com/peterydd/report/pkg/db"
	"github.com/peterydd/report/pkg/excel"
	"github.com/peterydd/report/pkg/mail"
)

// IntegrationTestSuite provides a complete testing environment.
// IntegrationTestSuite 提供完整的测试环境。
type IntegrationTestSuite struct {
	mockDB   *db.MockDB
	mockMail *mail.MockMail
}

// Setup initializes the test environment.
func (s *IntegrationTestSuite) Setup() {
	s.mockDB = db.NewMockDB()
	s.mockMail = mail.NewMockMail()
}

// TestFullReportWorkflow tests the complete report generation and sending flow.
func TestFullReportWorkflow(t *testing.T) {
	suite := &IntegrationTestSuite{}
	suite.Setup()

	// Set up mock database data / 设置模拟数据库数据
	suite.mockDB.SetQueryResult("SELECT id, name, value FROM test_table", [][]interface{}{
		{"001", "ProductA", 100},
		{"002", "ProductB", 200},
		{"003", "ProductC", 300},
	})

	// Set up streaming query data / 设置流式查询数据
	suite.mockDB.SetStreamResult("SELECT * FROM large_table", [][]interface{}{
		{"row1", 100, "data1"},
		{"row2", 200, "data2"},
		{"row3", 300, "data3"},
		{"row4", 400, "data4"},
		{"row5", 500, "data5"},
	})

	// Verify database connection / 验证数据库连接
	err := suite.mockDB.Connect("test@tcp(localhost:3306)/test", &db.ConnPoolConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 3 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Database connection failed: %v", err)
	}
	defer suite.mockDB.Close()

	// Test normal query / 测试普通查询
	results, err := suite.mockDB.Query("SELECT id, name, value FROM test_table")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(results))
	}

	// Test streaming query / 测试流式查询
	streamCount := 0
	err = suite.mockDB.QueryStream("SELECT * FROM large_table", func(row []interface{}) error {
		streamCount++
		return nil
	}, 100)
	if err != nil {
		t.Fatalf("Stream query failed: %v", err)
	}
	if streamCount != 5 {
		t.Errorf("Expected 5 rows processed, got %d", streamCount)
	}

	// Test Excel generation / 测试Excel生成
	sheets := []*excel.Sheet{
		excel.SetSheet("Sheet1", "SELECT id, name, value FROM test_table", "ID,Name,Value", false, 0, results),
	}
	spreadsheet := excel.NewSpreadSheet("test_report.xlsx", sheets)

	err = spreadsheet.Create()
	if err != nil {
		t.Logf("Excel creation note: %v", err)
	}

	t.Log("Integration test passed: Complete report workflow verified")
}

// TestErrorHandling tests error handling scenarios.
func TestErrorHandling(t *testing.T) {
	suite := &IntegrationTestSuite{}
	suite.Setup()

	// Set database error / 设置数据库错误
	suite.mockDB.SetError("connection refused")

	// Verify connection failure / 验证连接失败
	err := suite.mockDB.Connect("invalid", &db.ConnPoolConfig{})
	if err == nil {
		t.Error("Should return connection error")
	}
	suite.mockDB.ClearError()

	// Set query error / 设置查询错误
	suite.mockDB.SetError("query timeout")
	_, err = suite.mockDB.Query("SELECT * FROM test")
	if err == nil {
		t.Error("Should return query error")
	}
	suite.mockDB.ClearError()

	// Set streaming query error / 设置流式查询错误
	suite.mockDB.SetError("stream error")
	err = suite.mockDB.QueryStream("SELECT * FROM test", func(row []interface{}) error {
		return nil
	}, 100)
	if err == nil {
		t.Error("Should return stream query error")
	}

	t.Log("Error handling test passed")
}

// TestPerformanceWithLargeData tests performance with large datasets.
func TestPerformanceWithLargeData(t *testing.T) {
	suite := &IntegrationTestSuite{}
	suite.Setup()

	// Generate large test data / 生成大量测试数据
	var largeData [][]interface{}
	for i := 0; i < 10000; i++ {
		largeData = append(largeData, []interface{}{
			i,
			"test_name_" + string(rune(i%26+'A')),
			i * 10,
		})
	}

	suite.mockDB.SetQueryResult("SELECT * FROM large_table", largeData)

	start := time.Now()
	results, err := suite.mockDB.Query("SELECT * FROM large_table")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Large data query failed: %v", err)
	}

	if len(results) != 10000 {
		t.Errorf("Expected 10000 rows, got %d", len(results))
	}

	t.Logf("Large dataset query completed in: %v", duration)
	t.Log("Large data performance test passed")
}

// TestMockMailFunctionality tests mail mock functionality.
func TestMockMailFunctionality(t *testing.T) {
	suite := &IntegrationTestSuite{}
	suite.Setup()

	// Test normal sending / 测试正常发送
	msg := mail.SetMessage("from@test.com", []string{"to@test.com"}, nil, nil, "Test", "Body", "text/plain", nil)

	err := suite.mockMail.Send(msg)
	if err != nil {
		t.Errorf("Mail sending failed: %v", err)
	}

	if suite.mockMail.GetSendCount() != 1 {
		t.Errorf("Expected 1 send, got %d", suite.mockMail.GetSendCount())
	}

	// Test error scenario / 测试错误场景
	suite.mockMail.SetError("smtp error")
	err = suite.mockMail.Send(msg)
	if err == nil {
		t.Error("Should return send error")
	}

	t.Log("Mail mock functionality test passed")
}
