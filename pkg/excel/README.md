# pkg/excel

<p align="center">
  <a href="README.md">中文</a> | <a href="README.en.md">English</a>
</p>

基于 [excelize/v2](https://github.com/xuri/excelize) 的报表生成包：多 Sheet、自动求和、冻结表头、流式写入。

> 项目主页：[README](../../README.md) · 架构：[docs/architecture.md](../../docs/architecture.md)

## 特性

- ✅ 多 Sheet 工作簿
- ✅ 自定义表头（逗号分隔）
- ✅ 自动 SUM 公式（`isSum=true`）
- ✅ 冻结首行（`Freeze: true, YSplit: 1`）
- ✅ 流式 Sheet：`SetSheetStream` + `AddRow` 增量追加
- ✅ 中英文件名/列名兼容

## 安装

```bash
go get github.com/peterydd/report/pkg/excel
```

## 快速上手

```go
import (
    "log"
    "github.com/peterydd/report/pkg/excel"
)

// 普通 Sheet
data := [][]interface{}{
    {"2024-01-01", 100},
    {"2024-01-02", 200},
}
sheet := excel.SetSheet("Sales", "SELECT ...", "Date,Amount", true, 2, data)

// 流式 Sheet（大数据量）
stream := excel.SetSheetStream("Logs", "SELECT ...", "TS,Level,Msg", false, 0, 10000)
stream.AddRow([]interface{}{"2024-01-01 10:00", "INFO", "started"})
stream.AddRow([]interface{}{"2024-01-01 10:01", "INFO", "ready"})

// 渲染
spreadsheet := excel.NewSpreadSheet("report.xlsx", []*excel.Sheet{sheet, stream})
if err := spreadsheet.Create(); err != nil {
    log.Fatal(err)
}
```

## 导出 API

| 符号 | 说明 |
|------|------|
| `SetSheet(name, sql, column string, isSum bool, sumBeginCol int, data [][]interface{}) *Sheet` | 构造普通 Sheet（数据一次性传入） |
| `SetSheetStream(name, sql, column string, isSum bool, sumBeginCol, batchSize int) *Sheet` | 构造流式 Sheet（`batchSize<=0` 时默认 10000） |
| `(*Sheet).AddRow(row []interface{})` | 流式追加一行 |
| `(*Sheet).GetRowCount() int` | 行数（含表头？否，仅数据行） |
| `(*Sheet).ClearData()` | 清空数据，节省内存 |
| `NewSpreadSheet(name string, sheets []*Sheet) *SpreadSheet` | 构造工作簿 |
| `(*SpreadSheet).Create() error` | 渲染并保存为 xlsx |

## Sheet 字段

```go
type Sheet struct {
    name           string
    sql            string         // 仅记录，不参与渲染
    column         string         // 逗号分隔
    isSum          bool
    sumBeginColumn int            // 1-based
    data           [][]interface{}
    batchSize      int
    enableStream   bool
    rowCount       int
}
```

> 这些字段是包私有；通过构造函数和 `AddRow` 修改。

## SUM 行行为

`isSum=true` 时：

- 在最后一行写 `Total/总计`（合并首行到求和列的最后一列）
- 对 `sumBeginColumn` 到最后一列写 `=SUM(列2:列N)` 公式

示例：5 列（A-E），`sumBeginColumn=3` → 总计行写 `Total/总计` 合并 A3:E3，并对 C/D/E 写 `=SUM(C2:CN)`。

## 冻结窗格

每个 sheet 默认冻结首行（`A2` 为左上角）。当前不支持自定义；如需冻结首列或更多，v1.1 计划。

## 流式模式的限制

- 数据仍驻留在 `Sheet.data`（不会自动落盘），适用于百万行级
- 真正 O(1) 流写（`excelize.StreamWriter`）在 [ROADMAP v1.1](../../ROADMAP.md)

## 命名约束

- 工作表名：≤ 31 字符、不含 `\/:*?[]`
- 文件名：依赖 OS；Windows 不允许 `\/:*?"<>|`

## 测试

```bash
go test ./pkg/excel
```

`excel_test.go` 覆盖：空 sheet、含数据 sheet、含 SUM sheet、含冻结 sheet。

## 关联

- [`pkg/db`](../db) — 数据源
- [`internal/app`](../../internal/app) — 编排层
- [`docs/architecture.md`](../../docs/architecture.md#3-关键流程) — 调用链
