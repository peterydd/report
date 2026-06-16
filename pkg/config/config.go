/*
Package config - Configuration management with YAML support and hot-reloading.
包 config - 支持YAML配置管理和热重载。

This package provides:
- YAML configuration file parsing
- Configuration validation
- Hot-reloading support (file watching)
- Type-safe configuration structures
- Environment variable overrides for secrets / 热重载 + 环境变量覆盖

Features / 功能特性:
- Viper-based configuration management / 基于Viper的配置管理
- Config hot-reloading / 配置热重载
- Configuration validation / 配置验证
- Environment variable support / 环境变量支持

Configuration Search Paths / 配置搜索路径:
1. Current directory (./)
2. ./configs/
3. ../configs/
4. ../../configs/
5. Root directory (/)

Environment Variable Overrides / 环境变量覆盖:
- REPORT_DATABASE_DRIVER, REPORT_DATABASE_SOURCE, REPORT_DATABASE_MAXOPENCONNS,
  REPORT_DATABASE_MAXIDLECONNS, REPORT_DATABASE_CONNMAXLIFETIME
- REPORT_SMTP_HOST, REPORT_SMTP_PORT, REPORT_SMTP_USERNAME,
  REPORT_SMTP_PASSWORD, REPORT_SMTP_INSECURESKIPVERIFY, REPORT_SMTP_TIMEOUT

Use the env-var form for secrets (SMTP password) so they never land
in the on-disk config file. Non-secret values can stay in YAML.
敏感字段（SMTP 密码）应使用环境变量形式，绝不写入 yaml 文件；非敏感
值保留在 yaml 即可。
*/
package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// envPrefix is the prefix every recognised environment variable must
// carry. Keeping a single prefix prevents accidental collisions with
// unrelated env vars on the host.
// envPrefix 是所有可识别环境变量必须携带的前缀，避免与宿主机其他
// 环境变量冲突。
const envPrefix = "REPORT"

// validateWorkBookName rejects workbook prefix/suffix values that
// contain path separators or parent-directory references, blocking
// trivial path-traversal attacks against the attachment reader.
// validateWorkBookName 拒绝包含路径分隔符或父目录引用的 workbook
// 前缀/后缀值，阻止针对附件读取器的路径遍历攻击。
func validateWorkBookName(v, field string) error {
	if v == "" {
		return nil
	}
	if strings.ContainsAny(v, "/\\") {
		return fmt.Errorf("%s must not contain path separators", field)
	}
	if strings.Contains(v, "..") {
		return fmt.Errorf("%s must not contain parent directory references", field)
	}
	return nil
}

// Database holds database connection configuration.
type Database struct {
	Driver          string `mapstructure:"driver"`          // Database driver: mysql, postgresql, oracle, clickhouse
	Source          string `mapstructure:"source"`          // Data source name / connection string
	MaxOpenConns    int    `mapstructure:"maxOpenConns"`    // Maximum open connections
	MaxIdleConns    int    `mapstructure:"maxIdleConns"`    // Maximum idle connections
	ConnMaxLifetime string `mapstructure:"connMaxLifetime"` // Connection max lifetime (e.g., "3m")
}

// Smtp holds SMTP server configuration.
type Smtp struct {
	Host               string `mapstructure:"host"`               // SMTP server hostname
	Port               string `mapstructure:"port"`               // SMTP server port
	Username           string `mapstructure:"username"`           // Authentication username
	Password           string `mapstructure:"password"`           // Authentication password
	InsecureSkipVerify bool   `mapstructure:"insecureSkipVerify"` // Skip TLS certificate verification
	Timeout            string `mapstructure:"timeout"`            // Connection timeout (e.g., "30s")
}

// Reports defines a report configuration with sheets and message settings.
type Reports struct {
	Name     string    `mapstructure:"name"`     // Report name
	WorkBook *WorkBook `mapstructure:"workBook"` // Workbook settings
	Sheets   []*Sheet  `mapstructure:"sheets"`   // Sheet configurations
	Message  *Message  `mapstructure:"message"`  // Email message settings
}

