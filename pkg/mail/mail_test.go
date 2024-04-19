package mail

import (
	"testing"
)

func TestSend(t *testing.T) {
	sm := NewSendMail("smtp.example.com", "25", "test@example.com", "password")
	sm.Send(&Message{
		from:        "test@example.com",
		to:          []string{"test@outlook.com", "test@qq.com"},
		cc:          []string{"test@gmail.com"},
		bcc:         []string{"test@189.cn"},
		subject:     "test主题",
		body:        "test正文",
		contentType: "text/plain;charset=utf-8",
		attachment: &Attach{
			name:        "../excel/test.xlsx",
			contentType: "text/plain;charset=utf-8",
			withFile:    true,
		},
	})
}
