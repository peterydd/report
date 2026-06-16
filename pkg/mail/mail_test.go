package mail

import (
	"os"
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	if os.Getenv("REPORT_INTEGRATION") == "" {
		t.Skip("skipping integration test: set REPORT_INTEGRATION=1 to send a real email")
	}
	sm := NewSendMail("smtp.example.com", "25", "test@example.com", "password", false, 30*time.Second)
	err := sm.Send(&Message{
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
	if err != nil {
		t.Errorf("发送邮件失败: %v", err)
	}
}

// TestSanitizeHeaderValue verifies CR/LF injection is stripped from a
// single header value so attackers controlling the config cannot
// inject extra SMTP commands.
// TestSanitizeHeaderValue 验证 CR/LF 注入会从单个 header 值中剥离，
// 防止控制配置的攻击者注入额外 SMTP 命令。
func TestSanitizeHeaderValue(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"with\r\nBcc: attacker@evil", "withBcc: attacker@evil"},
		{"line1\nline2", "line1line2"},
		{"line1\rline2", "line1line2"},
		{"\r\nBcc: x", "Bcc: x"},
	}
	for _, tt := range tests {
		got := sanitizeHeaderValue(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeHeaderValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestSanitizeHeaders verifies the in-place sanitiser removes CR/LF
// from every value in the header map.
// TestSanitizeHeaders 验证原地清理器会从 header map 的每个值中
// 剥离 CR/LF。
func TestSanitizeHeaders(t *testing.T) {
	h := map[string]string{
		"From":    "f@x",
		"To":      "t@x\r\nBcc: evil@x",
		"Subject": "hello\nbye",
	}
	sanitizeHeaders(h)
	if h["To"] != "t@xBcc: evil@x" {
		t.Errorf("To header not sanitised: %q", h["To"])
	}
	if h["Subject"] != "hellobye" {
		t.Errorf("Subject header not sanitised: %q", h["Subject"])
	}
	if h["From"] != "f@x" {
		t.Errorf("From header changed unexpectedly: %q", h["From"])
	}
}
