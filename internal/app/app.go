/*
Package app - Main application entry point and orchestration.
包 app - 应用程序入口点和编排逻辑。

This package provides the main application logic for:
- Database connection management
- Report generation with Excel files
- Email sending with attachments
- Concurrent sheet processing

Features / 功能特性:
- Parallel sheet query execution / 并行工作表查询执行
- Streaming query support / 流式查询支持
- Connection pool management / 连接池管理
- Error resilience / 错误恢复能力
*/
package app

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/peterydd/report/pkg/config"
	"github.com/peterydd/report/pkg/db"
	"github.com/peterydd/report/pkg/excel"
	"github.com/peterydd/report/pkg/mail"
)

// maxConcurrentSheets caps the number of sheets that may query the database
// at the same time to avoid exhausting the connection pool.
// maxConcurrentSheets 限制同时查询数据库的工作表数量，避免耗尽连接池。
const maxConcurrentSheets = 8

// parseDuration parses a duration string, returning default value if empty or invalid.
// 解析时间字符串，如果为空或无效则返回默认值。
func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

// App represents the main application structure.
// 应用程序主结构体。
type App struct {
	*config.Config
	// Deps holds optional injected collaborators. When nil, Run() wires
	// production DB / mail / Excel implementations derived from Config.
	// Deps 保存可选的注入协作者。nil 时 Run() 使用从 Config 派生的
	// 生产 DB / 邮件 / Excel 实现。
	Deps *ReportDeps
}

// NewApp creates a new application instance by loading configuration.
// 创建应用程序实例并加载配置。
func NewApp() *App {
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}
	log.Printf("configuration loaded successfully")
	return &App{Config: conf}
}

// NewAppWithDeps creates an App with explicit injected collaborators.
// The caller is responsible for loading configuration. Use this constructor
// from tests to substitute mocks; production code should keep using NewApp.
// NewAppWithDeps 构造注入协作者显式的 App。配置由调用方加载。
// 测试用此构造函数替换 Mock；生产代码应继续使用 NewApp。
func NewAppWithDeps(conf *config.Config, deps *ReportDeps) *App {
	return &App{Config: conf, Deps: deps}
}