// WorkBook defines Excel workbook naming conventions.
type WorkBook struct {
	Prefix     string `mapstructure:"prefix"`     // Filename prefix
	DateFormat string `mapstructure:"dateFormat"` // Date format (Go reference format)
	Suffix     string `mapstructure:"suffix"`     // Filename suffix (e.g., ".xlsx")
}

// Sheet defines a single worksheet configuration.
type Sheet struct {
	Name           string `mapstructure:"name"`           // Sheet name
	Sql            string `mapstructure:"sql"`            // SQL query to execute
	Column         string `mapstructure:"column"`         // Comma-separated column headers
	IsSum          bool   `mapstructure:"isSum"`          // Enable auto-sum calculation
	SumBeginColumn int    `mapstructure:"sumBeginColumn"` // Starting column index for summation (1-based)
	BatchSize      int    `mapstructure:"batchSize"`      // Batch size for streaming queries
	EnableStream   bool   `mapstructure:"enableStream"`   // Enable streaming query mode
}

// Message defines email message properties.
type Message struct {
	From        string      `mapstructure:"from"`        // Sender email address
	To          []string    `mapstructure:"to"`          // Primary recipients
	Cc          []string    `mapstructure:"cc"`          // CC recipients
	Bcc         []string    `mapstructure:"bcc"`         // BCC recipients
	Subject     string      `mapstructure:"subject"`     // Email subject
	Body        string      `mapstructure:"body"`        // Email body content
	ContentType string      `mapstructure:"contentType"` // Content type (e.g., "text/html")
	Attachment  *Attachment `mapstructure:"attachment"`  // Attachment configuration
}

// Attachment defines email attachment settings.
type Attachment struct {
	ContentType string `mapstructure:"contentType"` // MIME content type
	WithFile    bool   `mapstructure:"withFile"`    // Enable attachment
}

// Config is the root configuration structure.
type Config struct {
	Database *Database     `mapstructure:"database"` // Database configuration
	Smtp     *Smtp         `mapstructure:"smtp"`     // SMTP configuration
	Reports  []*Reports    `mapstructure:"reports"`  // Report configurations
	mu       sync.RWMutex  // Mutex for thread-safe access
	onChange func(*Config) // Callback for config changes
}

// SetOnChange registers a callback function that will be called when configuration changes.
func (c *Config) SetOnChange(fn func(*Config)) {
	c.mu.Lock()
	c.onChange = fn
	c.mu.Unlock()
}

// NotifyChange triggers the registered callback if configuration has changed.
func (c *Config) NotifyChange() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.onChange != nil {
		c.onChange(c)
	}
}

// Clone creates a thread-safe read-only copy of the configuration.
func (c *Config) Clone() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &Config{
		Database: c.Database,
		Smtp:     c.Smtp,
		Reports:  c.Reports,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Database == nil {
		return fmt.Errorf("database configuration is required")
	}
	if c.Database.Driver == "" {
		return fmt.Errorf("database.driver is required")
	}
	if c.Database.Source == "" {
		return fmt.Errorf("database.source is required")
	}
	if c.Smtp == nil {
		return fmt.Errorf("smtp configuration is required")
	}
	if c.Smtp.Host == "" {
		return fmt.Errorf("smtp.host is required")
	}
	if c.Smtp.Port == "" {
		return fmt.Errorf("smtp.port is required")
	}
	if c.Smtp.Username == "" {
		return fmt.Errorf("smtp.username is required")
	}
	if c.Smtp.Password == "" {
		return fmt.Errorf("smtp.password is required")
	}
	if len(c.Reports) == 0 {
		return fmt.Errorf("at least one report configuration is required")
	}
	for i, rp := range c.Reports {
		if rp.Name == "" {
			return fmt.Errorf("reports[%d].name is required", i)
		}
		if len(rp.Sheets) == 0 {
			return fmt.Errorf("reports[%d].sheets cannot be empty", i)
		}
		if rp.WorkBook != nil {
			// Defence against path traversal: refuse prefix/suffix
			// that try to break out of the working directory.
			if err := validateWorkBookName(rp.WorkBook.Prefix, "prefix"); err != nil {
				return fmt.Errorf("reports[%d].workBook.%w", i, err)
			}
			if err := validateWorkBookName(rp.WorkBook.Suffix, "suffix"); err != nil {
				return fmt.Errorf("reports[%d].workBook.%w", i, err)
			}
		}
		if rp.Message == nil {
			return fmt.Errorf("reports[%d].message is required", i)
		}
		if rp.Message.From == "" {
			return fmt.Errorf("reports[%d].message.from is required", i)
		}
		if len(rp.Message.To) == 0 {
			return fmt.Errorf("reports[%d].message.to is required", i)
		}
		if rp.Message.Subject == "" {
			return fmt.Errorf("reports[%d].message.subject is required", i)
		}
	}
	return nil
}

