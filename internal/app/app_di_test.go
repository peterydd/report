package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/peterydd/report/pkg/config"
	"github.com/peterydd/report/pkg/db"
	"github.com/peterydd/report/pkg/excel"
	"github.com/peterydd/report/pkg/mail"
)

// fakeWorkbook records every Create() call so tests can assert that the
// App loop emitted exactly one workbook per report configuration.
// fakeWorkbook 记录每次 Create() 调用，便于测试断言 App 循环为每个报表
// 配置生成了一个工作簿。
type fakeWorkbook struct {
	createCalls int
	lastName    string
	sheetCount  int
	failNext    bool
}

func (f *fakeWorkbook) Create() error {
	f.createCalls++
	if f.failNext {
		return errWorkbookFailed
	}
	return nil
}

// errWorkbookFailed is a sentinel error to make Create() fail on demand.
// errWorkbookFailed 是按需让 Create() 失败的哨兵错误。
var errWorkbookFailed = &workbookError{msg: "fake workbook failure"}

type workbookError struct{ msg string }

func (e *workbookError) Error() string { return e.msg }

// newFakeWorkbookFactory returns a SpreadSheetFactory that emits fresh fakes
// while sharing the provided recorder so the test can inspect the totals.
// newFakeWorkbookFactory 返回一个 SpreadSheetFactory，它生成独立的 fake
// 实例，同时把 workbook 名字记录在共享的 recorder 里供测试断言。
func newFakeWorkbookFactory(recorder *[]string) SpreadSheetFactory {
	return func(name string, sheets []*excel.Sheet) SpreadSheetCreator {
		fw := &fakeWorkbook{}
		// Lazy capture: deferred so the closure sees final sheetCount.
		fw.sheetCount = len(sheets)
		fw.lastName = name
		*recorder = append(*recorder, name)
		// We need both the name and a way to mutate the call count, so
		// wrap a shared pointer-like reference through a second recorder.
		fw.createCalls = 0
		// Use a stable list so tests can compare via the captured instance.
		fakeRegistry = append(fakeRegistry, fw)
		return fw
	}
}

// fakeRegistry accumulates fake workbooks so the test can iterate over them.
// fakeRegistry 累积 fake 工作簿，使测试可遍历。
var fakeRegistry []*fakeWorkbook

// sampleConfig builds an in-memory Config with one report and two sheets
// (one streaming, one normal) — enough to exercise both code paths.
// sampleConfig 构造一个内存 Config：一个报表、两个 sheet（一个流式、一个普通），
// 足以覆盖两条代码路径。
func sampleConfig() *config.Config {
	return &config.Config{
		Database: &config.Database{
			Driver:          "mysql",
			Source:          "user:pw@tcp(localhost:3306)/db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: "3m",
		},
		Smtp: &config.Smtp{
			Host:               "smtp.example.com",
			Port:               "587",
			Username:           "u",
			Password:           "p",
			InsecureSkipVerify: true,
			Timeout:            "30s",
		},
		Reports: []*config.Reports{
			{
				Name: "daily",
				WorkBook: &config.WorkBook{
					Prefix:     "rpt_",
					DateFormat: "20060102",
					Suffix:     ".xlsx",
				},
				Sheets: []*config.Sheet{
					{
						Name:         "Orders",
						Sql:          "SELECT id, name FROM orders",
						Column:       "ID,Name",
						IsSum:        false,
						SumBeginColumn: 0,
						BatchSize:    100,
						EnableStream: true,
					},
					{
						Name:         "Summary",
						Sql:          "SELECT id, total FROM summary",
						Column:       "ID,Total",
						IsSum:        true,
						SumBeginColumn: 2,
						BatchSize:    100,
						EnableStream: false,
					},
				},
				Message: &config.Message{
					From:        "from@example.com",
					To:          []string{"to@example.com"},
					Cc:          []string{"cc@example.com"},
					Bcc:         nil,
					Subject:     "Daily Report",
					Body:        "<p>see attached</p>",
					ContentType: "text/html",
					Attachment: &config.Attachment{
						ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
						WithFile:    false,
					},
				},
			},
		},
	}
}

// resetTestState clears package-level fake registry and any captured
// workbook names between tests. Must be called at the start of each test.
// resetTestState 在测试间清除包级 fake 注册表与已捕获的 workbook 名称。
func resetTestState() {
	fakeRegistry = nil
}

