/*
Package app - Pure mock unit tests for App.Run() failure paths.
包 app - App.Run() 失败路径的纯 Mock 单元测试。

These tests do not touch the network or filesystem; they drive App
through the dependency injection ports introduced in ports.go and
assert that every failure mode is handled gracefully (i.e. the
affected report is skipped, the loop continues, Run() returns nil).
本文件中的测试不访问网络或文件系统；通过 ports.go 引入的依赖注入
端口驱动 App，并断言每种失败模式都被优雅处理（受影响的报表被跳过、
循环继续、Run() 返回 nil）。
*/
package app

import (
	"testing"
	"time"

	"github.com/peterydd/report/pkg/config"
	"github.com/peterydd/report/pkg/db"
	"github.com/peterydd/report/pkg/excel"
	"github.com/peterydd/report/pkg/mail"
)

// buildDeps wires the minimum collaborators required by App.Run() with
// fresh mocks. The returned recorder lets callers inspect which
// workbooks were emitted.
// buildDeps 用全新 Mock 连接 App.Run() 所需的最少协作者；返回的
// recorder 允许调用方检查哪些 workbook 被生成。
func buildDeps(t *testing.T) (*ReportDeps, *db.MockDB, *mail.MockMail, *[]string) {
	t.Helper()
	mockDB := db.NewMockDB()
	mockMail := mail.NewMockMail()
	var captured []string
	return &ReportDeps{
		DB:          mockDB,
		MailFactory: func(_, _, _, _ string, _ bool, _ time.Duration) mail.Mail { return mockMail },
		NewWorkbook: newFakeWorkbookFactory(&captured),
	}, mockDB, mockMail, &captured
}

