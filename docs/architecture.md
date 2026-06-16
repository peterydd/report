# 架构文档

> 适用版本：v1.0.x  
> 配套阅读：[配置参考](configuration.md) · [开发指南](development.md) · [运维指南](operations.md)

## 1. 设计目标

| 目标 | 现状 | 备注 |
|------|------|------|
| 单进程内多报表顺序执行 | ✅ | 通过 `Reports` 切片 |
| 单报表内多 Sheet 并发查询 | ✅ | 受 `maxConcurrentSheets=8` 信号量约束 |
| 百万行低内存 | ✅ | `pkg/db.QueryStream` + `pkg/excel.SetSheetStream` |
| 配置零停机生效 | ✅ | viper `WatchConfig` + `OnConfigChange` |
| 单二进制、可容器化 | ✅ | 33MB 静态二进制；多阶段 Dockerfile |
| 可测试 | ✅ | `db.MockDB` / `mail.MockMail` 覆盖所有边界 |

## 2. 分层

```
┌─────────────────────────────────────────────────────────────┐
│  CLI 层 (cmd/report)                                         │
│  - flag 解析（-version / -help）                              │
│  - ldflags 注入版本                                          │
│  - 调用 app.NewApp().Run()                                   │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│  应用编排层 (internal/app)                                   │
│  - 加载/校验配置                                             │
│  - 建立 DB 连接并 defer Close                                │
│  - 遍历 Reports：                                            │
│      · 为每个 Sheet 启动 goroutine（受信号量限流）            │
│      · 收集结果到 channel                                    │
│      · 渲染 Excel → 发送邮件                                 │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────┬───────────────┼───────────────┬─────────────────┐
│         │               │               │                 │
▼         ▼               ▼               ▼                 ▼
┌──────┐ ┌─────────┐  ┌────────┐  ┌─────────┐      ┌──────────┐
│ db   │ │ excel   │  │ mail   │  │ config  │      │ (扩展点) │
│ pkg  │ │  pkg    │  │  pkg   │  │  pkg    │      │          │
└──────┘ └─────────┘  └────────┘  └─────────┘      └──────────┘
  4驱动   excelize     smtp        viper
  Mock    冻结+SUM     附件+TLS   热重载
```

### 2.1 依赖方向

```
cmd/report ──▶ internal/app ──▶ pkg/{db,excel,mail,config}
```

`internal/app` 不被 `pkg/*` 引用，避免循环依赖；`pkg/*` 之间也保持独立，仅通过标准库协作。

### 2.2 接口边界

| 接口 | 定义位置 | 实现 |
|------|----------|------|
| `db.DB`        | `pkg/db/db.go:49`         | MySQL/Oracle/PostgreSQL/ClickHouse + Mock |
| `mail.Mail`    | `pkg/mail/mail.go:37`     | SMTP 真实 + Mock |

`internal/app` 期望通过 `db.NewDB(...)` 与 `mail.NewSendMail(...)` 拿到具体实现；理论上可注入 `MockDB`/`MockMail` 进行端到端测试（参见 [开发指南](development.md) 中「依赖注入」一节）。

## 3. 关键流程

### 3.1 启动时序

```
main() ──▶ app.NewApp()
                │
                ├─ config.NewConfig()
                │     ├─ viper.ReadInConfig()
                │     ├─ viper.Unmarshal(&Config)
                │     └─ Config.Validate()
                │
                └─ viper.WatchConfig() + OnConfigChange(...)
                              │
                              └─ 热重载时重新 Unmarshal + Validate
                                 并赋值回 cfg.Database / cfg.Smtp / cfg.Reports
                                 最后调用 cfg.NotifyChange()

main() ──▶ app.Run()
                │
                ├─ db.ParseDBType(driver)
                ├─ db.NewDB(...) + Connect(source, poolConfig)
                ├─ for each report { for each sheet (concurrent) { ... } }
                └─ defer database.Close()
```

### 3.2 Sheet 并发模型

```go
sem := make(chan struct{}, min(len(sheets), maxConcurrentSheets))

for _, sheet := range sheets {
    wg.Add(1)
    sem <- struct{}{}                       // 获取令牌（可能阻塞）
    go func(s) {
        defer wg.Done()
        defer func() { <-sem }()           // 归还令牌
        // QueryStream / Query → 填充 sheet → sheetChan <- sheet
    }(sheet)
}

go func() { wg.Wait(); close(sheetChan) }() // 关闭信号
for s := range sheetChan { sts = append(sts, s) } // 收集
```

要点：
- **信号量**（channel buffer）控制并发上限为 8，避免耗尽 `maxOpenConns`
- **收集者单 goroutine** 通过 `close(sheetChan)` 与所有 worker 解耦，无需额外协调
- 单 sheet 失败仅 `log.Printf` 并丢弃，不会中断整个报表

### 3.3 流式查询 → 流式写 Sheet

```go
err := db.QueryStream(sql, func(row []interface{}) error {
    streamSheet.AddRow(row)  // 追加到内部 [][]interface{}
    return nil
}, batchSize)
```

> 未来优化：在 `excelize.StreamWriter` 上直接写入，内存占用可降至每行 O(1)。当前实现仍把数据驻留在 `Sheet.data`，参见 [ROADMAP v1.1](../ROADMAP.md)。