// TestAppRun_WithDeps_HappyPath verifies the end-to-end flow with all
// collaborators mocked: DB returns canned rows, mail is captured,
// workbook creation is recorded, and Run() returns nil.
// TestAppRun_WithDeps_HappyPath 验证所有协作者被 Mock 时的端到端流程：
// DB 返回预设行，邮件被捕获，workbook 创建被记录，Run() 返回 nil。
func TestAppRun_WithDeps_HappyPath(t *testing.T) {
	resetTestState()

	mockDB := db.NewMockDB()
	mockDB.SetStreamResult("SELECT id, name FROM orders", [][]interface{}{
		{1, "alice"},
		{2, "bob"},
	})
	mockDB.SetQueryResult("SELECT id, total FROM summary", [][]interface{}{
		{1, 100},
		{2, 200},
	})

	mockMail := mail.NewMockMail()
	var captured []string
	factory := newFakeWorkbookFactory(&captured)

	deps := &ReportDeps{
		DB:          mockDB,
		MailFactory: func(_, _, _, _ string, _ bool, _ time.Duration) mail.Mail { return mockMail },
		NewWorkbook: factory,
	}

	a := NewAppWithDeps(sampleConfig(), deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if mockMail.GetSendCount() != 1 {
		t.Errorf("expected 1 email sent, got %d", mockMail.GetSendCount())
	}
	if len(captured) != 1 {
		t.Fatalf("expected 1 workbook created, got %d", len(captured))
	}
	if filepath.Ext(captured[0]) != ".xlsx" {
		t.Errorf("workbook name missing .xlsx suffix: %s", captured[0])
	}
	if got := mockDB.GetStreamCount("SELECT id, name FROM orders"); got != 1 {
		t.Errorf("expected 1 streaming query for orders, got %d", got)
	}
	if got := mockDB.GetQueryCount("SELECT id, total FROM summary"); got != 1 {
		t.Errorf("expected 1 normal query for summary, got %d", got)
	}
}

// TestAppRun_WithDeps_DBError ensures the error from Connect() propagates
// and the loop is never entered.
// TestAppRun_WithDeps_DBError 验证 Connect() 的错误被传播，且循环未被进入。
func TestAppRun_WithDeps_DBError(t *testing.T) {
	resetTestState()

	mockDB := db.NewMockDB()
	mockDB.SetError("connection refused")

	deps := &ReportDeps{
		DB:          mockDB,
		MailFactory: func(_, _, _, _ string, _ bool, _ time.Duration) mail.Mail { return mail.NewMockMail() },
		NewWorkbook: newFakeWorkbookFactory(&[]string{}),
	}

	a := NewAppWithDeps(sampleConfig(), deps)
	if err := a.Run(); err == nil {
		t.Fatal("expected Run() to return an error when DB.Connect fails")
	}
	if mockDB.IsConnected() {
		t.Error("expected mock DB to report disconnected after failed Connect")
	}
}

// TestAppRun_WithDeps_MultipleReports ensures the loop processes every
// report in the configuration (not just the first one).
// TestAppRun_WithDeps_MultipleReports 验证循环处理配置中的每个报表
// （而非仅第一个）。
func TestAppRun_WithDeps_MultipleReports(t *testing.T) {
	resetTestState()

	cfg := sampleConfig()
	cfg.Reports = append(cfg.Reports, &config.Reports{
		Name: "weekly",
		WorkBook: &config.WorkBook{
			Prefix:     "wk_",
			DateFormat: "2006-01-02",
			Suffix:     ".xlsx",
		},
		Sheets: []*config.Sheet{
			{
				Name:         "Weekly",
				Sql:          "SELECT * FROM weekly",
				Column:       "A,B",
				BatchSize:    50,
				EnableStream: false,
			},
		},
		Message: &config.Message{
			From:        "from@example.com",
			To:          []string{"to@example.com"},
			Subject:     "Weekly",
			Body:        "x",
			ContentType: "text/plain",
			Attachment: &config.Attachment{
				ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				WithFile:    false,
			},
		},
	})

	mockDB := db.NewMockDB()
	mockDB.SetQueryResult("SELECT * FROM weekly", [][]interface{}{{"v", 1}})
	mockMail := mail.NewMockMail()
	var captured []string

	deps := &ReportDeps{
		DB:          mockDB,
		MailFactory: func(_, _, _, _ string, _ bool, _ time.Duration) mail.Mail { return mockMail },
		NewWorkbook: newFakeWorkbookFactory(&captured),
	}

	a := NewAppWithDeps(cfg, deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if mockMail.GetSendCount() != 2 {
		t.Errorf("expected 2 emails, got %d", mockMail.GetSendCount())
	}
	if len(captured) != 2 {
		t.Errorf("expected 2 workbooks, got %d", len(captured))
	}
}

// TestResolveDeps_NilDriverFails validates that resolveDeps reports an
// unsupported driver when Deps is not injected. This covers the production
// default-code path in isolation.
// TestResolveDeps_NilDriverFails 验证未注入 Deps 时 resolveDeps 会对
// 不受支持的驱动报错，单独覆盖生产默认代码路径。
func TestResolveDeps_NilDriverFails(t *testing.T) {
	cfg := sampleConfig()
	cfg.Database.Driver = "unknown-driver"
	a := NewAppWithDeps(cfg, nil)
	_, err := a.resolveDeps()
	if err == nil {
		t.Fatal("expected resolveDeps to fail for unsupported driver")
	}
}
