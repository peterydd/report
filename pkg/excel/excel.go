/*
Package excel - Excel file generation and manipulation.
包 excel - Excel文件生成和操作。

This package provides functionality for:
- Creating Excel files with multiple sheets
- Setting headers and data rows
- Formulas and auto-sum calculations
- Frozen pane support for large datasets
- Streaming mode for memory-efficient processing

Features / 功能特性:
- Multiple sheets per workbook / 每个工作簿支持多个工作表
- Custom column headers / 自定义列标题
- Auto-sum calculations / 自动求和计算
- Row freeze (header freeze) / 行冻结（表头冻结）
- Streaming mode for large data / 大数据流式处理模式
*/
package excel

import (
	"fmt"
	"log"
	"strings"

	"github.com/xuri/excelize/v2"
)

// conf represents a workbook configuration.
type conf struct {
	name   string   // Workbook name / 工作簿名称
	sheets []*Sheet // Sheets in the workbook / 工作簿中的工作表
}

// Sheet represents a single worksheet with data and configuration.
type Sheet struct {
	name           string          // Sheet name / 工作表名称
	sql            string          // SQL query (for reference) / SQL查询（用于参考）
	column         string          // Column headers comma-separated / 逗号分隔的列标题
	isSum          bool            // Enable auto-sum feature / 启用自动求和
	sumBeginColumn int             // Starting column for summation / 开始求和的列
	data           [][]interface{} // Sheet data / 工作表数据
	batchSize      int             // Batch size for streaming mode / 流式处理批次大小
	enableStream   bool            // Enable streaming mode / 启用流式处理
	rowCount       int             // Row count in streaming mode / 流式模式下的行数
}

// SpreadSheet represents a complete Excel workbook.
type SpreadSheet struct {
	*conf
}

// NewSpreadSheet creates a new spreadsheet with the specified sheets.
func NewSpreadSheet(name string, sheets []*Sheet) *SpreadSheet {
	return &SpreadSheet{
		conf: &conf{
			name:   name,
			sheets: sheets,
		},
	}
}

// SetSheet creates a new sheet with pre-loaded data.
// This is suitable for small to medium datasets.
func SetSheet(name, sql, column string, isSum bool, sumBeginColumn int, data [][]interface{}) *Sheet {
	return &Sheet{
		name:           name,
		sql:            sql,
		column:         column,
		isSum:          isSum,
		sumBeginColumn: sumBeginColumn,
		data:           data,
		batchSize:      10000,
		enableStream:   false,
	}
}

// SetSheetStream creates a new sheet with streaming support.
// This is suitable for large datasets where data is added incrementally.
func SetSheetStream(name, sql, column string, isSum bool, sumBeginColumn int, batchSize int) *Sheet {
	if batchSize <= 0 {
		batchSize = 10000
	}
	return &Sheet{
		name:           name,
		sql:            sql,
		column:         column,
		isSum:          isSum,
		sumBeginColumn: sumBeginColumn,
		data:           make([][]interface{}, 0),
		batchSize:      batchSize,
		enableStream:   true,
		rowCount:       0,
	}
}

// AddRow adds a row of data to the sheet (streaming mode).
func (s *Sheet) AddRow(row []interface{}) {
	s.data = append(s.data, row)
	s.rowCount++
}

// GetRowCount returns the number of data rows in the sheet.
func (s *Sheet) GetRowCount() int {
	if s.enableStream {
		return s.rowCount
	}
	return len(s.data)
}

// ClearData clears all data from the sheet (for memory optimization).
func (s *Sheet) ClearData() {
	s.data = make([][]interface{}, 0)
}

// Create generates the Excel file with all sheets and their data.
func (s *SpreadSheet) Create() error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close Excel file: %v", err)
		}
	}()

	for _, st := range s.sheets {
		_, err := f.NewSheet(st.name)
		if err != nil {
			return fmt.Errorf("failed to create worksheet %s: %w", st.name, err)
		}

		columns := strings.Split(st.column, ",")
		if err := f.SetSheetRow(st.name, "A1", &columns); err != nil {
			return fmt.Errorf("failed to set header row: %w", err)
		}

		rowCount := len(st.data)
		startRow := 2

		for i := 0; i < rowCount; i++ {
			startCell, _ := excelize.JoinCellName("A", startRow+i)
			if err := f.SetSheetRow(st.name, startCell, &st.data[i]); err != nil {
				return fmt.Errorf("failed to write data row: %w", err)
			}
		}

		if st.isSum {
			totalRow := rowCount + 2
			sumColumnCount := len(columns) - st.sumBeginColumn + 1
			columnNames := make([]string, sumColumnCount)
			for i := 0; i < sumColumnCount; i++ {
				columnNames[i], _ = excelize.ColumnNumberToName(st.sumBeginColumn + i)
			}

			startCell, _ := excelize.JoinCellName("A", totalRow)
			endCell, _ := excelize.JoinCellName(columnNames[len(columnNames)-1], totalRow)
			_ = f.MergeCell(st.name, startCell, endCell)
			_ = f.SetCellValue(st.name, startCell, "Total/总计")

			for _, colName := range columnNames {
				sumCell, _ := excelize.JoinCellName(colName, totalRow)
				startCell, _ := excelize.JoinCellName(colName, 2)
				endCell, _ := excelize.JoinCellName(colName, rowCount+1)
				_ = f.SetCellFormula(st.name, sumCell, "SUM("+startCell+":"+endCell+")")
			}
		}

		if err := f.SetPanes(st.name,
			&excelize.Panes{
				Freeze:      true,
				Split:       false,
				XSplit:      0,
				YSplit:      1,
				TopLeftCell: "A2",
				ActivePane:  "bottomLeft",
			}); err != nil {
			return fmt.Errorf("failed to set frozen panes: %w", err)
		}
	}

	if err := f.DeleteSheet("Sheet1"); err != nil {
		return fmt.Errorf("failed to delete default sheet: %w", err)
	}
	if err := f.SaveAs(s.name); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}