## 4. 数据结构

### 4.1 `config.Config`

```
Config
├── Database     *Database     // DSN + 连接池
├── Smtp         *Smtp         // host/port/user/pass/timeout
├── Reports      []*Reports
│     └── Reports
│         ├── Name    string
│         ├── WorkBook *WorkBook    // 文件名模板
│         ├── Sheets   []*Sheet
│         │     └── Sheet
│         │         ├── Name, Sql, Column
│         │         ├── IsSum, SumBeginColumn
│         │         └── EnableStream, BatchSize
│         └── Message  *Message
│               ├── From, To[], Cc[], Bcc[], Subject, Body, ContentType
│               └── Attachment *Attachment
├── mu           sync.RWMutex
└── onChange     func(*Config)
```

`Config.Validate()` 对所有必填字段做零值检查并返回首个错误。

### 4.2 `excel.Sheet`

| 字段 | 作用 |
|------|------|
| `data [][]interface{}` | 完整数据或流式累积 |
| `batchSize int` | 流式批次大小（仅记录，不影响 excelize） |
| `enableStream bool` | 标记 |
| `rowCount int` | 流式下只增不重复计算 |

### 4.3 `mail.Message`

| 字段 | 关键点 |
|------|--------|
| `from / to / cc / bcc` | 都为字符串切片；`bcc` 不会出现在 `Header` 中（隐私）但会出现在 `Rcpt` 中 |
| `contentType` | 如 `text/plain;charset=utf-8` / `text/html;charset=utf-8` |
| `attachment *Attach` | `withFile=true` 时 `mail.Send` 会读取 `name` 指定的文件并 base64 编码 |

## 5. 错误处理策略

| 层 | 策略 |
|----|------|
| `cmd/report/main.go` | 错误时 `log.Fatalf`（CLI 进程必须失败可见） |
| `internal/app.Run` | 单 sheet / 单报表错误 `log.Printf` 并跳过，**整个 Run 不返回错误**（避免一个报表失败导致其他报表也无法发送） |
| `pkg/*` | 通过 `fmt.Errorf("...: %w", err)` 包装，调用方判断 `errors.Is/As` |
| `pkg/db` | 错误时关闭 rows，避免连接泄漏 |
| `pkg/mail` | 任意一步失败都会 `c.Close()`，SMTP 连接不留尾巴 |

## 6. 可观测性

当前实现使用 `log.Printf` 输出关键事件：

| 事件 | 格式 |
|------|------|
| 配置加载 | `configuration loaded successfully` |
| Sheet 流式 | `sheet %s using streaming mode, batch size: %d` |
| Sheet 查询完成 | `sheet %s query completed, %d rows fetched` |
| 报表生成 | `report %s generated successfully` |
| 邮件发送 | `email %s sent successfully` |
| 失败 | `... failed: %v` |

生产建议：替换为结构化日志库（zap/zerolog）并接入集中日志系统。详细操作见 [运维指南](operations.md#日志采集)。

## 7. 扩展点

### 7.1 添加新的数据库驱动

1. 在 `pkg/db/db.go` 中追加 `DBType` 常量
2. 实现 `Connect / Query / QueryStream / Execute / Close`
3. 在 `NewDBFactory` 中注册
4. 写一组 Mock + 集成测试

### 7.2 添加新的导出格式（PDF / CSV）

1. 创建 `pkg/pdf` 或 `pkg/csv`，实现类似 `NewSpreadSheet(...).Create()` 的接口
2. 在 `internal/app` 中根据配置选择导出器
3. 文档：[开发指南 → 添加新导出格式](development.md#添加新的导出格式)

### 7.3 替换为更细粒度的依赖注入

将 `internal/app` 改造为：

```go
type App struct {
    DBFactory   func() db.DB
    MailFactory func() mail.Mail
    Config      *config.Config
}
```

从而在测试中注入 `MockDB` / `MockMail`，无须真实环境。详见 [开发指南](development.md#依赖注入)。

## 8. 已知限制

| 限制 | 影响 | 计划 |
|------|------|------|
| 邮件发送阻塞主流程 | 大量小报表串行 | v1.1 worker pool |
| 流式 Sheet 数据仍驻内存 | 极大数据量（>10M 行）仍可能 OOM | v1.1 切到 `excelize.StreamWriter` |
| 无重试 | 网络抖动即失败 | v1.1 |
| 仅 SMTP 25/465/587 | 其他传输方式不支持 | 长期 |
| 无审计/监控指标 | 难以追责 | v1.2 |

## 9. 决策记录（ADR）

- **为什么用 `database/sql` 而不是 ORM？** 报表场景下 ORM 抽象成本不划算，且 `database/sql` 自带连接池。
- **为什么用 `excelize/v2`？** Go 生态最成熟的 xlsx 库，支持公式/冻结/流式写。
- **为什么不用 `gomail.v2`？** 它更现代，但本项目起步早（v0.1），手写 MIME 也能完全控制；后续 v1.2 会评估迁移。
- **为什么 internal 包叫 `app` 而不是 `report`？** 避免与项目名 `report` 重复（见 [migration.md](migration.md)）。
