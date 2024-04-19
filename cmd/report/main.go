package main

import (
	"github.com/peterydd/report/internal/report"
	"log"
)

func main() {
	r := report.NewReport()
	if err := r.Start(); err != nil {
		log.Fatal(err)
	}
	log.Println("报表生成和发送成功")
}
