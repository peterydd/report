package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestNewConfig(t *testing.T) {
	if _, err := os.Stat("config.yaml"); err != nil {
		t.Skip("skipping: config.yaml not present in test working directory")
	}
	conf, err := NewConfig()
	if err != nil {
		spew.Dump(err)
	}
	spew.Dump(conf)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "完整配置",
			config: &Config{
				Database: &database{
					Driver: "mysql",
					Source: "user:pass@tcp(localhost:3306)/db",
				},
				Smtp: &smtp{
					Host:     "smtp.example.com",
					Port:     "587",
					Username: "test@example.com",
					Password: "password",
				},
				Reports: []*reports{
					{
						Name: "测试报表",
						Sheets: []*sheet{
							{
								Name:   "Sheet1",
								Sql:    "SELECT * FROM test",
								Column: "col1,col2",
							},
						},
						Message: &message{
							From:    "from@example.com",
							To:      []string{"to@example.com"},
							Subject: "测试",
							Body:    "测试内容",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "缺少database配置",
			config: &Config{
				Database: nil,
				Smtp: &smtp{
					Host:     "smtp.example.com",
					Port:     "587",
					Username: "test@example.com",
					Password: "password",
				},
				Reports: []*reports{{}},
			},
			wantErr: true,
		},
		{
			name: "缺少database.driver",
			config: &Config{
				Database: &database{
					Driver: "",
					Source: "user:pass@tcp(localhost:3306)/db",
				},
				Smtp: &smtp{
					Host:     "smtp.example.com",
					Port:     "587",
					Username: "test@example.com",
					Password: "password",
				},
				Reports: []*reports{{}},
			},
			wantErr: true,
		},
		{
			name: "缺少database.source",
			config: &Config{
				Database: &database{
					Driver: "mysql",
					Source: "",
				},
				Smtp: &smtp{
					Host:     "smtp.example.com",
					Port:     "587",
					Username: "test@example.com",
					Password: "password",
				},
				Reports: []*reports{{}},
			},
			wantErr: true,
		},
		{
			name: "缺少smtp配置",
			config: &Config{
				Database: &database{
					Driver: "mysql",
					Source: "user:pass@tcp(localhost:3306)/db",
				},
				Smtp:    nil,
				Reports: []*reports{{}},
			},
			wantErr: true,
		},
		{
			name: "缺少smtp.host",
			config: &Config{
				Database: &database{
					Driver: "mysql",
					Source: "user:pass@tcp(localhost:3306)/db",
				},
				Smtp: &smtp{
					Host:     "",
					Port:     "587",
					Username: "test@example.com",
					Password: "password",
				},
				Reports: []*reports{{}},
			},
			wantErr: true,
		},
		{
			name: "缺少reports配置",
			config: &Config{
				Database: &database{
					Driver: "mysql",
					Source: "user:pass@tcp(localhost:3306)/db",
				},
				Smtp: &smtp{
					Host:     "smtp.example.com",
					Port:     "587",
					Username: "test@example.com",
					Password: "password",
				},
				Reports: []*reports{},
			},
			wantErr: true,
		},
		{
			name: "缺少report.name",
			config: &Config{
				Database: &database{
					Driver: "mysql",
					Source: "user:pass@tcp(localhost:3306)/db",
				},
				Smtp: &smtp{
					Host:     "smtp.example.com",
					Port:     "587",
					Username: "test@example.com",
					Password: "password",
				},
				Reports: []*reports{
					{
						Name: "",
						Sheets: []*sheet{
							{Name: "Sheet1", Sql: "SELECT *", Column: "col1"},
						},
						Message: &message{
							From:    "from@example.com",
							To:      []string{"to@example.com"},
							Subject: "测试",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "缺少report.sheets",
			config: &Config{
				Database: &database{
					Driver: "mysql",
					Source: "user:pass@tcp(localhost:3306)/db",
				},
				Smtp: &smtp{
					Host:     "smtp.example.com",
					Port:     "587",
					Username: "test@example.com",
					Password: "password",
				},
				Reports: []*reports{
					{
						Name:   "测试报表",
						Sheets: []*sheet{},
						Message: &message{
							From:    "from@example.com",
							To:      []string{"to@example.com"},
							Subject: "测试",
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfigValidate_WorkBookPathTraversal ensures workbook prefix
// and suffix cannot smuggle path separators or parent-directory
// references that would let the attachment reader escape the working
// directory.
// TestConfigValidate_WorkBookPathTraversal 确保 workbook 前缀与后缀
// 不能夹带路径分隔符或父目录引用，从而阻止附件读取器逃逸工作目录。
func TestConfigValidate_WorkBookPathTraversal(t *testing.T) {
	base := func() *Config {
		return &Config{
			Database: &database{Driver: "mysql", Source: "x"},
			Smtp: &smtp{
				Host: "h", Port: "25", Username: "u", Password: "p",
			},
			Reports: []*reports{
				{
					Name: "r",
					WorkBook: &workBook{
						Prefix:     "ok_",
						DateFormat: "20060102",
						Suffix:     ".xlsx",
					},
					Sheets: []*sheet{{Name: "S", Sql: "SELECT 1", Column: "c"}},
					Message: &message{
						From:    "f@x",
						To:      []string{"t@x"},
						Subject: "s",
					},
				},
			},
		}
	}

	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{"valid prefix/suffix", func(*Config) {}, false},
		{"slash in prefix", func(c *Config) { c.Reports[0].WorkBook.Prefix = "../" }, true},
		{"backslash in prefix", func(c *Config) { c.Reports[0].WorkBook.Prefix = `..\` }, true},
		{"absolute path in suffix", func(c *Config) { c.Reports[0].WorkBook.Suffix = "/etc/passwd" }, true},
		{"parent reference in suffix", func(c *Config) { c.Reports[0].WorkBook.Suffix = "..xlsx" }, true},
		{"empty prefix and suffix allowed", func(c *Config) {
			c.Reports[0].WorkBook.Prefix = ""
			c.Reports[0].WorkBook.Suffix = ""
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := base()
			tt.mutate(c)
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewConfigFromPath_EnvOverridesSMTP ensures REPORT_SMTP_PASSWORD
// and friends override values loaded from config.yaml so secrets can
// be supplied without baking them into the file on disk.
// TestNewConfigFromPath_EnvOverridesSMTP 验证 REPORT_SMTP_PASSWORD 等
// 环境变量覆盖 config.yaml 的值，使敏感字段可不写入磁盘文件。
func TestNewConfigFromPath_EnvOverridesSMTP(t *testing.T) {
	dir := t.TempDir()
	yaml := []byte(`
database:
  driver: mysql
  source: "user:pw@tcp(127.0.0.1:3306)/db"
  maxOpenConns: 5
  maxIdleConns: 2
  connMaxLifetime: "1m"
smtp:
  host: "smtp.yaml.example"
  port: "587"
  username: "yaml-user"
  password: "yaml-pass"
  insecureSkipVerify: false
  timeout: "10s"
reports:
  - name: "r1"
    workBook:
      prefix: "p_"
      dateFormat: "20060102"
      suffix: ".xlsx"
    sheets:
      - name: "S"
        sql: "SELECT 1"
        column: "c"
    message:
      from: "f@x"
      to: ["t@x"]
      subject: "subj"
`)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), yaml, 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	t.Setenv("REPORT_SMTP_HOST", "smtp.env.example")
	t.Setenv("REPORT_SMTP_PASSWORD", "env-pass")
	t.Setenv("REPORT_SMTP_INSECURESKIPVERIFY", "true")

	conf, err := NewConfigFromPath(dir)
	if err != nil {
		t.Fatalf("NewConfigFromPath: %v", err)
	}
	if conf.Smtp.Host != "smtp.env.example" {
		t.Errorf("smtp.host = %q, want env override", conf.Smtp.Host)
	}
	if conf.Smtp.Password != "env-pass" {
		t.Errorf("smtp.password = %q, want env override", conf.Smtp.Password)
	}
	if !conf.Smtp.InsecureSkipVerify {
		t.Error("smtp.insecureSkipVerify should be true from env")
	}
	// Untouched field still comes from yaml.
	if conf.Smtp.Username != "yaml-user" {
		t.Errorf("smtp.username = %q, want yaml value", conf.Smtp.Username)
	}
}
