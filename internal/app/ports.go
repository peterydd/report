/*
Package app - Dependency injection ports for the application.
包 app - 应用程序的依赖注入端口定义。

This file defines the minimal interfaces the App relies on, enabling
unit tests to substitute real DB / mail / excel implementations with
in-memory mocks without touching network or filesystem.
本文件定义 App 所依赖的最小接口，使单元测试能用内存 Mock 替换真实的
DB / 邮件 / Excel 实现，无需访问网络或文件系统。

Design / 设计要点:
- Reuse existing pkg/db.DB and pkg/mail.Mail interfaces (already mockable).
- Introduce SpreadSheetCreator to abstract *excel.SpreadSheet.Create().
- ReportDeps bundles all three plus a Config — pass once at construction.
- When ReportDeps is nil, Run() wires the production implementations.
*/
package app

import (
	"time"

	"github.com/peterydd/report/pkg/db"
	"github.com/peterydd/report/pkg/excel"
	"github.com/peterydd/report/pkg/mail"
)

// SpreadSheetCreator abstracts the Excel workbook generation step so tests
// can assert the workbook was emitted without writing a real .xlsx file.
// SpreadSheetCreator 抽象 Excel 工作簿生成步骤，使测试可在不写真实
// .xlsx 文件的前提下断言工作簿已生成。
type SpreadSheetCreator interface {
	Create() error
}

// SpreadSheetFactory builds a SpreadSheetCreator from a name and pre-built sheets.
// SpreadSheetFactory 根据名称与已构造的 sheets 构造 SpreadSheetCreator。
type SpreadSheetFactory func(name string, sheets []*excel.Sheet) SpreadSheetCreator

// MailFactory builds a mail.Mail from SMTP configuration. It is called once
// per report; tests may return a shared mock instance regardless of inputs.
// MailFactory 根据 SMTP 配置构造 mail.Mail。每个报表调用一次；
// 测试中可忽略入参返回共享 Mock 实例。
type MailFactory func(host, port, username, password string, insecureSkipVerify bool, timeout time.Duration) mail.Mail

// ReportDeps groups the injectable collaborators of App.
// When nil, Run() falls back to production constructors (config-driven DB
// and per-report SMTP client). Tests should populate every field.
// ReportDeps 汇总 App 的可注入协作者。nil 时 Run() 回退到生产构造器
// （配置驱动的 DB 与每个报表的 SMTP 客户端）。测试应填充全部字段。
type ReportDeps struct {
	DB          db.DB
	MailFactory MailFactory
	NewWorkbook SpreadSheetFactory
}
