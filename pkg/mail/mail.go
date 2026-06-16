/*
Package mail - Email sending functionality with SMTP support.
提供邮件发送功能，支持SMTP协议。

This package implements email sending with:
- SMTP authentication (PLAIN mechanism)
- TLS/STARTTLS encryption
- Multipart MIME messages
- Attachment support with base64 encoding
- CC/BCC recipient handling

Features / 功能特性:
- Support for multiple recipients (To, CC, BCC)
- HTML and plain text email content types
- File attachment with automatic base64 encoding
- Configurable connection timeout
- TLS certificate verification control
*/
package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Mail interface defines the contract for email services.
// 定义邮件服务的接口规范。
type Mail interface {
	// Auth performs SMTP authentication.
	// 执行SMTP认证。
	Auth()
	// Send sends an email message.
	// 发送邮件消息。
	Send(m *Message) error
}

// smtpConf holds SMTP connection configuration.
// SMTP连接配置。
type smtpConf struct {
	host               string        // SMTP server hostname / SMTP服务器地址
	port               string        // SMTP server port / SMTP服务器端口
	username           string        // Authentication username / 认证用户名
	password           string        // Authentication password / 认证密码
	insecureSkipVerify bool          // Skip TLS certificate verification / 跳过TLS证书验证
	timeout            time.Duration // Connection timeout / 连接超时时间
}

// Message represents an email message with recipients, content, and optional attachments.
// 邮件消息结构，包含收件人、内容和可选附件。
type Message struct {
	from        string   // Sender email address / 发件人地址
	to          []string // Primary recipients / 主要收件人
	cc          []string // Carbon copy recipients / 抄送收件人
	bcc         []string // Blind carbon copy recipients / 密送收件人
	subject     string   // Email subject line / 邮件主题
	body        string   // Email body content / 邮件正文
	contentType string   // MIME content type (e.g., "text/html", "text/plain") / MIME内容类型
	attachment  *Attach  // Optional attachment / 可选附件
}

// SetMessage creates a new email message with specified parameters.
// 创建邮件消息。
func SetMessage(from string, to, cc, bcc []string, subject, body, contentType string, attachment *Attach) *Message {
	return &Message{
		from:        from,
		to:          to,
		cc:          cc,
		bcc:         bcc,
		subject:     subject,
		body:        body,
		contentType: contentType,
		attachment:  attachment,
	}
}

// Attach represents an email attachment.
// 邮件附件。
type Attach struct {
	name        string // Filename as displayed in email / 邮件中显示的文件名
	contentType string // MIME content type of the file / 文件的MIME类型
	withFile    bool   // Whether attachment has file content / 是否有文件内容
}

// SetAttach creates a new attachment configuration.
// 创建附件配置。
func SetAttach(name, contentType string, withFile bool) *Attach {
	return &Attach{
		name:        name,
		contentType: contentType,
		withFile:    withFile,
	}
}

// SendMail implements the Mail interface for SMTP email sending.
// SendMail实现Mail接口，提供SMTP邮件发送功能。
type SendMail struct {
	*smtpConf
	auth smtp.Auth // SMTP authentication credentials / SMTP认证信息
}

// NewSendMail creates a new SendMail instance with the specified configuration.
// 创建SendMail实例。
func NewSendMail(host, port, username, password string, insecureSkipVerify bool, timeout time.Duration) *SendMail {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &SendMail{
		smtpConf: &smtpConf{
			host:               host,
			port:               port,
			username:           username,
			password:           password,
			insecureSkipVerify: insecureSkipVerify,
			timeout:            timeout,
		},
	}
}

// Auth performs SMTP authentication using PLAIN mechanism.
// 使用PLAIN机制执行SMTP认证。
func (sm *SendMail) Auth() {
	if sm.insecureSkipVerify {
		log.Printf("WARNING: smtp.InsecureSkipVerify=true; TLS certificate verification is disabled (MITM risk).")
	}
	sm.auth = smtp.PlainAuth("", sm.username, sm.password, sm.host)
}

