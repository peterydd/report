package excel

import (
	"github.com/xuri/excelize/v2"
	"log"
	"strings"
)

type conf struct {
	name   string
	sheets []*Sheet
}

type Sheet struct {
	name           string
	sql            string
	column         string
	isSum          bool
	sumBeginColumn int
	data           [][]interface{}
}

type SpreadSheet struct {
	*conf
}

func NewSpreadSheet(name string, sheets []*Sheet) *SpreadSheet {
	return &SpreadSheet{
		conf: &conf{
			name:   name,
			sheets: sheets,
		},
	}
}

func SetSheet(name, sql, column string, isSum bool, sumBeginColumn int, data [][]interface{}) *Sheet {
	return &Sheet{
		name:           name,
		sql:            sql,
		column:         column,
		isSum:          isSum,
		sumBeginColumn: sumBeginColumn,
		data:           data,
	}
}

func (s *SpreadSheet) Create() error {
	// 创建excel文件
	f := excelize.NewFile()
	// 关闭文件
	defer func() {
		if err := f.Close(); err != nil {
			log.Println(err)
		}
	}()
	// 创建sheet
	for _, st := range s.sheets {
		_, err := f.NewSheet(st.name)
		if err != nil {
			log.Fatal(err)
		}
		// 设置表头
		columns := strings.Split(st.column, ",")
		if err := f.SetSheetRow(st.name, "A1", &columns); err != nil {
			log.Fatal(err)
		}

		for i, row := range st.data {
			startCell, err := excelize.JoinCellName("A", i+2)
			if err != nil {
				log.Fatal(err)
			}
			if err := f.SetSheetRow(st.name, startCell, &row); err != nil {
				log.Fatal(err)
			}
		}

		// 自动求和
		if st.isSum {
			columnName, err := excelize.ColumnNumberToName(st.sumBeginColumn - 1)
			if err != nil {
				log.Fatal(err)
			}
			startCell, err := excelize.JoinCellName("A", len(st.data)+2)
			if err != nil {
				log.Fatal(err)
			}
			endCell, err := excelize.JoinCellName(columnName, len(st.data)+2)
			if err != nil {
				log.Fatal(err)
			}
			err = f.MergeCell(st.name, startCell, endCell)
			if err != nil {
				log.Fatal(err)
			}
			err = f.SetCellValue(st.name, startCell, "总计")
			if err != nil {
				log.Fatal(err)
			}
			for i := st.sumBeginColumn; i <= len(columns); i++ {
				columnName, err = excelize.ColumnNumberToName(i)
				if err != nil {
					log.Fatal(err)
				}
				sumCell, err := excelize.JoinCellName(columnName, len(st.data)+2)
				if err != nil {
					log.Fatal(err)
				}
				startCell, err := excelize.JoinCellName(columnName, 2)
				if err != nil {
					log.Fatal(err)
				}
				endCell, err := excelize.JoinCellName(columnName, len(st.data)+1)
				if err != nil {
					log.Fatal(err)
				}

				err = f.SetCellFormula(st.name, sumCell, "SUM("+startCell+":"+endCell+")")
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		// 设置窗格（表头固定,冻结行）
		if err := f.SetPanes(st.name,
			&excelize.Panes{
				Freeze:      true,
				Split:       false,
				XSplit:      0,
				YSplit:      1,
				TopLeftCell: "A2",
				ActivePane:  "bottomLeft",
			}); err != nil {
			log.Fatal(err)
		}
	}

	// 删除默认sheet1
	if err := f.DeleteSheet("Sheet1"); err != nil {
		log.Fatal(err)
	}
	// 保存文件
	if err := f.SaveAs(s.name); err != nil {
		log.Fatal(err)
	}

	return nil
}
