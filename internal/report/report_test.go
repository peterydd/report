package report

import (
	"github.com/davecgh/go-spew/spew"
	"testing"
)

func TestReport(t *testing.T) {
	r := NewReport()
	if err := r.Start(); err != nil {
		spew.Dump(err)
	}
}
