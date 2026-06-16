/*
Report Generator - A high-performance data reporting and email system.
报表生成器 - 高性能数据报表生成和邮件发送系统。

This application provides:
- Multi-database support (MySQL, PostgreSQL, Oracle, ClickHouse)
- Excel report generation with multiple sheets
- Automatic email sending with attachments
- Streaming query support for large datasets

Usage / 使用方法:

	report [options]

Options / 选项:

	-version    Show version information
	-help       Show help information

Configuration / 配置:

	Default configuration file: ./config.yaml
	Supported paths: ./, ./configs/, /
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/peterydd/report/internal/app"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("report version %s (built at %s)\n", version, buildTime)
		os.Exit(0)
	}

	if *showHelp {
		fmt.Println("report - High-performance data reporting and email system")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  report [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -version    Show version information")
		fmt.Println("  -help       Show help information")
		fmt.Println()
		fmt.Println("Configuration:")
		fmt.Println("  Default: ./config.yaml")
		fmt.Println("  Supported paths: ./, ./configs/, /")
		os.Exit(0)
	}

	a := app.NewApp()
	if err := a.Run(); err != nil {
		log.Fatalf("report generation and sending failed: %v", err)
	}
	log.Println("report generation and sending completed successfully")
}
