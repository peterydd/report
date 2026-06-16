# pkg/excel

<p align="center">
  <a href="README.md">中文</a> | <a href="README.en.md">English</a>
</p>

Excel report generation built on [excelize/v2](https://github.com/xuri/excelize): multi-sheet, auto-sum, frozen header, streaming writes.

> Project home: [README](../../README.md) · Architecture: [docs/architecture.md](../../docs/architecture.md)

## Features

- ✅ Multi-sheet workbooks
- ✅ Custom headers (comma-separated)
- ✅ Auto SUM formulas (`isSum=true`)
- ✅ Frozen first row (`Freeze: true, YSplit: 1`)
- ✅ Streaming sheets via `SetSheetStream` + `AddRow`
- ✅ Unicode-friendly file/column names

## Install

```bash
go get github.com/peterydd/report/pkg/excel
```

## Quick Start

```go
import (
    "log"
    "github.com/peterydd/report/pkg/excel"
)

// One-shot sheet
data := [][]interface{}{
    {"2024-01-01", 100},
    {"2024-01-02", 200},
}
sheet := excel.SetSheet("Sales", "SELECT ...", "Date,Amount", true, 2, data)

// Streaming sheet
stream := excel.SetSheetStream("Logs", "SELECT ...", "TS,Level,Msg", false, 0, 10000)
stream.AddRow([]interface{}{"2024-01-01 10:00", "INFO", "started"})
stream.AddRow([]interface{}{"2024-01-01 10:01", "INFO", "ready"})

// Render
spreadsheet := excel.NewSpreadSheet("report.xlsx", []*excel.Sheet{sheet, stream})
if err := spreadsheet.Create(); err != nil {
    log.Fatal(err)
}
```

## Exported API

| Symbol | Description |
|--------|-------------|
| `SetSheet(name, sql, column string, isSum bool, sumBeginCol int, data [][]interface{}) *Sheet` | Build a sheet with full data |
| `SetSheetStream(name, sql, column string, isSum bool, sumBeginCol, batchSize int) *Sheet` | Build a streaming sheet (default batchSize 10000) |
| `(*Sheet).AddRow(row []interface{})` | Append a row (streaming) |
| `(*Sheet).GetRowCount() int` | Data-row count (no header) |
| `(*Sheet).ClearData()` | Drop in-memory data |
| `NewSpreadSheet(name string, sheets []*Sheet) *SpreadSheet` | Build a workbook |
| `(*SpreadSheet).Create() error` | Render and save as xlsx |

## Sheet Fields

```go
type Sheet struct {
    name           string
    sql            string
    column         string
    isSum          bool
    sumBeginColumn int    // 1-based
    data           [][]interface{}
    batchSize      int
    enableStream   bool
    rowCount       int
}
```

> Unexported; use the constructors and `AddRow`.

## SUM Row Behavior

When `isSum=true`:

- Last row shows `Total/总计` (merged across A → last summed column)
- Each cell from `sumBeginColumn` to the last column gets `=SUM(Col2:ColN)`

Example: 5 columns (A–E), `sumBeginColumn=3` → merges A:E on the total row and writes `SUM` in C/D/E.

## Frozen Panes

Every sheet freezes the first row (top-left cell `A2`). Customization (freeze first column, multiple rows) is on the [ROADMAP](../../ROADMAP.md) for v1.1.

## Streaming Caveats

- Rows are still accumulated in `Sheet.data`; not flushed to disk row-by-row
- True O(1) streaming via `excelize.StreamWriter` is planned for v1.1

## Naming Constraints

- Sheet name: ≤ 31 chars, no `\/:*?[]`
- File name: subject to OS rules (Windows forbids `\/:*?"<>|`)

## Testing

```bash
go test ./pkg/excel
```

`excel_test.go` covers: empty sheet, with data, with SUM, with frozen pane.

## Related

- [`pkg/db`](../db) — data source
- [`internal/app`](../../internal/app) — orchestration
- [`docs/architecture.md`](../../docs/architecture.md#3-关键流程) — call chain
