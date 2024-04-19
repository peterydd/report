package report

import (
	"github.com/peterydd/report/pkg/config"
	"github.com/peterydd/report/pkg/db"
	"github.com/peterydd/report/pkg/excel"
	"github.com/peterydd/report/pkg/mail"
	"log"
	"time"
)

type Report struct {
	*config.Config
}

func NewReport() *Report {
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}
	log.Printf("读取配置文件成功！\n")
	return &Report{conf}
}

func (r *Report) Start() error {
	// 连接数据库
	t := db.DBType(r.Database.Driver)
	db := db.NewDB(t)
	if err := db.Connect(r.Database.Source); err != nil {
		return err
	}
	// 断开数据库连接
	defer db.Close()

	// 生成报表
	for _, rp := range r.Reports {
		sts := []*excel.Sheet{}
		for _, st := range rp.Sheets {
			data, err := db.Query(st.Sql)
			if err != nil {
				return err
			}
			log.Println(data)
			sts = append(sts, excel.SetSheet(st.Name, st.Sql, st.Column, data))

		}
		bookName := rp.WorkBook.Prefix + time.Now().Format(rp.WorkBook.DateFormat) + rp.WorkBook.Suffix
		sp := excel.NewSpreadSheet(bookName, sts)
		if err := sp.Create(); err != nil {
			return err
		}
		log.Printf("生成报表 %s 成功！\n", bookName)

		// 发送邮件
		attachment := mail.SetAttach(bookName, rp.Message.Attachment.ContentType, rp.Message.Attachment.WithFile)
		message := mail.SetMessage(rp.Message.From, rp.Message.To, rp.Message.Cc, rp.Message.Bcc, rp.Message.Subject, rp.Message.Body, rp.Message.ContentType, attachment)
		sm := mail.NewSendMail(r.Smtp.Host, r.Smtp.Port, r.Smtp.Username, r.Smtp.Password)
		log.Println(message)
		if err := sm.Send(message); err != nil {
			return err
		}
		log.Printf("发送邮件 %s 成功！\n", rp.Message.Subject)
	}
	return nil

}