// Send sends an email message with the specified recipients and content.
// 发送邮件消息。
func (sm *SendMail) Send(m *Message) error {
	sm.Auth()

	// Build MIME headers / 构建MIME头部
	buffer := bytes.NewBuffer(nil)
	boundary := "GoBoundary"
	Header := make(map[string]string)
	Header["From"] = m.from
	Header["To"] = strings.Join(m.to, ",")
	Header["Cc"] = strings.Join(m.cc, ",")
	Header["Bcc"] = strings.Join(m.bcc, ",")
	Header["Subject"] = mime.QEncoding.Encode("UTF-8", m.subject)
	Header["Content-Type"] = "multipart/mixed;boundary=" + boundary
	Header["Mime-Version"] = "1.0"
	Header["Date"] = time.Now().String()
	// Sanitise every header value to strip CR/LF that would let a
	// malicious config inject extra SMTP commands.
	sanitizeHeaders(Header)
	writeHeader(buffer, Header)

	// Write email body / 写入邮件正文
	bodyBuilder := strings.Builder{}
	bodyBuilder.WriteString("\r\n--")
	bodyBuilder.WriteString(boundary)
	bodyBuilder.WriteString("\r\nContent-Type:")
	bodyBuilder.WriteString(sanitizeHeaderValue(m.contentType))
	bodyBuilder.WriteString("\r\n\r\n")
	bodyBuilder.WriteString(m.body)
	bodyBuilder.WriteString("\r\n")
	buffer.WriteString(bodyBuilder.String())

	// Add attachment if present / 添加附件
	if m.attachment.withFile {
		attachmentBuilder := strings.Builder{}
		attachmentBuilder.WriteString("\r\n--")
		attachmentBuilder.WriteString(boundary)
		attachmentBuilder.WriteString("\r\nContent-Transfer-Encoding:base64\r\n")
		attachmentBuilder.WriteString("Content-Disposition:attachment\r\n")
		attachmentBuilder.WriteString("Content-Type:")
		attachmentBuilder.WriteString(m.attachment.contentType)
		attachmentBuilder.WriteString(";name=\"")
		attachmentBuilder.WriteString(mime.BEncoding.Encode("UTF-8", filepath.Base(m.attachment.name)))
		attachmentBuilder.WriteString("\"\r\n")
		buffer.WriteString(attachmentBuilder.String())
		if err := writeFile(buffer, m.attachment.name); err != nil {
			return fmt.Errorf("failed to read attachment: %w", err)
		}
	}

	// End of MIME message / MIME消息结束
	buffer.WriteString("\r\n--" + boundary + "--")

	// Build recipient list / 构建收件人列表
	rcpt := buildRecipients(m.to, m.cc, m.bcc)

	// Connect to SMTP server with TLS / 使用TLS连接到SMTP服务器
	conn, err := net.DialTimeout("tcp", sm.host+":"+sm.port, sm.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	c, err := smtp.NewClient(conn, sm.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer c.Close()

	// Start TLS encryption / 启动TLS加密
	tlsConfig := &tls.Config{
		InsecureSkipVerify: sm.insecureSkipVerify,
		ServerName:         sm.host,
		// Enforce TLS 1.2 as the minimum protocol. Older versions
		// (SSL 3.0, TLS 1.0/1.1) have known weaknesses and are
		// disabled by Go 1.22+ defaults, but we set the field
		// explicitly to defend against accidental downgrades from
		// intermediate proxies or future Go changes.
		MinVersion: tls.VersionTLS12,
	}
	if err = c.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate with SMTP server / SMTP服务器认证
	if err = c.Auth(sm.auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set sender / 设置发件人
	if err = c.Mail(m.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Add recipients / 添加收件人
	for _, addr := range rcpt {
		if err = c.Rcpt(addr); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr, err)
		}
	}

	// Send email body / 发送邮件内容
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}
	_, err = buffer.WriteTo(wc)
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}
	err = wc.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Send QUIT command and close connection / 发送QUIT命令并关闭连接
	err = c.Quit()
	if err != nil {
		return fmt.Errorf("failed to close SMTP connection: %w", err)
	}

	return nil
}

// sanitizeHeaderValue strips CR/LF characters from a single header
// value so that an attacker controlling the config cannot inject
// additional SMTP commands or headers.
// sanitizeHeaderValue 从单个 header 值中删除 CR/LF 字符，
// 防止控制配置的攻击者注入额外 SMTP 命令或 header。
func sanitizeHeaderValue(v string) string {
	r := strings.NewReplacer("\r", "", "\n", "")
	return r.Replace(v)
}

// sanitizeHeaders applies sanitizeHeaderValue to every header value
// in place. It also trims trailing whitespace, which would otherwise
// cause some SMTP servers to reject the message.
// sanitizeHeaders 对每个 header 值原地应用 sanitizeHeaderValue，
// 同时删除尾部空白以避免部分 SMTP 服务器拒收。
func sanitizeHeaders(h map[string]string) {
	for k, v := range h {
		h[k] = strings.TrimRight(sanitizeHeaderValue(v), " \t")
	}
}

// buildRecipients combines To, CC, and BCC recipients into a single slice.
// 合并To、CC和BCC收件人到一个切片。
func buildRecipients(to, cc, bcc []string) []string {
	if len(to) == 0 && len(cc) == 0 && len(bcc) == 0 {
		return nil
	}

	recipients := make([]string, 0, len(to)+len(cc)+len(bcc))
	recipients = append(recipients, to...)
	recipients = append(recipients, cc...)
	recipients = append(recipients, bcc...)
	return recipients
}

// writeHeader writes MIME header fields to the buffer.
// 将MIME头部字段写入缓冲区。
func writeHeader(buffer *bytes.Buffer, Header map[string]string) {
	headerBuilder := strings.Builder{}
	for key, value := range Header {
		headerBuilder.WriteString(key)
		headerBuilder.WriteString(":")
		headerBuilder.WriteString(value)
		headerBuilder.WriteString("\r\n")
	}
	headerBuilder.WriteString("\r\n")
	buffer.WriteString(headerBuilder.String())
}

// writeFile reads a file and writes its base64-encoded content to the buffer.
// 读取文件并将其base64编码的内容写入缓冲区。
func writeFile(buffer *bytes.Buffer, fileName string) error {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	payload := make([]byte, base64.StdEncoding.EncodedLen(len(file)))
	base64.StdEncoding.Encode(payload, file)
	buffer.WriteString("\r\n")
	for index, line := 0, len(payload); index < line; index++ {
		buffer.WriteByte(payload[index])
		if (index+1)%76 == 0 {
			buffer.WriteString("\r\n")
		}
	}
	return nil
}