// NewConfig creates a new configuration instance by reading the config.yaml file.
// Supports automatic configuration hot-reloading when the file changes.
func NewConfig() (conf *Config, err error) {
	return NewConfigFromPath("")
}

// NewConfigFromPath is like NewConfig but loads config.yaml from the supplied
// directory. When path is empty the default search list is used. Hot-reload
// is disabled in the explicit-path form to avoid leaking viper state across
// tests; the production hot-reload path remains untouched.
// NewConfigFromPath 与 NewConfig 类似，但从指定目录加载 config.yaml。
// path 为空时使用默认搜索列表；显式路径模式关闭热重载，避免 viper
// 状态在测试间泄漏。生产热重载路径不受影响。
func NewConfigFromPath(path string) (conf *Config, err error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	if path != "" {
		v.AddConfigPath(path)
	} else {
		v.AddConfigPath("/")
		v.AddConfigPath("./")
		v.AddConfigPath("./configs")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")
	}

	err = v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	bindEnvOverrides(v)

	conf = &Config{}
	err = v.Unmarshal(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	if err = conf.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	if path == "" {
		v.WatchConfig()
		v.OnConfigChange(func(e fsnotify.Event) {
			newConf := &Config{}
			if err := v.Unmarshal(newConf); err == nil {
				if err := newConf.Validate(); err == nil {
					conf.mu.Lock()
					conf.Database = newConf.Database
					conf.Smtp = newConf.Smtp
					conf.Reports = newConf.Reports
					conf.mu.Unlock()
					conf.NotifyChange()
				}
			}
		})
	}

	return conf, nil
}

// bindEnvOverrides wires REPORT_* environment variables into viper so
// secrets (notably SMTP password) can be supplied without baking them
// into config.yaml. Environment values take precedence over file values.
// bindEnvOverrides 把 REPORT_* 环境变量绑定到 viper，使敏感字段
// （特别是 SMTP 密码）可以不写入 config.yaml 而通过环境变量提供。
// 环境变量优先级高于 yaml。
func bindEnvOverrides(v *viper.Viper) {
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	// viper's AutomaticEnv only matches the full key, so allow dotted
	// paths via a replacer (REPORT_SMTP_PASSWORD -> smtp.password).
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Explicit BindEnv for the secret-bearing fields; this also
	// documents the supported env var names.
	envKeys := []string{
		"database.driver",
		"database.source",
		"database.maxOpenConns",
		"database.maxIdleConns",
		"database.connMaxLifetime",
		"smtp.host",
		"smtp.port",
		"smtp.username",
		"smtp.password",
		"smtp.insecureSkipVerify",
		"smtp.timeout",
	}
	for _, key := range envKeys {
		_ = v.BindEnv(key)
	}
}

// Backward compatibility aliases for lowercase types
// Deprecated: Use uppercase types instead
type (
	database = Database
	smtp     = Smtp
	reports  = Reports
	workBook = WorkBook
	sheet    = Sheet
	message  = Message
	attach   = Attachment
)