// Run executes the report generation and email sending workflow.
// 执行报表生成和邮件发送流程。
func (a *App) Run() error {
	deps, err := a.resolveDeps()
	if err != nil {
		return err
	}

	// Configure connection pool
	poolConfig := &db.ConnPoolConfig{
		MaxOpenConns:    a.Database.MaxOpenConns,
		MaxIdleConns:    a.Database.MaxIdleConns,
		ConnMaxLifetime: parseDuration(a.Database.ConnMaxLifetime, 3*time.Minute),
	}
	// Apply default values if not configured
	if poolConfig.MaxOpenConns == 0 {
		poolConfig.MaxOpenConns = 25
	}
	if poolConfig.MaxIdleConns == 0 {
		poolConfig.MaxIdleConns = 5
	}

	// Establish database connection
	if err := deps.DB.Connect(a.Database.Source, poolConfig); err != nil {
		return err
	}
	defer deps.DB.Close()

	// Process each report configuration
	for _, rp := range a.Reports {
		// Guard against the (test-only) case where a report has zero
		// sheets: the semaphore below is unbuffered when its capacity
		// is zero, which would deadlock the loop. NewAppWithDeps
		// bypasses Config.Validate so we cannot rely on it here.
		if len(rp.Sheets) == 0 {
			log.Printf("report %s skipped: no sheets configured", rp.Name)
			continue
		}
		sts := make([]*excel.Sheet, 0, len(rp.Sheets))

		var wg sync.WaitGroup
		sheetChan := make(chan *excel.Sheet, len(rp.Sheets))
		concurrency := len(rp.Sheets)
		if concurrency > maxConcurrentSheets {
			concurrency = maxConcurrentSheets
		}
		sem := make(chan struct{}, concurrency)

		// Process sheets concurrently (capped by maxConcurrentSheets)
		for _, st := range rp.Sheets {
			wg.Add(1)
			sem <- struct{}{}
			go func(sheetConfig *config.Sheet) {
				defer wg.Done()
				defer func() { <-sem }()
				// Recover from panics so one bad sheet cannot deadlock
				// the WaitGroup / sheetChan close and stall the report
				// loop. The panic is logged and the sheet is dropped.
				defer func() {
					if r := recover(); r != nil {
						log.Printf("sheet %s panicked: %v", sheetConfig.Name, r)
					}
				}()
				var sheet *excel.Sheet

				if sheetConfig.EnableStream {
					log.Printf("sheet %s using streaming mode, batch size: %d", sheetConfig.Name, sheetConfig.BatchSize)
					streamSheet := excel.SetSheetStream(sheetConfig.Name, sheetConfig.Sql, sheetConfig.Column, sheetConfig.IsSum, sheetConfig.SumBeginColumn, sheetConfig.BatchSize)

					rowCount := 0
					err := deps.DB.QueryStream(sheetConfig.Sql, func(row []interface{}) error {
						streamSheet.AddRow(row)
						rowCount++
						return nil
					}, sheetConfig.BatchSize)

					if err != nil {
						log.Printf("sheet %s streaming query failed: %v", sheetConfig.Name, err)
						return
					}
					log.Printf("sheet %s streaming query completed, %d rows processed", sheetConfig.Name, rowCount)
					sheet = streamSheet
				} else {
					data, err := deps.DB.Query(sheetConfig.Sql)
					if err != nil {
						log.Printf("sheet %s query failed: %v", sheetConfig.Name, err)
						return
					}
					log.Printf("sheet %s query completed, %d rows fetched", sheetConfig.Name, len(data))
					sheet = excel.SetSheet(sheetConfig.Name, sheetConfig.Sql, sheetConfig.Column, sheetConfig.IsSum, sheetConfig.SumBeginColumn, data)
				}

				sheetChan <- sheet
			}(st)
		}

		// Wait for all sheet processing to complete
		go func() {
			wg.Wait()
			close(sheetChan)
		}()

		// Collect processed sheets
		for s := range sheetChan {
			if s != nil {
				sts = append(sts, s)
			}
		}

		// Skip the report entirely if no sheet produced any data; sending
		// an empty workbook is rarely useful and often indicates a
		// configuration error that should surface in logs.
		if len(sts) == 0 {
			log.Printf("report %s skipped: all %d sheet(s) failed to produce data", rp.Name, len(rp.Sheets))
			continue
		}

		// Generate Excel workbook
		bookName := rp.WorkBook.Prefix + time.Now().Format(rp.WorkBook.DateFormat) + rp.WorkBook.Suffix
		sp := deps.NewWorkbook(bookName, sts)
		if err := sp.Create(); err != nil {
			log.Printf("report %s generation failed: %v", bookName, err)
			continue
		}
		log.Printf("report %s generated successfully", bookName)

		// Send email with attachment
		attachment := mail.SetAttach(bookName, rp.Message.Attachment.ContentType, rp.Message.Attachment.WithFile)
		message := mail.SetMessage(rp.Message.From, rp.Message.To, rp.Message.Cc, rp.Message.Bcc, rp.Message.Subject, rp.Message.Body, rp.Message.ContentType, attachment)
		sm := deps.MailFactory(a.Smtp.Host, a.Smtp.Port, a.Smtp.Username, a.Smtp.Password, a.Smtp.InsecureSkipVerify, parseDuration(a.Smtp.Timeout, 30*time.Second))
		if err := sm.Send(message); err != nil {
			log.Printf("email %s sending failed: %v", rp.Message.Subject, err)
			continue
		}
		log.Printf("email %s sent successfully", rp.Message.Subject)
	}
	return nil
}

// resolveDeps returns the injected ReportDeps or builds the production
// implementation derived from the loaded configuration.
// resolveDeps 返回注入的 ReportDeps；若未注入则基于已加载配置构造
// 生产实现。
func (a *App) resolveDeps() (*ReportDeps, error) {
	if a.Deps != nil {
		return a.Deps, nil
	}
	dbType := db.ParseDBType(a.Database.Driver)
	if dbType == -1 {
		return nil, fmt.Errorf("unsupported database driver: %s", a.Database.Driver)
	}
	return &ReportDeps{
		DB:          db.NewDB(dbType),
		MailFactory: realMailFactory,
		NewWorkbook: defaultWorkbookFactory,
	}, nil
}

// realMailFactory adapts mail.NewSendMail (which returns *SendMail) to the
// MailFactory port signature returning the mail.Mail interface.
// realMailFactory 将 mail.NewSendMail（返回 *SendMail）适配为返回
// mail.Mail 接口的 MailFactory。
func realMailFactory(host, port, username, password string, insecureSkipVerify bool, timeout time.Duration) mail.Mail {
	return mail.NewSendMail(host, port, username, password, insecureSkipVerify, timeout)
}

// defaultWorkbookFactory wraps excel.NewSpreadSheet behind the
// SpreadSheetCreator port. Centralised here so tests only need to mock once.
// defaultWorkbookFactory 将 excel.NewSpreadSheet 包装到 SpreadSheetCreator
// 端口之后，集中在此处让测试只需 mock 一次。
func defaultWorkbookFactory(name string, sheets []*excel.Sheet) SpreadSheetCreator {
	return excel.NewSpreadSheet(name, sheets)
}

// Start is an alias for Run() for backward compatibility.
// Run()的别名，保持向后兼容。
func (a *App) Start() error {
	return a.Run()
}
