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
	name   string
	sql    string
	column string
	data   [][]interface{}
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

func SetSheet(name, sql, column string, data [][]interface{}) *Sheet {
	return &Sheet{
		name:   name,
		sql:    sql,
		column: column,
		data:   data,
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
