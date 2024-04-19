package excel

import (
	"github.com/davecgh/go-spew/spew"
	"testing"
)

func TestSpreadSheet(t *testing.T) {
	data1 := make([][]interface{}, 0)
	data1 = append(data1, []interface{}{"bundle_001", 100.01, 200.01, 21, 1024.01})
	data1 = append(data1, []interface{}{"bundle_002", 100.02, 200.02, 22, 1024.02})
	data1 = append(data1, []interface{}{"bundle_003", 100.03, 200.03, 23, 1024.03})
	data1 = append(data1, []interface{}{"bundle_004", 100.04, 200.04, 24, 1024.04})

	data2 := make([][]interface{}, 0)
	data2 = append(data2, []interface{}{"bundle_005", 100.05, 200.05, 25, 1024.05})
	data2 = append(data2, []interface{}{"bundle_006", 100.06, 200.06, 26, 1024.06})
	data2 = append(data2, []interface{}{"bundle_007", 100.07, 200.07, 27, 1024.07})
	data2 = append(data2, []interface{}{"bundle_008", 100.08, 200.08, 28, 1024.08})

	sts := []*Sheet{
		{
			name:   "sheet页1",
			sql:    "select col1,col2,col3,col4,col5 from table1",
			column: "字段1,字段2,字段3,字段4,字段5",
			data:   data1,
		},
		{
			name:   "sheet页2",
			sql:    "select col1,col2,col3,col4,col5 from table2",
			column: "字段1,字段2,字段3,字段4,字段5",
			data:   data2,
		},
	}

	sp := NewSpreadSheet("test.xlsx", sts)
	if err := sp.Create(); err != nil {
		spew.Dump(err)
	}

}