// TestAppRun_StreamQueryError verifies that a streaming query error is
// logged, the affected sheet is dropped, but the workbook and email
// for the surrounding report still complete using the remaining
// healthy sheet.
// TestAppRun_StreamQueryError 验证流式查询出错时会被记录，受影响的
// sheet 被丢弃，但同一报表的 workbook 和邮件仍由剩余健康 sheet 完成。
func TestAppRun_StreamQueryError(t *testing.T) {
	resetTestState()

	mockDB := db.NewMockDB()
	mockDB.SetStreamError("SELECT id, name FROM orders", "simulated stream failure")
	mockDB.SetQueryResult("SELECT id, total FROM summary", [][]interface{}{{1, 10}})

	mockMail := mail.NewMockMail()
	var captured []string

	deps := &ReportDeps{
		DB:          mockDB,
		MailFactory: func(_, _, _, _ string, _ bool, _ time.Duration) mail.Mail { return mockMail },
		NewWorkbook: newFakeWorkbookFactory(&captured),
	}

	a := NewAppWithDeps(sampleConfig(), deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if got := mockMail.GetSendCount(); got != 1 {
		t.Errorf("expected 1 email despite stream failure, got %d", got)
	}
	if len(captured) != 1 {
		t.Errorf("expected 1 workbook, got %d", len(captured))
	}
	if len(fakeRegistry) != 1 || fakeRegistry[0].sheetCount != 1 {
		t.Errorf("expected the surviving sheet to be passed to the workbook, got %d sheet(s)", len(fakeRegistry))
	}
}

// TestAppRun_NormalQueryError verifies that a non-streaming query
// failure logs and drops the affected sheet while the other sheet
// (and the surrounding email) still complete.
// TestAppRun_NormalQueryError 验证非流式查询失败会被记录并丢弃受影响的
// sheet，其他 sheet（以及对应的邮件）仍能完成。
func TestAppRun_NormalQueryError(t *testing.T) {
	resetTestState()

	mockDB := db.NewMockDB()
	mockDB.SetStreamResult("SELECT id, name FROM orders", [][]interface{}{{1, "a"}})
	mockDB.SetQueryError("SELECT id, total FROM summary", "simulated query failure")

	mockMail := mail.NewMockMail()
	var captured []string

	deps := &ReportDeps{
		DB:          mockDB,
		MailFactory: func(_, _, _, _ string, _ bool, _ time.Duration) mail.Mail { return mockMail },
		NewWorkbook: newFakeWorkbookFactory(&captured),
	}

	a := NewAppWithDeps(sampleConfig(), deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if got := mockMail.GetSendCount(); got != 1 {
		t.Errorf("expected 1 email despite query failure, got %d", got)
	}
	if len(captured) != 1 {
		t.Errorf("expected 1 workbook, got %d", len(captured))
	}
	if len(fakeRegistry) != 1 || fakeRegistry[0].sheetCount != 1 {
		t.Errorf("expected exactly the streaming sheet to reach the workbook, got sheetCount=%d", fakeRegistry[0].sheetCount)
	}
}

// TestAppRun_WorkbookCreateFails verifies that a workbook creation
// failure prevents email sending for that report and the loop moves
// on to subsequent reports.
// TestAppRun_WorkbookCreateFails 验证 workbook 创建失败时会阻止该报表
// 的邮件发送，且循环继续处理后续报表。
func TestAppRun_WorkbookCreateFails(t *testing.T) {
	resetTestState()

	cfg := sampleConfig()
	cfg.Reports = append(cfg.Reports, &config.Reports{
		Name: "second",
		WorkBook: &config.WorkBook{
			Prefix:     "sec_",
			DateFormat: "20060102",
			Suffix:     ".xlsx",
		},
		Sheets: []*config.Sheet{
			{
				Name:         "All",
				Sql:          "SELECT * FROM second",
				Column:       "A",
				BatchSize:    10,
				EnableStream: false,
			},
		},
		Message: &config.Message{
			From:        "from@example.com",
			To:          []string{"to@example.com"},
			Subject:     "Second",
			Body:        "x",
			ContentType: "text/plain",
			Attachment: &config.Attachment{
				ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				WithFile:    false,
			},
		},
	})

	mockDB := db.NewMockDB()
	mockDB.SetQueryResult("SELECT id, total FROM summary", [][]interface{}{{1, 10}})
	mockDB.SetQueryResult("SELECT * FROM second", [][]interface{}{{99}})

	mockMail := mail.NewMockMail()
	var captured []string

	failOn := map[string]bool{"rpt_": true}
	factory := func(name string, sheets []*excel.Sheet) SpreadSheetCreator {
		fw := &fakeWorkbook{lastName: name, sheetCount: len(sheets)}
		captured = append(captured, name)
		fakeRegistry = append(fakeRegistry, fw)
		if failOn[name[:4]] {
			fw.failNext = true
		}
		return fw
	}

	deps := &ReportDeps{
		DB:          mockDB,
		MailFactory: func(_, _, _, _ string, _ bool, _ time.Duration) mail.Mail { return mockMail },
		NewWorkbook: factory,
	}

	a := NewAppWithDeps(cfg, deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if got := mockMail.GetSendCount(); got != 1 {
		t.Errorf("expected 1 email (only the second report succeeds), got %d", got)
	}
	if len(captured) != 2 {
		t.Fatalf("expected 2 workbook attempts, got %d", len(captured))
	}
	if fakeRegistry[0].createCalls != 1 || fakeRegistry[1].createCalls != 1 {
		t.Errorf("expected Create() called once per workbook, got %d and %d", fakeRegistry[0].createCalls, fakeRegistry[1].createCalls)
	}
}

// TestAppRun_MailSendFails verifies that a Send() error stops email
// for that report without aborting Run(); the next report still
// gets sent.
// TestAppRun_MailSendFails 验证 Send() 错误会阻止该报表的邮件但不会
// 中断 Run()，下一封邮件仍能发送。
func TestAppRun_MailSendFails(t *testing.T) {
	resetTestState()

	cfg := sampleConfig()
	cfg.Reports = append(cfg.Reports, &config.Reports{
		Name: "weekly",
		WorkBook: &config.WorkBook{
			Prefix:     "wk_",
			DateFormat: "20060102",
			Suffix:     ".xlsx",
		},
		Sheets: []*config.Sheet{
			{
				Name:         "W",
				Sql:          "SELECT 1",
				Column:       "X",
				BatchSize:    10,
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

	deps, mockDB, mockMail, _ := buildDeps(t)
	mockDB.SetQueryResult("SELECT id, total FROM summary", [][]interface{}{{1, 10}})
	mockDB.SetQueryResult("SELECT 1", [][]interface{}{{1}})

	failingMail := mail.NewMockMail()
	failingMail.SetError("simulated smtp failure")
	var returnFirst, returnSecond bool
	deps.MailFactory = func(_, _, _, _ string, _ bool, _ time.Duration) mail.Mail {
		if !returnFirst {
			returnFirst = true
			return failingMail
		}
		returnSecond = true
		return mockMail
	}

	a := NewAppWithDeps(cfg, deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if failingMail.GetSendCount() != 1 {
		t.Errorf("expected 1 send attempt on failing mailer, got %d", failingMail.GetSendCount())
	}
	if !returnSecond {
		t.Error("expected MailFactory to be called for the second report after the first Send failed")
	}
	if mockMail.GetSendCount() != 1 {
		t.Errorf("expected 1 successful send on the fallback mailer, got %d", mockMail.GetSendCount())
	}
}

// TestAppRun_AllSheetsDropped verifies that when every sheet in a
// report fails to produce data, no workbook is created and no email
// is sent for that report, but Run() still returns nil.
// TestAppRun_AllSheetsDropped 验证报表中所有 sheet 全部失败时不会
// 生成 workbook 也不会发送邮件，但 Run() 仍返回 nil。
func TestAppRun_AllSheetsDropped(t *testing.T) {
	resetTestState()

	deps, mockDB, mockMail, captured := buildDeps(t)
	mockDB.SetStreamError("SELECT id, name FROM orders", "stream boom")
	mockDB.SetQueryError("SELECT id, total FROM summary", "query boom")

	a := NewAppWithDeps(sampleConfig(), deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if mockMail.GetSendCount() != 0 {
		t.Errorf("expected 0 emails when all sheets fail, got %d", mockMail.GetSendCount())
	}
	if len(*captured) != 0 {
		t.Errorf("expected 0 workbooks when all sheets fail, got %d", len(*captured))
	}
}

// TestAppRun_MailCapturesRecipients verifies that the To/Cc/Bcc
// configured in the report are propagated to the email message that
// hits the mailer.
// TestAppRun_MailCapturesRecipients 验证报表中配置的 To/Cc/Bcc 被
// 正确传递到发送的邮件消息中。
func TestAppRun_MailCapturesRecipients(t *testing.T) {
	resetTestState()

	cfg := sampleConfig()
	cfg.Reports[0].Message.To = []string{"alice@example.com", "bob@example.com"}
	cfg.Reports[0].Message.Cc = []string{"carol@example.com"}
	cfg.Reports[0].Message.Bcc = []string{"dave@example.com"}
	cfg.Reports[0].Message.Subject = "Hello"

	deps, mockDB, mockMail, _ := buildDeps(t)
	mockDB.SetQueryResult("SELECT id, total FROM summary", [][]interface{}{{1, 10}})

	a := NewAppWithDeps(cfg, deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	msgs := mockMail.GetSentMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 captured message, got %d", len(msgs))
	}
	if got := msgs[0].Subject; got != "Hello" {
		t.Errorf("subject = %q, want %q", got, "Hello")
	}
	if len(msgs[0].To) != 2 || msgs[0].To[0] != "alice@example.com" || msgs[0].To[1] != "bob@example.com" {
		t.Errorf("To recipients not preserved: %v", msgs[0].To)
	}
}

// TestAppRun_PanicInSheetGoroutine_Recovered verifies that a panic
// raised inside a sheet's processing goroutine is recovered by
// Run(), the affected sheet is dropped, and the surrounding
// report loop continues normally.
// TestAppRun_PanicInSheetGoroutine_Recovered 验证 sheet 处理 goroutine
// 内 panic 会被 Run() 恢复，受影响 sheet 被丢弃，外层报表循环正常继续。
func TestAppRun_PanicInSheetGoroutine_Recovered(t *testing.T) {
	resetTestState()

	deps, mockDB, mockMail, captured := buildDeps(t)
	mockDB.SetStreamPanic("SELECT id, name FROM orders", "boom in stream sheet")
	mockDB.SetQueryResult("SELECT id, total FROM summary", [][]interface{}{{1, 10}})

	a := NewAppWithDeps(sampleConfig(), deps)
	if err := a.Run(); err != nil {
		t.Fatalf("Run() returned error (expected nil because panic is recovered): %v", err)
	}

	if got := mockMail.GetSendCount(); got != 1 {
		t.Errorf("expected 1 email after recovered panic, got %d", got)
	}
	if len(*captured) != 1 {
		t.Errorf("expected 1 workbook after recovered panic, got %d", len(*captured))
	}
	if len(fakeRegistry) != 1 || fakeRegistry[0].sheetCount != 1 {
		t.Errorf("expected the healthy sheet to be the only one in the workbook, got sheetCount=%d", fakeRegistry[0].sheetCount)
	}
}

// TestAppRun_ReportWithZeroSheets_NoDeadlock verifies that a report
// with no sheets is skipped instead of deadlocking on the unbuffered
// semaphore. This guards the NewAppWithDeps code path that bypasses
// Config.Validate.
// TestAppRun_ReportWithZeroSheets_NoDeadlock 验证无 sheet 的报表
// 会被跳过而不是在未缓冲信号量上死锁。这保护绕过 Config.Validate
// 的 NewAppWithDeps 路径。
func TestAppRun_ReportWithZeroSheets_NoDeadlock(t *testing.T) {
	resetTestState()

	cfg := sampleConfig()
	cfg.Reports[0].Sheets = nil

	deps, _, mockMail, captured := buildDeps(t)

	done := make(chan error, 1)
	go func() {
		done <- NewAppWithDeps(cfg, deps).Run()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() deadlocked on a zero-sheet report")
	}

	if mockMail.GetSendCount() != 0 {
		t.Errorf("expected 0 emails for zero-sheet report, got %d", mockMail.GetSendCount())
	}
	if len(*captured) != 0 {
		t.Errorf("expected 0 workbooks, got %d", len(*captured))
	}
}
